package statement

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"

	stmtdomain "github.com/DenysonJ/financial-wallet/internal/domain/statement"
	stmtvo "github.com/DenysonJ/financial-wallet/internal/domain/statement/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/statement/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/statement/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/DenysonJ/financial-wallet/pkg/ofx"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
)

// ImportUseCase implements the use case for importing statements from an OFX file.
type ImportUseCase struct {
	repo        interfaces.Repository
	accountRepo interfaces.AccountRepository
}

// NewImportUseCase creates a new ImportUseCase instance.
func NewImportUseCase(repo interfaces.Repository, accountRepo interfaces.AccountRepository) *ImportUseCase {
	return &ImportUseCase{repo: repo, accountRepo: accountRepo}
}

// Execute parses an OFX file and batch-creates statements for the given account.
func (uc *ImportUseCase) Execute(ctx context.Context, input dto.ImportOFXInput) (*dto.ImportOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Statement.Import")
	defer span.End()

	ctx = injectLogContext(ctx, ActionImport)

	// Validate AccountID
	accountID, parseErr := pkgvo.ParseID(input.AccountID)
	if parseErr != nil {
		span.SetStatus(otelcodes.Error, parseErr.Error())
		logutil.LogWarn(ctx, "statement import failed: invalid account ID", "error", parseErr.Error())
		return nil, parseErr
	}

	span.SetAttributes(attribute.String("account.id", input.AccountID))

	// Find account and verify ownership
	account, findErr := uc.accountRepo.FindByID(ctx, accountID)
	if findErr != nil {
		span.SetStatus(otelcodes.Error, findErr.Error())
		logutil.LogWarn(ctx, "statement import failed: account not found", "error", findErr.Error())
		return nil, findErr
	}

	if input.RequestingUserID != "" && account.UserID.String() != input.RequestingUserID {
		span.SetStatus(otelcodes.Error, "forbidden")
		logutil.LogWarn(ctx, "statement import forbidden: not owner", "account.id", accountID.String())
		return nil, stmtdomain.ErrStatementNotFound
	}

	if !account.Active {
		span.SetStatus(otelcodes.Error, "account not active")
		logutil.LogWarn(ctx, "statement import failed: account not active", "account.id", accountID.String())
		return nil, stmtdomain.ErrAccountNotActive
	}

	// Parse OFX file
	parseResult, ofxErr := ofx.Parse(input.FileContent)
	if ofxErr != nil {
		span.SetStatus(otelcodes.Error, ofxErr.Error())
		logutil.LogWarn(ctx, "statement import failed: OFX parse error", "error", ofxErr.Error())
		return nil, ofxErr
	}

	totalTransactions := len(parseResult.Transactions)
	span.SetAttributes(attribute.Int("import.total", totalTransactions))

	// Collect FITIDs for dedup lookup
	fitIDs := make([]string, 0, totalTransactions)
	for _, txn := range parseResult.Transactions {
		if txn.FITID != "" {
			fitIDs = append(fitIDs, txn.FITID)
		}
	}

	// Find existing external IDs for this account
	existingIDs, findIDsErr := uc.repo.FindExternalIDs(ctx, accountID, fitIDs)
	if findIDsErr != nil {
		span.SetStatus(otelcodes.Error, findIDsErr.Error())
		logutil.LogError(ctx, "statement import failed: finding external IDs", "error", findIDsErr.Error())
		return nil, findIDsErr
	}

	// Filter out duplicates and map to domain statements
	var statements []*stmtdomain.Statement
	skipped := 0

	for _, txn := range parseResult.Transactions {
		// Skip duplicates (FITID already exists for this account)
		if txn.FITID != "" && existingIDs[txn.FITID] {
			skipped++
			continue
		}

		// Skip zero-amount transactions (e.g., fee waivers, informational entries)
		if txn.Amount == 0 {
			skipped++
			continue
		}

		stmt, mapErr := mapOFXTransaction(txn, accountID)
		if mapErr != nil {
			span.SetStatus(otelcodes.Error, mapErr.Error())
			logutil.LogWarn(ctx, "statement import failed: mapping transaction", "fitid", txn.FITID, "error", mapErr.Error())
			return nil, mapErr
		}
		statements = append(statements, stmt)
	}

	// Batch create (if any new statements)
	if len(statements) > 0 {
		_, batchErr := uc.repo.CreateBatch(ctx, statements, accountID)
		if batchErr != nil {
			span.SetStatus(otelcodes.Error, batchErr.Error())
			logutil.LogError(ctx, "statement import failed: batch create", "error", batchErr.Error())
			return nil, batchErr
		}
	}

	created := len(statements)
	span.SetAttributes(attribute.Int("import.created", created), attribute.Int("import.skipped", skipped))
	logutil.LogInfo(ctx, "statements imported", "total", totalTransactions, "created", created, "skipped", skipped)

	return &dto.ImportOutput{
		TotalTransactions: totalTransactions,
		Created:           created,
		Skipped:           skipped,
	}, nil
}

// mapOFXTransaction converts an OFX transaction to a domain Statement.
func mapOFXTransaction(txn ofx.Transaction, accountID pkgvo.ID) (*stmtdomain.Statement, error) {
	// Determine type from signed amount
	var stmtType stmtvo.StatementType
	if txn.Amount >= 0 {
		stmtType = stmtvo.TypeCredit
	} else {
		stmtType = stmtvo.TypeDebit
	}

	// Convert to absolute cents for the Amount VO (direct negation avoids float64 precision loss)
	absCents := txn.Amount
	if absCents < 0 {
		absCents = -absCents
	}
	amount, amountErr := stmtvo.NewAmount(absCents)
	if amountErr != nil {
		return nil, amountErr
	}

	// Build description from NAME + MEMO
	description := buildDescription(txn.Name, txn.Memo)

	// Use NewImportedStatement (with external_id) only when FITID is present;
	// otherwise use NewStatement to avoid creating empty-string external_id rows
	if txn.FITID != "" {
		return stmtdomain.NewImportedStatement(accountID, stmtType, amount, description, txn.FITID, txn.DatePosted), nil
	}

	stmt := stmtdomain.NewStatement(accountID, stmtType, amount, description)
	stmt.PostedAt = txn.DatePosted
	return stmt, nil
}

// buildDescription combines OFX NAME and MEMO fields into a single description.
func buildDescription(name, memo string) string {
	name = strings.TrimSpace(name)
	memo = strings.TrimSpace(memo)

	if name != "" && memo != "" {
		return name + " - " + memo
	}
	if name != "" {
		return name
	}
	return memo
}
