package dto


type LoginResponseDTO struct{
	WebToken string `json:"token"`
	UserRole string `json:"user_role"`
}