package auth

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	"github.com/DenysonJ/financial-wallet/internal/usecases/auth/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/auth/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/DenysonJ/financial-wallet/pkg/telemetry"
)

// RefreshUseCase implementa o caso de uso de refresh de token.
type RefreshUseCase struct {
	token interfaces.TokenService
}

// NewRefreshUseCase cria uma nova instância do RefreshUseCase.
func NewRefreshUseCase(token interfaces.TokenService) *RefreshUseCase {
	return &RefreshUseCase{
		token: token,
	}
}

// Execute valida um refresh token e gera um novo par de tokens.
//
// Fluxo:
//  1. Validar refresh token (assinatura, expiração)
//  2. Verificar que o tipo é "refresh"
//  3. Gerar novo access token e refresh token
func (uc *RefreshUseCase) Execute(ctx context.Context, input dto.RefreshInput) (*dto.RefreshOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Auth.Refresh")
	defer span.End()

	ctx = injectLogContext(ctx, "refresh")

	claims, validateErr := uc.token.ValidateToken(input.RefreshToken)
	if validateErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_credentials"))
		logutil.LogWarn(ctx, "token refresh failed")
		return nil, userdomain.ErrInvalidCredentials
	}

	if claims.TokenType != interfaces.TokenTypeRefresh {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_credentials"))
		logutil.LogWarn(ctx, "token refresh failed")
		return nil, userdomain.ErrInvalidCredentials
	}

	accessToken, accessErr := uc.token.GenerateAccessToken(claims.UserID)
	if accessErr != nil {
		telemetry.FailSpan(span, accessErr, "token refresh failed: token generation")
		logutil.LogError(ctx, "token refresh failed: token generation error", "error", accessErr.Error())
		return nil, accessErr
	}

	refreshToken, refreshErr := uc.token.GenerateRefreshToken(claims.UserID)
	if refreshErr != nil {
		telemetry.FailSpan(span, refreshErr, "token refresh failed: token generation")
		logutil.LogError(ctx, "token refresh failed: token generation error", "error", refreshErr.Error())
		return nil, refreshErr
	}

	telemetry.OkSpan(span)
	logutil.LogInfo(ctx, "token refreshed")

	return &dto.RefreshOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
