package repository

import (
	"context"
	entities "e-commerce/core/entities/product"
	"math"
	"strings"
)

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// escapeLike escapa os curingas do LIKE para que sejam tratados como texto
func escapeLike(term string) string {
	return likeEscaper.Replace(term)
}

// SearchProducts busca produtos por nome, sku ou descricao usando LIKE parametrizado.
// No MySQL o LIKE ja e case-insensitive com collation *_ci.
func (r *productRepository) SearchProducts(ctx context.Context, term string, page, limit int) (*entities.PaginatedProducts, error) {
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

	pattern := "%" + escapeLike(term) + "%"
	where := "WHERE product_name LIKE ? OR product_sku LIKE ? OR product_description LIKE ?"

	var totalRows int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM products "+where, pattern, pattern, pattern).Scan(&totalRows)
	if err != nil {
		return nil, err
	}

	query := "SELECT id, product_name, product_description, product_sku, product_price, product_image, updated_at, created_at, product_quantity " +
		"FROM products " + where + " ORDER BY product_name, id LIMIT ? OFFSET ?"

	rows, err := r.db.QueryContext(ctx, query, pattern, pattern, pattern, limit, offset)
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

	return &entities.PaginatedProducts{
		Data:       listProducts,
		Page:       page,
		Limit:      limit,
		TotalRows:  totalRows,
		TotalPages: int(math.Ceil(float64(totalRows) / float64(limit))),
	}, nil
}
