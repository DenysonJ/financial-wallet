package auth

import (
	"context"
	"crypto/rand"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/crypto/bcrypt"

	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/auth/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/auth/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/DenysonJ/financial-wallet/pkg/telemetry"
)

// LoginUseCase implementa o caso de uso de login.
type LoginUseCase struct {
	repo       interfaces.UserRepository
	token      interfaces.TokenService
	bcryptCost int
	// dummyHash equalizes bcrypt CPU cost on failure branches to prevent
	// timing-based email enumeration.
	dummyHash []byte
}

// NewLoginUseCase cria uma nova instância do LoginUseCase.
func NewLoginUseCase(repo interfaces.UserRepository, token interfaces.TokenService) *LoginUseCase {
	uc := &LoginUseCase{
		repo:       repo,
		token:      token,
		bcryptCost: vo.DefaultBcryptCost,
	}
	uc.dummyHash = generateDummyHash(uc.bcryptCost)
	return uc
}

// WithBcryptCost sets the bcrypt cost (builder pattern). Must match the cost
// used to hash real passwords or the dummy compare won't equalize timing.
func (uc *LoginUseCase) WithBcryptCost(cost int) *LoginUseCase {
	uc.bcryptCost = cost
	uc.dummyHash = generateDummyHash(cost)
	return uc
}

// generateDummyHash builds a bcrypt hash over an unrecoverable random secret.
func generateDummyHash(cost int) []byte {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = vo.DefaultBcryptCost
	}
	secret := make([]byte, 32)
	if _, randErr := rand.Read(secret); randErr != nil {
		secret = []byte("login-dummy-secret-fallback-32by")
	}
	hash, hashErr := bcrypt.GenerateFromPassword(secret, cost)
	if hashErr != nil {
		hash, _ = bcrypt.GenerateFromPassword(secret, vo.DefaultBcryptCost)
	}
	return hash
}

// Execute autentica um usuário por email e senha, retornando tokens JWT.
//
// Fluxo:
//  1. Buscar usuário por email
//  2. Verificar se o usuário está ativo
//  3. Verificar se possui senha cadastrada
//  4. Verificar senha contra hash armazenado
//  5. Gerar access token e refresh token
//
// Retorna ErrInvalidCredentials para qualquer falha (sem revelar causa específica).
func (uc *LoginUseCase) Execute(ctx context.Context, input dto.LoginInput) (*dto.LoginOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Auth.Login")
	defer span.End()

	ctx = injectLogContext(ctx, "login")

	emailVO, emailErr := vo.NewEmail(input.Email)
	if emailErr != nil {
		uc.equalizeTiming(input.Password)
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_credentials"))
		logutil.LogWarn(ctx, "login failed")
		return nil, userdomain.ErrInvalidCredentials
	}

	// Explicit IsExpected branch (not ClassifyError): the expected arm must
	// not log error details to avoid a credential oracle via logs.
	e, findErr := uc.repo.FindByEmail(ctx, emailVO)
	if findErr != nil {
		uc.equalizeTiming(input.Password)
		if telemetry.IsExpected(findErr) {
			telemetry.WarnSpan(span, attribute.String("app.result", "invalid_credentials"))
			logutil.LogWarn(ctx, "login failed")
		} else {
			telemetry.FailSpan(span, findErr, "login: repository error")
			logutil.LogError(ctx, "login failed: unexpected", "error", findErr.Error())
		}
		return nil, userdomain.ErrInvalidCredentials
	}

	if !e.Active {
		uc.equalizeTiming(input.Password)
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_credentials"))
		logutil.LogWarn(ctx, "login failed")
		return nil, userdomain.ErrInvalidCredentials
	}

	if e.PasswordHash == "" {
		uc.equalizeTiming(input.Password)
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_credentials"))
		logutil.LogWarn(ctx, "login failed")
		return nil, userdomain.ErrInvalidCredentials
	}

	checkErr := vo.CheckPassword(e.PasswordHash, input.Password)
	if checkErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_credentials"))
		logutil.LogWarn(ctx, "login failed")
		return nil, userdomain.ErrInvalidCredentials
	}

	accessToken, accessErr := uc.token.GenerateAccessToken(e.ID.String())
	if accessErr != nil {
		telemetry.FailSpan(span, accessErr, "login failed: token generation")
		logutil.LogError(ctx, "login failed: token generation error", "error", accessErr.Error())
		return nil, accessErr
	}

	refreshToken, refreshErr := uc.token.GenerateRefreshToken(e.ID.String())
	if refreshErr != nil {
		telemetry.FailSpan(span, refreshErr, "login failed: token generation")
		logutil.LogError(ctx, "login failed: token generation error", "error", refreshErr.Error())
		return nil, refreshErr
	}

	telemetry.OkSpan(span)
	logutil.LogInfo(ctx, "login successful")

	return &dto.LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// equalizeTiming runs a bcrypt compare against the dummy hash so login failure
// branches consume CPU comparable to a real password verification
func (uc *LoginUseCase) equalizeTiming(password string) {
	_ = bcrypt.CompareHashAndPassword(uc.dummyHash, []byte(password))
}
