package dto

import "time"

type RabbitMQUserDto struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Email            string    `json:"email"`
	Password         string    `json:"password"`
	Role             string    `json:"role"`
	VerificationCode string    `json:"verificationCode"`
	IsVerified       bool      `json:"isVerified"`
	CodeExpires      time.Time `json:"codeExpires"`
	UpdatedAt        time.Time `json:"updateAt"`
	CreatedAt        time.Time `json:"createdAt"`
}
