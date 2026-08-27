package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	accountvo "github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Testes de borda HTTP do limite de crédito (SPEC-account-001). Cobrem os
// status da tabela de erros da spec: 400 para valor malformado, 422 para a
// combinação tipo × limite.

func int64Ptr(v int64) *int64 { return &v }

func TestAccountHandler_Create_CreditLimit(t *testing.T) {
	userID := pkgvo.NewID()

	tests := []struct {
		name           string
		body           string
		wantStatus     int
		wantMsg        string
		expectRepoCall bool
	}{
		{
			name:           "GIVEN a credit card WHEN created with a valid credit limit THEN 201 (UC-01)",
			body:           `{"name":"Cartao Nubank","type":"credit_card","credit_limit":500000}`,
			wantStatus:     http.StatusCreated,
			expectRepoCall: true,
		},
		{
			name:       "GIVEN a credit card WHEN created without a credit limit THEN 422 (UC-02)",
			body:       `{"name":"Cartao Nubank","type":"credit_card"}`,
			wantStatus: http.StatusUnprocessableEntity,
			wantMsg:    "credit limit is required for credit card accounts",
		},
		{
			name:       "GIVEN a bank account WHEN created with a credit limit THEN 422 (UC-03)",
			body:       `{"name":"Nubank","type":"bank_account","credit_limit":100000}`,
			wantStatus: http.StatusUnprocessableEntity,
			wantMsg:    "credit limit is only allowed for credit card accounts",
		},
		{
			name:       "GIVEN a cash account WHEN created with a zero credit limit THEN 422 by type, not by value (RN-02)",
			body:       `{"name":"Carteira","type":"cash","credit_limit":0}`,
			wantStatus: http.StatusUnprocessableEntity,
			wantMsg:    "credit limit is only allowed for credit card accounts",
		},
		{
			name:       "GIVEN a credit card WHEN the credit limit is zero THEN 400 (UC-04)",
			body:       `{"name":"Cartao","type":"credit_card","credit_limit":0}`,
			wantStatus: http.StatusBadRequest,
			wantMsg:    "credit limit must be greater than zero and at most 1000000000000 cents",
		},
		{
			name:       "GIVEN a credit card WHEN the credit limit is negative THEN 400 (UC-04)",
			body:       `{"name":"Cartao","type":"credit_card","credit_limit":-1}`,
			wantStatus: http.StatusBadRequest,
			wantMsg:    "credit limit must be greater than zero and at most 1000000000000 cents",
		},
		{
			name:       "GIVEN a credit card WHEN the credit limit is above the ceiling THEN 400 (UC-04)",
			body:       `{"name":"Cartao","type":"credit_card","credit_limit":1000000000001}`,
			wantStatus: http.StatusBadRequest,
			wantMsg:    "credit limit must be greater than zero and at most 1000000000000 cents",
		},
		{
			name:       "GIVEN a credit card WHEN the credit limit is fractional THEN 400 without rounding (RN-04)",
			body:       `{"name":"Cartao","type":"credit_card","credit_limit":1500.75}`,
			wantStatus: http.StatusBadRequest,
			wantMsg:    "invalid request body",
		},
		{
			name:       "GIVEN a credit card WHEN the credit limit is not a number THEN 400 (RN-04)",
			body:       `{"name":"Cartao","type":"credit_card","credit_limit":"500000"}`,
			wantStatus: http.StatusBadRequest,
			wantMsg:    "invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockRepo := newAccountHandler(t)
			if tt.expectRepoCall {
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*account.Account")).Return(nil)
			}
			r := setupAccountRouterWithAuth(h, userID.String())

			req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantMsg != "" {
				var errBody map[string]map[string]string
				assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &errBody))
				assert.Equal(t, tt.wantMsg, errBody["errors"]["message"])
			}
			if !tt.expectRepoCall {
				mockRepo.AssertNotCalled(t, "Create")
			}
		})
	}
}

