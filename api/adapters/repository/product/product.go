package repository

import (
	"database/sql"
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/product"
	"errors"

	"github.com/go-sql-driver/mysql"
)

type productRepository struct {
	db *sql.DB
}

// NewProcutRepository retorna uma instancia de productRepository
func NewProductRepository(db *sql.DB) entities.ProductRepository {
	return &productRepository{
		db: db,
	}
}

// CreateProduct recebe um produto e persisti no banco de dados
func (r *productRepository) CreateProduct(product entities.Product) error {

	statement, err := r.db.Prepare("INSERT INTO products " +
		"(Id, product_name, product_description, product_sku, product_price, product_quantity, product_image ) " +
		"VALUES (?,?,?,?,?,?,?)")

	if err != nil {
		return err
	}

	defer statement.Close()

	if _, err = statement.Exec(
		product.Id,
		product.Name,
		product.Description,
		product.Sku,
		product.Price,
		product.Quantity,
		product.Image,
	); err != nil {

		var errSQL *mysql.MySQLError

		if errors.As(err, &errSQL) && errSQL.Number == 1062 {
			return apperrors.ErrConflict
		}

		return err
	}

	return nil
}

func (r *productRepository) ReadProdutById(Id string) (*entities.Product, error) {

	row, err := r.db.Query("SELECT * FROM products WHERE id = ?", Id)

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

	var product entities.Product

	if err = row.Scan(
		&product.Id,
		&product.Name,
		&product.Description,
		&product.Sku,
		&product.Price,
		&product.Image,
		&product.UpdateAt,
		&product.CreateAt,
		&product.Quantity,
	); err != nil {
		return nil, err
	}

	return &product, nil
}
func (r *productRepository) ReadAllProduct() ([]entities.Product, error) {

	rows, err := r.db.Query("SELECT * FROM products")

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var listProducts []entities.Product

	for rows.Next() {

		var product entities.Product

		if err = rows.Scan(
			&product.Id,
			&product.Name,
			&product.Description,
			&product.Sku,
			&product.Price,
			&product.Image,
			&product.UpdateAt,
			&product.CreateAt,
			&product.Quantity,
		); err != nil {
			return nil, err
		}

		listProducts = append(listProducts, product)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return listProducts, nil
}

func (r *productRepository) DeleteProductById(Id string) error {

	statement, err := r.db.Prepare("DELETE FROM products WHERE id = ?")

	if err != nil {
		return err
	}

	defer statement.Close()

	if _, err := statement.Exec(Id); err != nil {
		return err
	}

	return nil
}

func (r *productRepository) UpdateProduct(product entities.Product, ID string) error {

	statement, err := r.db.Prepare("UPDATE products SET " +
		"product_name = ? , product_sku = ?, product_description = ? , product_price = ?, product_image = ? WHERE id = ?")

	if err != nil {
		return err
	}

	defer statement.Close()

	_, err = statement.Exec(
		product.Name,
		product.Sku,
		product.Description,
		product.Price,
		product.Image,
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
