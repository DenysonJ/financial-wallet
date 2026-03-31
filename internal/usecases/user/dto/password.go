package dto

// SetPasswordInput representa os dados de entrada para cadastrar senha.
type SetPasswordInput struct {
	UserID               string `json:"-"`                                          // ID vem da URL ou contexto JWT
	Password             string `json:"password" binding:"required"`                // Senha em texto plano
	PasswordConfirmation string `json:"password_confirmation" binding:"required"`   // Confirmação da senha
}

// ChangePasswordInput representa os dados de entrada para alterar senha.
type ChangePasswordInput struct {
	UserID                  string `json:"-"`                                             // ID vem do contexto JWT
	CurrentPassword         string `json:"current_password" binding:"required"`           // Senha atual
	NewPassword             string `json:"new_password" binding:"required"`               // Nova senha
	NewPasswordConfirmation string `json:"new_password_confirmation" binding:"required"`  // Confirmação da nova senha
}
