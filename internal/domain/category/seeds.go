package category

import pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"

// Fixed UUIDs of default categories seeded by migration
// 20260501162515_create_categories_and_tags.sql. Keep in sync with the
// migration's INSERT INTO categories. Used by usecases/statement/reverse.go
// to auto-assign the "Estorno" category without a DB roundtrip.
const (
	// Default "Estorno" category of type credit; auto-applied to reversals of debit statements.
	SystemCategoryEstornoCreditID pkgvo.ID = "00000000-0000-0000-0000-210000000006"
	// Default "Estorno" category of type debit; auto-applied to reversals of credit statements.
	SystemCategoryEstornoDebitID pkgvo.ID = "00000000-0000-0000-0000-22000000000c"
)
