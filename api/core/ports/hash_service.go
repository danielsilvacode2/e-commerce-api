package ports

type HashService interface {
	CompareHash(hash, text string)error
	HashGenerate(password string) (string, error)
}