func TestAccountHandler_Update_CreditLimit(t *testing.T) {
	userID := pkgvo.NewID()
	accountID := pkgvo.NewID()

	creditCard := func() *accountdomain.Account {
		limit := accountvo.ParseCreditLimit(500000)
		return &accountdomain.Account{
			ID: accountID, UserID: userID, Name: "Cartao", Type: accountvo.TypeCreditCard,
			Balance: -300000, Active: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
			CreditLimit: &limit,
		}
	}
	bankAccount := func() *accountdomain.Account {
		return &accountdomain.Account{
			ID: accountID, UserID: userID, Name: "Nubank", Type: accountvo.TypeBankAccount,
			Active: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
	}

	tests := []struct {
		name         string
		body         string
		existing     func() *accountdomain.Account
		wantStatus   int
		wantMsg      string
		wantLimit    *int64
		expectUpdate bool
	}{
		{
			name:         "GIVEN a credit card WHEN updating only the credit limit THEN 200 (UC-05)",
			body:         `{"credit_limit":800000}`,
			existing:     creditCard,
			wantStatus:   http.StatusOK,
			wantLimit:    int64Ptr(800000),
			expectUpdate: true,
		},
		{
			name:         "GIVEN a credit card with an open invoice WHEN reducing the limit below the amount used THEN 200 (UC-07)",
			body:         `{"credit_limit":100000}`,
			existing:     creditCard,
			wantStatus:   http.StatusOK,
			wantLimit:    int64Ptr(100000),
			expectUpdate: true,
		},
		{
			name:         "GIVEN a credit card WHEN only the name is sent THEN the limit is preserved (UC-08)",
			body:         `{"name":"Cartao Nubank"}`,
			existing:     creditCard,
			wantStatus:   http.StatusOK,
			wantLimit:    int64Ptr(500000),
			expectUpdate: true,
		},
		{
			name:         "GIVEN a credit card WHEN credit_limit is explicitly null THEN the limit is preserved (UC-08)",
			body:         `{"credit_limit":null}`,
			existing:     creditCard,
			wantStatus:   http.StatusOK,
			wantLimit:    int64Ptr(500000),
			expectUpdate: true,
		},
		{
			name:         "GIVEN an account without a credit limit WHEN reading the update response THEN credit_limit is null (UC-10)",
			body:         `{"name":"Nubank Conta"}`,
			existing:     bankAccount,
			wantStatus:   http.StatusOK,
			wantLimit:    nil,
			expectUpdate: true,
		},
		{
			name:       "GIVEN an account that does not accept a limit WHEN setting one THEN 422 (UC-06)",
			body:       `{"credit_limit":50000}`,
			existing:   bankAccount,
			wantStatus: http.StatusUnprocessableEntity,
			wantMsg:    "credit limit is only allowed for credit card accounts",
		},
		{
			name:       "GIVEN a credit card WHEN setting a zero credit limit THEN 400 (UC-04)",
			body:       `{"credit_limit":0}`,
			existing:   creditCard,
			wantStatus: http.StatusBadRequest,
			wantMsg:    "credit limit must be greater than zero and at most 1000000000000 cents",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockRepo := newAccountHandler(t)
			mockRepo.On("FindByID", mock.Anything, mock.AnythingOfType("vo.ID")).Return(tt.existing(), nil)
			if tt.expectUpdate {
				mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*account.Account")).Return(nil)
			}
			r := setupAccountRouterWithAuth(h, userID.String())

			req := httptest.NewRequest(http.MethodPut, "/accounts/"+accountID.String(), bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.wantMsg != "" {
				var errBody map[string]map[string]string
				assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &errBody))
				assert.Equal(t, tt.wantMsg, errBody["errors"]["message"])
				mockRepo.AssertNotCalled(t, "Update")
				return
			}

			var body struct {
				Data struct {
					CreditLimit *int64 `json:"credit_limit"`
				} `json:"data"`
			}
			assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			if tt.wantLimit == nil {
				assert.Nil(t, body.Data.CreditLimit)
			} else {
				assert.NotNil(t, body.Data.CreditLimit)
				assert.Equal(t, *tt.wantLimit, *body.Data.CreditLimit)
			}
		})
	}
}

// TestAccountHandler_Update_IgnoresType: o update não altera o tipo
// da account, que é o que sustenta a invariante tipo × limite sem revalidação
// cruzada. Enviar "type" no corpo é silenciosamente ignorado.
func TestAccountHandler_Update_IgnoresType(t *testing.T) {
	userID := pkgvo.NewID()
	accountID := pkgvo.NewID()

	h, mockRepo := newAccountHandler(t)
	existing := &accountdomain.Account{
		ID: accountID, UserID: userID, Name: "Carteira", Type: accountvo.TypeCash,
		Active: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	mockRepo.On("FindByID", mock.Anything, mock.AnythingOfType("vo.ID")).Return(existing, nil)
	mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*account.Account")).Return(nil)
	r := setupAccountRouterWithAuth(h, userID.String())

	req := httptest.NewRequest(http.MethodPut, "/accounts/"+accountID.String(),
		bytes.NewBufferString(`{"type":"credit_card","name":"Virou cartao?"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, accountvo.TypeCash, existing.Type, "o tipo da account nunca muda no update")
	assert.Contains(t, w.Body.String(), `"type":"cash"`)
	assert.Contains(t, w.Body.String(), `"credit_limit":null`)
}

// TestAccountHandler_GetByID_ExposesCreditLimit: o campo sai
// como número inteiro de centavos, e como null quando a account não tem limite.
func TestAccountHandler_GetByID_ExposesCreditLimit(t *testing.T) {
	userID := pkgvo.NewID()
	accountID := pkgvo.NewID()

	t.Run("GIVEN a credit card WHEN reading THEN credit_limit is an integer in cents (RN-15)", func(t *testing.T) {
		h, mockRepo := newAccountHandler(t)
		limit := accountvo.ParseCreditLimit(500000)
		mockRepo.On("FindByID", mock.Anything, mock.AnythingOfType("vo.ID")).Return(&accountdomain.Account{
			ID: accountID, UserID: userID, Name: "Cartao", Type: accountvo.TypeCreditCard,
			Balance: -300000, Active: true, CreatedAt: time.Now(), UpdatedAt: time.Now(), CreditLimit: &limit,
		}, nil)
		r := setupAccountRouterWithAuth(h, userID.String())

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/accounts/"+accountID.String(), nil))

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"credit_limit":500000`)
	})

	t.Run("GIVEN an account without a limit WHEN reading THEN credit_limit is null (RN-14)", func(t *testing.T) {
		h, mockRepo := newAccountHandler(t)
		mockRepo.On("FindByID", mock.Anything, mock.AnythingOfType("vo.ID")).Return(&accountdomain.Account{
			ID: accountID, UserID: userID, Name: "Carteira", Type: accountvo.TypeCash,
			Active: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}, nil)
		r := setupAccountRouterWithAuth(h, userID.String())

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/accounts/"+accountID.String(), nil))

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"credit_limit":null`)
	})
}
