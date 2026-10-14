package ports


type PasswordService interface{
 ComparePassword(string, string) error
 HashPassword(password string) (string, error)
}
