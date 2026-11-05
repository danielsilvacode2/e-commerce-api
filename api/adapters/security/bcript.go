package security

import "golang.org/x/crypto/bcrypt"

type BcriptPasswordService struct{}

func NewPasswordBcript()  BcriptPasswordService{
	return  BcriptPasswordService{}
}

func (p  BcriptPasswordService) HashGenerate(password string) (string, error) {
	
	hashByte, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		return "", err
	}

	return string(hashByte), nil
}

func (p  BcriptPasswordService) CompareHash(hash, text string) error{
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(text))
}