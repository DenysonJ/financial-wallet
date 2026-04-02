package auth

import (
	"context"

	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/auth/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/auth/interfaces"
)

// LoginUseCase implementa o caso de uso de login.
type LoginUseCase struct {
	repo  interfaces.UserRepository
	token interfaces.TokenService
}

// NewLoginUseCase cria uma nova instância do LoginUseCase.
func NewLoginUseCase(repo interfaces.UserRepository, token interfaces.TokenService) *LoginUseCase {
	return &LoginUseCase{
		repo:  repo,
		token: token,
	}
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
	emailVO, emailErr := vo.NewEmail(input.Email)
	if emailErr != nil {
		return nil, userdomain.ErrInvalidCredentials
	}

	e, findErr := uc.repo.FindByEmail(ctx, emailVO)
	if findErr != nil {
		return nil, userdomain.ErrInvalidCredentials
	}

	if !e.Active {
		return nil, userdomain.ErrInvalidCredentials
	}

	if e.PasswordHash == "" {
		return nil, userdomain.ErrInvalidCredentials
	}

	checkErr := vo.CheckPassword(e.PasswordHash, input.Password)
	if checkErr != nil {
		return nil, userdomain.ErrInvalidCredentials
	}

	accessToken, accessErr := uc.token.GenerateAccessToken(e.ID.String())
	if accessErr != nil {
		return nil, accessErr
	}

	refreshToken, refreshErr := uc.token.GenerateRefreshToken(e.ID.String())
	if refreshErr != nil {
		return nil, refreshErr
	}

	return &dto.LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
