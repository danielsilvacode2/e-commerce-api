package dto

type UserConfirmResetPasswordDTO struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	NewPassword string `json:"newpass"`
}
