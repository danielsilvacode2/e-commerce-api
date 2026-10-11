package repository

import (
	"database/sql"
	"e-commerce/adapters/controller/apperrors"
	entities "e-commerce/core/entities/user"
	"errors"

	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
)

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) entities.UserInterface {
	return &userRepository{
		db: db,
	}
}

// CreateUser cria um usuario no banco de dados
func (r *userRepository) CreateUser(user entities.User) error {

	statement, err := r.db.Prepare("INSERT INTO users ( id, user_name, user_email, user_password, user_role) VALUES (?,?,?,?,?)")

	if err != nil {
		return err
	}

	defer statement.Close()

	if _, err := statement.Exec(user.ID, user.Name, user.Email, user.Password, user.Role); err != nil {

		var errSQL *mysql.MySQLError

		if errors.As(err, &errSQL) && errSQL.Number == 1062{
			return  apperrors.ErrConflict
		}
	
		return err
	}

	return nil
}

func (r *userRepository) ReadUser(ID string) (*entities.User, error) {

	row, err := r.db.Query("SELECT * FROM users WHERE id = ?", ID)

	if err != nil {
		return nil, err
	}

	defer row.Close()

	var user entities.User

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, err
		}
		return nil, apperrors.ErrNotFound
	}

	err = row.Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.Role,
		&user.UpdatedAt,
		&user.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) DeleteUser(ID string) error {

	statement, err := r.db.Prepare("DELETE FROM users WHERE id = ?")

	if err != nil {
		return err
	}

	defer statement.Close()

	if _,err = statement.Exec(ID); err != nil{
		return err
	}

	return nil
}
