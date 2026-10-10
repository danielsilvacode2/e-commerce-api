package security

import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
	
	hashByte, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		return "", err
	}

	return string(hashByte), nil
}

func ComparePassword(hashPassword, password string) error{
	return bcrypt.CompareHashAndPassword([]byte(hashPassword), []byte(password))
}