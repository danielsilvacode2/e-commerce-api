package repository

import (
	"database/sql"
	entities "e-commerce/core/entities/user"
)

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) entities.UserInterface {
	return &userRepository{
		db: db,
	}
}

//CreateUser cria um usuario no banco de dados
func (r *userRepository) CreateUser(user entities.User) error {

	statement, err := r.db.Prepare("INSERT INTO users ( id, user_name, user_email, user_password, user_role) VALUES (?,?,?,?,?)")

	if err != nil {
		return err
	}

	defer statement.Close()

	if _, err := statement.Exec(user.ID, user.Name, user.Email, user.Password, user.Role); err != nil {
		return err
	}

	return nil
}

func (r *userRepository) ReadUser(ID string) (*entities.User, error){

	row, err := r.db.Query("SELECT * FROM users WHERE id = ?", ID)

	if err != nil {
		return nil, err
	}

	defer row.Close()
	
	var user entities.User

	row.Next()

	err = row.Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.Role,
		&user.UpdatedAt,
		&user.CreatedAt,
	)

	if err != nil{
		return nil,err
	}

	return &user, nil
}

func (r *userRepository) DeleteUser(ID string) (error){
	return nil
}