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

type UserInterface interface {
	CreateUser(user User) error
	ReadUser(ID string) (*User, error)
	DeleteUser(ID string) error
}
