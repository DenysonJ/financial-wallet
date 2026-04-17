package ofx

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// Parse reads an OFX file and extracts all bank statement transactions.
// Supports both SGML (v1.x) and XML (v2.x) formats.
func Parse(r io.Reader) (*ParseResult, error) {
	data, readErr := io.ReadAll(r)
	if readErr != nil {
		return nil, fmt.Errorf("%w: reading input: %s", ErrInvalidFormat, readErr.Error())
	}

	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("%w: empty input", ErrInvalidFormat)
	}

	header, xmlBody, splitErr := splitHeaderAndBody(data)
	if splitErr != nil {
		return nil, splitErr
	}

	if isSGML(header) {
		xmlBody = sgmlToXML(xmlBody)
	}

	transactions, parseErr := parseXMLBody(xmlBody)
	if parseErr != nil {
		return nil, parseErr
	}

	if len(transactions) == 0 {
		return nil, ErrNoTransactions
	}

	return &ParseResult{
		Header:       header,
		Transactions: transactions,
	}, nil
}

// splitHeaderAndBody separates the OFX header from the XML/SGML body.
// The body starts at the first occurrence of "<OFX>" (case-insensitive).
func splitHeaderAndBody(data []byte) (Header, []byte, error) {
	content := string(data)
	upperContent := strings.ToUpper(content)

	ofxIdx := strings.Index(upperContent, "<OFX>")
	if ofxIdx == -1 {
		return Header{}, nil, fmt.Errorf("%w: missing <OFX> tag", ErrInvalidFormat)
	}

	headerStr := content[:ofxIdx]
	body := []byte(content[ofxIdx:])

	header := parseHeader(headerStr)
	return header, body, nil
}

// parseHeader extracts key-value pairs from the OFX header block.
func parseHeader(s string) Header {
	h := Header{}
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "<?") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch strings.ToUpper(key) {
		case "VERSION":
			h.Version = value
		case "ENCODING":
			h.Encoding = value
		}
	}

	// For XML-based OFX (v2), try to parse version from <?OFX ... ?> processing instruction
	if h.Version == "" {
		if idx := strings.Index(s, "<?OFX"); idx != -1 {
			piEnd := strings.Index(s[idx:], "?>")
			if piEnd != -1 {
				pi := s[idx : idx+piEnd+2]
				h.Version = extractAttr(pi, "VERSION")
			}
		}
	}

	return h
}

// extractAttr extracts a named attribute value from a processing instruction string.
func extractAttr(pi, name string) string {
	upper := strings.ToUpper(pi)
	nameUpper := strings.ToUpper(name)

	idx := strings.Index(upper, nameUpper+"=")
	if idx == -1 {
		return ""
	}

	rest := pi[idx+len(name)+1:]
	rest = strings.TrimLeft(rest, "\"' ")

	endIdx := strings.IndexAny(rest, "\"' ?>")
	if endIdx == -1 {
		return rest
	}
	return rest[:endIdx]
}

// isSGML returns true if the header indicates SGML-based OFX (v1.x).
func isSGML(h Header) bool {
	// Version 200+ is XML-based
	if h.Version != "" && len(h.Version) >= 3 && h.Version[0] >= '2' {
		return false
	}
	// If version starts with "1" or is legacy format, it's SGML
	return true
}

// sgmlToXML converts OFX SGML to valid XML by adding closing tags
// for leaf elements that don't have them.
func sgmlToXML(data []byte) []byte {
	var buf bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		buf.WriteString(processLine(line))
		buf.WriteByte('\n')
	}

	return buf.Bytes()
}

// processLine handles a single line of SGML OFX, adding closing tags where needed.
func processLine(line string) string {
	// If the line is a closing tag, pass through: </TAG>
	if strings.HasPrefix(line, "</") {
		return line
	}

	// If the line starts with < but has content after the tag name,
	// it's a leaf element like: <TRNAMT>-150.75
	if !strings.HasPrefix(line, "<") {
		return line
	}

	// Find end of opening tag
	tagEnd := strings.Index(line, ">")
	if tagEnd == -1 {
		return line
	}

	tagName := line[1:tagEnd]
	valueAfter := strings.TrimSpace(line[tagEnd+1:])

	// If there's a value after the tag (leaf element), add closing tag
	if valueAfter != "" && !strings.HasPrefix(valueAfter, "<") {
		return fmt.Sprintf("<%s>%s</%s>", tagName, valueAfter, tagName)
	}

	return line
}

// XML structures for parsing the OFX body

type xmlOFX struct {
	BankMsgs xmlBankMsgsRsV1 `xml:"BANKMSGSRSV1"`
}

type xmlBankMsgsRsV1 struct {
	StmtTrnRs xmlStmtTrnRs `xml:"STMTTRNRS"`
}

type xmlStmtTrnRs struct {
	StmtRs xmlStmtRs `xml:"STMTRS"`
}

type xmlStmtRs struct {
	BankTranList xmlBankTranList `xml:"BANKTRANLIST"`
}

type xmlBankTranList struct {
	Transactions []xmlStmtTrn `xml:"STMTTRN"`
}

type xmlStmtTrn struct {
	TrnType  string `xml:"TRNTYPE"`
	DtPosted string `xml:"DTPOSTED"`
	TrnAmt   string `xml:"TRNAMT"`
	FITID    string `xml:"FITID"`
	Name     string `xml:"NAME"`
	Memo     string `xml:"MEMO"`
}

// parseXMLBody decodes the XML body and extracts transactions.
func parseXMLBody(data []byte) ([]Transaction, error) {
	var ofxDoc xmlOFX
	decodeErr := xml.Unmarshal(data, &ofxDoc)
	if decodeErr != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidFormat, decodeErr.Error())
	}

	rawTxns := ofxDoc.BankMsgs.StmtTrnRs.StmtRs.BankTranList.Transactions
	if len(rawTxns) == 0 {
		return nil, nil
	}

	transactions := make([]Transaction, 0, len(rawTxns))
	for _, raw := range rawTxns {
		txn, mapErr := mapTransaction(raw)
		if mapErr != nil {
			return nil, mapErr
		}
		transactions = append(transactions, txn)
	}

	return transactions, nil
}

// mapTransaction converts a raw XML transaction to the domain Transaction type.
func mapTransaction(raw xmlStmtTrn) (Transaction, error) {
	amount, amountErr := ParseAmount(strings.TrimSpace(raw.TrnAmt))
	if amountErr != nil {
		return Transaction{}, fmt.Errorf("parsing amount for FITID %s: %w", raw.FITID, amountErr)
	}

	datePosted, dateErr := ParseDate(strings.TrimSpace(raw.DtPosted))
	if dateErr != nil {
		return Transaction{}, fmt.Errorf("parsing date for FITID %s: %w", raw.FITID, dateErr)
	}

	return Transaction{
		FITID:      strings.TrimSpace(raw.FITID),
		Type:       strings.TrimSpace(raw.TrnType),
		Amount:     amount,
		Name:       strings.TrimSpace(raw.Name),
		Memo:       strings.TrimSpace(raw.Memo),
		DatePosted: datePosted,
	}, nil
}
