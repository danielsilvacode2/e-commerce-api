package repository

import (
	"database/sql"
	"e-commerce/core/apperrors"
	entitiesCart "e-commerce/core/entities/cart"
	entitiesCartProduct "e-commerce/core/entities/cart_product"
	entitiesProduct "e-commerce/core/entities/product"
	entitiesUser "e-commerce/core/entities/user"
	"errors"

	"github.com/go-sql-driver/mysql"
)

type cartProductRepository struct {
	db *sql.DB
}

func NewCartProductRepository(db *sql.DB) entitiesCartProduct.CartProductRepository {
	return &cartProductRepository{
		db: db,
	}
}

func (r *cartProductRepository) AddProductCart(cartProduct entitiesCartProduct.CartProduct) (err error) {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	rows, err := tx.Query(
		"SELECT id FROM cart_products WHERE cart_id = ? AND product_id = ? ORDER BY updated_at DESC, id DESC FOR UPDATE",
		cartProduct.Cart.Id,
		cartProduct.Product.Id,
	)
	if err != nil {
		return err
	}

	var existingIDs []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			_ = rows.Close()
			return err
		}
		existingIDs = append(existingIDs, id)
	}
	if err = rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	_ = rows.Close()

	if len(existingIDs) > 0 {
		primaryID := existingIDs[0]
		if _, err = tx.Exec(
			"UPDATE cart_products SET quantity = ? WHERE id = ?",
			cartProduct.Quantity,
			primaryID,
		); err != nil {
			return err
		}

		// Auto-saneia eventuais linhas duplicadas anteriores a aplicacao da constraint UNIQUE(cart_id, product_id)
		if len(existingIDs) > 1 {
			if _, err = tx.Exec(
				"DELETE FROM cart_products WHERE cart_id = ? AND product_id = ? AND id <> ?",
				cartProduct.Cart.Id,
				cartProduct.Product.Id,
				primaryID,
			); err != nil {
				return err
			}
		}

		return tx.Commit()
	}

	if _, err = tx.Exec("INSERT INTO cart_products "+
		"(id, cart_id, product_id, quantity, isSelect) "+
		"VALUES (?,?,?,?,?) "+
		"ON DUPLICATE KEY UPDATE "+
		"quantity = VALUES(quantity)",
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

	return tx.Commit()
}

func (r *cartProductRepository) DecreaseProductCart(cartProductId string, quantity int64) error {

	result, err := r.db.Exec(
		"UPDATE cart_products SET quantity = (quantity - ?) WHERE id = ? AND quantity > ?",
		quantity,
		cartProductId,
		quantity,
	)
	if err != nil {

		var sqlErr *mysql.MySQLError

		if errors.As(err, &sqlErr) && sqlErr.Number == 1062 {
			return apperrors.ErrConflict
		}

		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return apperrors.ErrUnprocessableEntity
	}

	return nil
}


func (r *cartProductRepository) ReadAllProductsCart(userId string) ([]entitiesCartProduct.CartProduct, error) {

	rows, err := r.db.Query(`
        SELECT 
            cartproducts.id,
            cartproducts.quantity,
            cartproducts.isSelect,
            cart.id,
            cart.status,
            user.id,
            user.user_name,
            user.user_email,
            user.user_role,
            product.id,
            product.product_name,
            product.product_description,
            product.product_price,
            product.product_image,
            product.product_sku,
            product.product_quantity
        FROM cart_products cartproducts
        JOIN products product ON cartproducts.product_id = product.id
        JOIN carts cart ON cartproducts.cart_id = cart.id AND cart.status = 'ACTIVE'
        JOIN users user ON cart.user_id = user.id
        WHERE user.id = ?
        ORDER BY cartproducts.created_at ASC, cartproducts.id ASC
    `, userId)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var listProductCarts []entitiesCartProduct.CartProduct

	for rows.Next() {

		cartProduct := entitiesCartProduct.CartProduct{
			Cart: &entitiesCart.Cart{
				User: &entitiesUser.User{},
			},
			Product: &entitiesProduct.Product{},
		}

		if err = rows.Scan(
			&cartProduct.Id,
			&cartProduct.Quantity,
			&cartProduct.IsSelect,
			&cartProduct.Cart.Id,
			&cartProduct.Cart.Status,
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
			&cartProduct.Product.Quantity,
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

func (r *cartProductRepository) ReadProductsCart(cartProductId string) (*entitiesCartProduct.CartProduct, error) {

	row, err := r.db.Query(`
        SELECT 
            cartproducts.id,
            cartproducts.quantity,
            cartproducts.isSelect,
            cart.id,
            cart.status,
            user.id,
            user.user_name,
            user.user_email,
            user.user_role,
            product.id,
            product.product_name,
            product.product_description,
            product.product_price,
            product.product_image,
            product.product_sku,
            product.product_quantity
        FROM cart_products cartproducts
        JOIN products product ON cartproducts.product_id = product.id
        JOIN carts cart ON cartproducts.cart_id = cart.id AND cart.status = 'ACTIVE'
        JOIN users user ON cart.user_id = user.id
        WHERE cartproducts.id = ?
    `, cartProductId)

	if err != nil {
		return nil, err
	}

	defer row.Close()

	cartProduct := entitiesCartProduct.CartProduct{
		Cart: &entitiesCart.Cart{
			User: &entitiesUser.User{},
		},
		Product: &entitiesProduct.Product{},
	}

	if !row.Next() {
		if err = row.Err(); err != nil {
			return nil, err
		}
		return nil, apperrors.ErrNotFound
	}

	if err = row.Scan(
		&cartProduct.Id,
		&cartProduct.Quantity,
		&cartProduct.IsSelect,
		&cartProduct.Cart.Id,
		&cartProduct.Cart.Status,
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
		&cartProduct.Product.Quantity,
	); err != nil {
		return nil, err
	}

	if err = row.Err(); err != nil {
		return nil, err
	}
	return &cartProduct, nil
}

func (r *cartProductRepository) RemoveProductCart(id string) error {

	result, err := r.db.Exec("DELETE FROM cart_products WHERE id = ?", id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return apperrors.ErrNotFound
	}

	return nil
}

func (r *cartProductRepository) UpdateIsSelect(valueSelect bool, cartProductId string) error {

	if _, err := r.db.Exec("UPDATE cart_products SET isSelect = ? WHERE id = ?", valueSelect, cartProductId); err != nil {
		return err
	}

	return nil
}

func (r *cartProductRepository) UpdateAllIsSelect(valueSelect bool, userID string) error {

	if _, err := r.db.Exec("UPDATE cart_products cartp "+
		"JOIN carts c ON cartp.cart_id = c.id "+
		"SET cartp.isSelect = ? "+
		"WHERE c.user_id = ? AND c.status = 'ACTIVE'", valueSelect, userID); err != nil {
		return err
	}

	return nil
}
