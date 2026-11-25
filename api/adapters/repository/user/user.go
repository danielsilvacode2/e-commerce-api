package repository

import (
	"database/sql"
	"database/sql/driver"
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/user"
	"errors"
	"net"

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

func (r *userRepository) isUnavailable(err error) bool {
	if errors.Is(err, driver.ErrBadConn) || errors.Is(err, mysql.ErrInvalidConn) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var myErr *mysql.MySQLError
	if errors.As(err, &myErr) {
		switch myErr.Number {
		case 1040, 1053, 1205, 1213:
			return true
		}
	}
	return false
}

// CreateUser cria um usuario no banco de dados
func (r *userRepository) CreateUser(user entities.User) error {

	if _, err := r.db.Exec(
		"INSERT INTO users"+
			" ( id, user_name, user_email, user_password, user_role, verification_code, is_verified,  code_expires_at)"+
			" VALUES (?,?,?,?,?,?,?,?)",
		user.ID,
		user.Name,
		user.Email,
		user.Password,
		user.Role,
		user.VerificationCode,
		user.IsVerified,
		user.CodeExpires,
	); err != nil {
		if r.isUnavailable(err) {
			return apperrors.ErrDatabaseUnavailable
		}

		var errSQL *mysql.MySQLError

		if errors.As(err, &errSQL) {
			if errSQL.Number == 1062 {
				return apperrors.ErrConflict
			}
			return apperrors.ErrDatabaseUnavailable
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
		&user.VerificationCode,
		&user.CodeExpires,
		&user.PasswordResetCode,
		&user.PasswordResetExpires,
		&user.IsVerified,
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
		&user.VerificationCode,
		&user.CodeExpires,
		&user.PasswordResetCode,
		&user.PasswordResetExpires,
		&user.IsVerified,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// ReadAllUsers retorna todos os usuarios da api sem carregar hashes de senha ou codigos sensiveis
func (r *userRepository) ReadAllUsers() ([]entities.User, error) {

	rows, err := r.db.Query("SELECT id, user_name, user_email, user_role, updated_at, created_at, is_verified FROM users ORDER BY created_at DESC")

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
			&user.Role,
			&user.UpdatedAt,
			&user.CreatedAt,
			&user.IsVerified,
		); err != nil {
			return nil, err
		}

		listUsers = append(listUsers, user)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return listUsers, nil
}

// DeleteUser deleta um usuario pelo ID
func (r *userRepository) DeleteUserById(ID string) error {

	if _, err := r.db.Exec("DELETE FROM users WHERE id = ?", ID); err != nil {
		return err
	}

	return nil
}

func (r *userRepository) UpdateUser(user entities.User, ID string) error {

	_, err := r.db.Exec("UPDATE users SET "+
		" user_name = ? ,"+
		" user_email = ? ,"+
		" verification_code = ? ,"+
		" code_expires_at = ? ,"+
		" is_verified = ? ,"+
		" password_reset_code = ?,"+
		" password_reset_expires_at = ? "+
		" WHERE id = ?",
		user.Name,
		user.Email,
		user.VerificationCode,
		user.CodeExpires,
		user.IsVerified,
		user.PasswordResetCode,
		user.PasswordResetExpires,
		ID)

	if err != nil {
		var errSQL *mysql.MySQLError

		if errors.As(err, &errSQL) && errSQL.Number == 1062 {
			return apperrors.ErrConflict
		}

		return err

	}

	return nil
}

func (r *userRepository) UpdatePasswordUser(user entities.User, ID string) error {

	_, err := r.db.Exec("UPDATE users SET "+
		" user_password = ?,"+
		" password_reset_code = ?,"+
		" password_reset_expires_at = ? "+
		" WHERE id = ?",
		user.Password,
		user.PasswordResetCode,
		user.PasswordResetExpires,
		ID)

	if err != nil {
		var errSQL *mysql.MySQLError

		if errors.As(err, &errSQL) && errSQL.Number == 1062 {
			return apperrors.ErrConflict
		}

		return err

	}

	return nil
}
