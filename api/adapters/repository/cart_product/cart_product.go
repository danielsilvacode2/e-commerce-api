package repository

import (
	"database/sql"
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/cart_product"
	"errors"

	"github.com/go-sql-driver/mysql"
)

type cartProductRepository struct {
	db *sql.DB
}

func NewCartProductRepository(db *sql.DB) entities.CartProductRepository {
	return &cartProductRepository{
		db: db,
	}
}

func (r *cartProductRepository) AddProductCart(cartProduct entities.CartProduct) error {

	statement, err := r.db.Prepare("INSERT INTO cart_products " +
		"(id, cart_id, product_id,quantity ,isSelect)" +
		"VALUES (?,?,?,?,?)")

	if err != nil {
		return err
	}

	defer statement.Close()

	if _, err = statement.Exec(
		cartProduct.Id,
		cartProduct.Cart.Id,
		cartProduct.Product.Id,
		cartProduct.Quantity,
		cartProduct.IsSelect,
	); err != nil {

		var sqlErr *mysql.MySQLError

		if errors.As(err, &sqlErr) && sqlErr.Number == 1062 {
			return apperrors.ErrConflict
		}

		return err
	}

	return nil
}

func (r *cartProductRepository) ReadAllProductsCart(userId string) ([]entities.CartProduct, error) {

	rows, err := r.db.Query(`
        SELECT 
            cartproducts.id,
            cartproducts.quantity,
            cartproducts.isSelect,
            cart.id,
            user.id,
            user.user_name,
            user.user_email,
            user.user_role,
            product.id,
            product.product_name,
            product.product_description,
            product.product_price,
            product.product_image,
			product.product_sku
        FROM cart_products cartproducts
        JOIN products product ON cartproducts.product_id = product.id
        JOIN carts cart ON cartproducts.cart_id = cart.id
        JOIN users user ON cart.user_id = user.id
        WHERE user.id = ?
    `, userId)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var listProductCarts []entities.CartProduct

	for rows.Next() {

		var cartProduct entities.CartProduct

		if err = rows.Scan(
			&cartProduct.Id,
			&cartProduct.Quantity,
			&cartProduct.IsSelect,
			&cartProduct.Cart.Id,
			&cartProduct.Cart.User.ID,
			&cartProduct.Cart.User.Name,
			&cartProduct.Cart.User.Email,
			&cartProduct.Cart.User.Role,
			&cartProduct.Product.Id,
			&cartProduct.Product.Name,
			&cartProduct.Product.Description,
			&cartProduct.Product.Price,
			&cartProduct.Product.Image,
			&cartProduct.Product.Sku,
		); err != nil {

			return nil, err
		}

		listProductCarts = append(listProductCarts, cartProduct)

	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return listProductCarts, nil
}

func (r *cartProductRepository) RemoveProductCart(id string) error {
	return nil
}

func (r *cartProductRepository) UpdateIsSelect(bool, cartProductId string) error {
	return nil
}
