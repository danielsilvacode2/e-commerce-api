package entities

import (
	"time"
	"github.com/google/uuid"
)

type User struct {
	ID uuid.UUID
	Name string
	Email string
	Password string
	Role string
	UpdatedAt time.Time
	CreatedAt time.Time
}

type UserInterface interface{
	CreateUser(user User) (error)
	ReadUser(ID string) (*User, error)
	DeleteUser(ID string) (error)
}