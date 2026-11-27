package repository

import (
	"database/sql"
	"e-commerce/core/apperrors"
	entitiesAddress "e-commerce/core/entities/address"
	"errors"
	"strings"

	"github.com/go-sql-driver/mysql"
)

type addressRepository struct {
	db *sql.DB
}

func NewAddressRepository(db *sql.DB) entitiesAddress.AddressRepository {
	return &addressRepository{
		db: db,
	}
}

func nullableAddressString(val string) sql.NullString {
	trimmed := strings.TrimSpace(val)
	if trimmed == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: trimmed, Valid: true}
}

func (r *addressRepository) CreateAddress(address entitiesAddress.Address) error {
	if _, err := r.db.Exec(`
		INSERT INTO addresses (id, user_id, cep, street, number, complement, neighborhood, city, state)
		VALUES (?,?,?,?,?,?,?,?,?)
	`,
		address.Id,
		address.UserId,
		address.Cep,
		address.Street,
		address.Number,
		nullableAddressString(address.Complement),
		nullableAddressString(address.Neighborhood),
		address.City,
		address.State,
	); err != nil {
		var sqlErr *mysql.MySQLError
		if errors.As(err, &sqlErr) && sqlErr.Number == 1062 {
			return apperrors.ErrConflict
		}
		return err
	}

	return nil
}

func (r *addressRepository) ReadAddressById(ID string) (*entitiesAddress.Address, error) {
	row, err := r.db.Query(`
		SELECT
			id,
			user_id,
			cep,
			street,
			number,
			complement,
			neighborhood,
			city,
			state,
			updated_at,
			created_at
		FROM addresses
		WHERE id = ?
	`, ID)
	if err != nil {
		return nil, err
	}
	defer row.Close()

	if !row.Next() {
		if err = row.Err(); err != nil {
			return nil, err
		}
		return nil, apperrors.ErrNotFound
	}

	var addr entitiesAddress.Address
	var complement, neighborhood sql.NullString

	if err = row.Scan(
		&addr.Id,
		&addr.UserId,
		&addr.Cep,
		&addr.Street,
		&addr.Number,
		&complement,
		&neighborhood,
		&addr.City,
		&addr.State,
		&addr.UpdateAt,
		&addr.CreateAt,
	); err != nil {
		return nil, err
	}

	addr.Complement = complement.String
	addr.Neighborhood = neighborhood.String

	return &addr, nil
}

// ReadAddressesByUserId retorna os enderecos completos salvos pelo usuario, do mais recente para o mais antigo.
// Enderecos legados migrados apenas com CEP (sem rua/numero/cidade/estado) sao omitidos da lista reutilizavel.
func (r *addressRepository) ReadAddressesByUserId(userId string) ([]entitiesAddress.Address, error) {
	rows, err := r.db.Query(`
		SELECT
			id,
			user_id,
			cep,
			street,
			number,
			complement,
			neighborhood,
			city,
			state,
			updated_at,
			created_at
		FROM addresses
		WHERE user_id = ?
		  AND street <> ''
		  AND number <> ''
		  AND city <> ''
		  AND state <> ''
		ORDER BY created_at DESC
	`, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]entitiesAddress.Address, 0)
	for rows.Next() {
		var addr entitiesAddress.Address
		var complement, neighborhood sql.NullString

		if err = rows.Scan(
			&addr.Id,
			&addr.UserId,
			&addr.Cep,
			&addr.Street,
			&addr.Number,
			&complement,
			&neighborhood,
			&addr.City,
			&addr.State,
			&addr.UpdateAt,
			&addr.CreateAt,
		); err != nil {
			return nil, err
		}

		addr.Complement = complement.String
		addr.Neighborhood = neighborhood.String
		list = append(list, addr)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

// FindMatchingUserAddress busca se o usuario ja possui um endereco identico salvo para evitar duplicatas exatas.
func (r *addressRepository) FindMatchingUserAddress(userId string, address entitiesAddress.Address) (*entitiesAddress.Address, error) {
	row, err := r.db.Query(`
		SELECT
			id,
			user_id,
			cep,
			street,
			number,
			complement,
			neighborhood,
			city,
			state,
			updated_at,
			created_at
		FROM addresses
		WHERE user_id = ?
		  AND cep = ?
		  AND street = ?
		  AND number = ?
		  AND COALESCE(complement, '') = ?
		  AND COALESCE(neighborhood, '') = ?
		  AND city = ?
		  AND state = ?
		ORDER BY created_at DESC
		LIMIT 1
	`,
		userId,
		address.Cep,
		address.Street,
		address.Number,
		strings.TrimSpace(address.Complement),
		strings.TrimSpace(address.Neighborhood),
		address.City,
		address.State,
	)
	if err != nil {
		return nil, err
	}
	defer row.Close()

	if !row.Next() {
		if err = row.Err(); err != nil {
			return nil, err
		}
		return nil, apperrors.ErrNotFound
	}

	var addr entitiesAddress.Address
	var complement, neighborhood sql.NullString
	if err = row.Scan(
		&addr.Id,
		&addr.UserId,
		&addr.Cep,
		&addr.Street,
		&addr.Number,
		&complement,
		&neighborhood,
		&addr.City,
		&addr.State,
		&addr.UpdateAt,
		&addr.CreateAt,
	); err != nil {
		return nil, err
	}

	addr.Complement = complement.String
	addr.Neighborhood = neighborhood.String
	return &addr, nil
}
