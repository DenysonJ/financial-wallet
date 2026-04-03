package account

// ListFilter contém os parâmetros de filtragem e paginação para listagem de accounts.
type ListFilter struct {
	Page       int
	Limit      int
	UserID     string
	Name       string
	Type       string
	ActiveOnly bool
}

// Normalize aplica valores padrão aos parâmetros de paginação.
func (f *ListFilter) Normalize() {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 {
		f.Limit = 10
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
}

// Offset calcula o offset para a query SQL.
func (f *ListFilter) Offset() int {
	return (f.Page - 1) * f.Limit
}

// ListResult contém o resultado paginado de accounts.
type ListResult struct {
	Accounts []*Account
	Total    int
	Page     int
	Limit    int
}
