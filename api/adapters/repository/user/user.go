package repository

import (
	"database/sql"
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/user"
	"errors"

	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
)

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) entities.UserRepository {
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

		if errors.As(err, &errSQL) && errSQL.Number == 1062 {
			return apperrors.ErrConflict
		}

		return err
	}

	return nil
}

// ReadUser retorna um usuario com base no ID
func (r *userRepository) ReadUserById(ID string) (*entities.User, error) {

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

func (r *userRepository) ReadUserByEmail(email string) (*entities.User, error) {

	row, err := r.db.Query("SELECT * FROM users WHERE user_email = ?", email)

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

// ReadAllUsers retorna todos os usuarios da api
func (r *userRepository) ReadAllUsers() ([]entities.User, error) {

	rows, err := r.db.Query("SELECT * FROM users")

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var listUsers []entities.User

	for rows.Next() {
		var user entities.User

		if err = rows.Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.Password,
			&user.Role,
			&user.UpdatedAt,
			&user.CreatedAt,
		); err != nil {
			return nil, err
		}

		if err = rows.Err(); err != nil {
			return nil, err
		}

		listUsers = append(listUsers, user)

	}

	return listUsers, nil
}

// DeleteUser deleta um usuario pelo ID
func (r *userRepository) DeleteUserById(ID string) error {

	statement, err := r.db.Prepare("DELETE FROM users WHERE id = ?")

	if err != nil {
		return err
	}

	defer statement.Close()

	if _, err = statement.Exec(ID); err != nil {
		return err
	}

	return nil
}

func (r *userRepository) UpdateUser(user entities.User, ID string) error {

	statement, err := r.db.Prepare("UPDATE users SET user_name = ? , user_email = ? WHERE id = ?")

	if err != nil {
		return err
	}

	defer statement.Close()

	_, err = statement.Exec(user.Name, user.Email, ID)

	if err != nil {
		var errSQL *mysql.MySQLError

		if errors.As(err, &errSQL) && errSQL.Number == 1062 {
			return apperrors.ErrConflict
		}

		return err

	}

	return nil
}


