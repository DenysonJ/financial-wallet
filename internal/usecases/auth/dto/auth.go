package dto

// LoginInput representa os dados de entrada para login.
type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`    // Email do usuário
	Password string `json:"password" binding:"required"`       // Senha em texto plano
}

// LoginOutput representa os dados de saída após login bem-sucedido.
type LoginOutput struct {
	AccessToken  string `json:"access_token"`  // JWT access token (curta duração)
	RefreshToken string `json:"refresh_token"` // JWT refresh token (longa duração)
}

// RefreshInput representa os dados de entrada para refresh de token.
type RefreshInput struct {
	RefreshToken string `json:"refresh_token" binding:"required"` // Refresh token atual
}

// RefreshOutput representa os dados de saída após refresh bem-sucedido.
type RefreshOutput struct {
	AccessToken  string `json:"access_token"`  // Novo JWT access token
	RefreshToken string `json:"refresh_token"` // Novo JWT refresh token
}
