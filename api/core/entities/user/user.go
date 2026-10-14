package entities

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID
	Name      string
	Email     string
	Password  string
	Role      string
	UpdatedAt time.Time
	CreatedAt time.Time
}

func (u *User) NormalizeData() {

	u.Name = strings.TrimSpace(u.Name)
	u.Email = strings.TrimSpace(u.Email)
	u.Email = strings.ToLower(u.Email)

}

type UserRepository interface {
	CreateUser(user User) error
	ReadUserById(ID string) (*User, error)
	DeleteUserById(ID string) error
	ReadAllUsers() ([]User, error)
	ReadUserByEmail(email string) (*User, error)
}
