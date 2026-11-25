package repository

import (
	"database/sql"
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/product"
	"errors"
	"math"

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

	if _, err := r.db.Exec("INSERT INTO products "+
		"(Id, product_name, product_description, product_sku, product_price, product_quantity, product_image ) "+
		"VALUES (?,?,?,?,?,?,?)",
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

	var product entities.Product

	err := r.db.QueryRow(
		"SELECT id, product_name, product_description, product_sku, product_price, product_image, updated_at, created_at, product_quantity FROM products WHERE id = ?",
		Id,
	).Scan(
		&product.Id,
		&product.Name,
		&product.Description,
		&product.Sku,
		&product.Price,
		&product.Image,
		&product.UpdateAt,
		&product.CreateAt,
		&product.Quantity,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}

	return &product, nil
}

func (r *productRepository) ReadAllProduct(page, limit int) (*entities.PaginatedProducts, error) {
	// 1. Validação básica dos parâmetros de paginação
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	// 2. Obter o total de registros na tabela
	var totalRows int64
	err := r.db.QueryRow("SELECT COUNT(*) FROM products").Scan(&totalRows)
	if err != nil {
		return nil, err
	}

	// 3. Executar a query com ORDER BY deterministico, LIMIT e OFFSET
	query := "SELECT id, product_name, product_description, product_sku, product_price, product_image, updated_at, created_at, product_quantity FROM products ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?"
	
	// Nota: Se estiver usando PostgreSQL, substitua '?' por '$1', '$2' ($1 = limit, $2 = offset)
	rows, err := r.db.Query(query, limit, offset)
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

	// 4. Calcular o total de páginas
	totalPages := int(math.Ceil(float64(totalRows) / float64(limit)))

	// 5. Retornar o resultado paginado
	return &entities.PaginatedProducts{
		Data:       listProducts,
		Page:       page,
		Limit:      limit,
		TotalRows:  totalRows,
		TotalPages: totalPages,
	}, nil
}

func (r *productRepository) DeleteProductById(Id string) error {

	if _, err := r.db.Exec("DELETE FROM products WHERE id = ?", Id); err != nil {
		return err
	}

	return nil
}

func (r *productRepository) UpdateProduct(product entities.Product, ID string) error {

	_, err := r.db.Exec("UPDATE products SET "+
		"product_name = ? , product_sku = ?, product_description = ? , product_price = ?, product_image = ?, product_quantity = ? WHERE id = ?",
		product.Name,
		product.Sku,
		product.Description,
		product.Price,
		product.Image,
		product.Quantity,
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
