package repository

import (
	"database/sql"
	"e-commerce/core/apperrors"
	entitiesOrder "e-commerce/core/entities/order"
	entitiesOrderItem "e-commerce/core/entities/order_item"
	entitiesProduct "e-commerce/core/entities/product"
	"errors"

	"github.com/go-sql-driver/mysql"
)

type orderItemRepository struct {
	db *sql.DB
}

func NewOrderItemRepository(db *sql.DB) entitiesOrderItem.OrderItemRepository {
	return &orderItemRepository{
		db: db,
	}
}

const selectOrderItem = `
        SELECT
            orderitems.id,
            orderitems.order_id,
            orderitems.quantity,
            orderitems.unit_price,
            product.id,
            product.product_name,
            product.product_sku,
            product.product_image
        FROM order_items orderitems
        INNER JOIN products product ON orderitems.product_id = product.id
`

// AddOrderItem insere o item e baixa o estoque na mesma transacao
func (r *orderItemRepository) AddOrderItem(item entitiesOrderItem.OrderItem) (err error) {

	tx, err := r.db.Begin()

	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	if err = lockEditableOrder(tx, item.OrderId.String()); err != nil {
		return err
	}

	if err = decreaseStock(tx, item.Product.Id.String(), item.Quantity); err != nil {
		return err
	}

	if _, err = tx.Exec("INSERT INTO order_items (id, order_id, product_id, quantity, unit_price) VALUES (?,?,?,?,?)",
		item.Id,
		item.OrderId,
		item.Product.Id,
		item.Quantity,
		item.UnitPrice,
	); err != nil {

		var sqlErr *mysql.MySQLError

		if errors.As(err, &sqlErr) && sqlErr.Number == 1062 {
			err = apperrors.ErrConflict
		}

		return err
	}

	return tx.Commit()
}

func (r *orderItemRepository) ReadOrderItemById(ID string) (*entitiesOrderItem.OrderItem, error) {

	row, err := r.db.Query(selectOrderItem+" WHERE orderitems.id = ?", ID)

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

	item := entitiesOrderItem.OrderItem{
		Product: &entitiesProduct.Product{},
	}

	if err = row.Scan(
		&item.Id,
		&item.OrderId,
		&item.Quantity,
		&item.UnitPrice,
		&item.Product.Id,
		&item.Product.Name,
		&item.Product.Sku,
		&item.Product.Image,
	); err != nil {
		return nil, err
	}

	return &item, nil
}

func (r *orderItemRepository) ReadOrderItemsByOrderId(orderId string) ([]entitiesOrderItem.OrderItem, error) {

	rows, err := r.db.Query(selectOrderItem+" WHERE orderitems.order_id = ?", orderId)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var listItems []entitiesOrderItem.OrderItem

	for rows.Next() {

		item := entitiesOrderItem.OrderItem{
			Product: &entitiesProduct.Product{},
		}

		if err = rows.Scan(
			&item.Id,
			&item.OrderId,
			&item.Quantity,
			&item.UnitPrice,
			&item.Product.Id,
			&item.Product.Name,
			&item.Product.Sku,
			&item.Product.Image,
		); err != nil {
			return nil, err
		}

		listItems = append(listItems, item)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return listItems, nil
}

// UpdateOrderItemQuantity altera a quantidade do item ajustando o estoque pela diferenca
func (r *orderItemRepository) UpdateOrderItemQuantity(orderId string, orderItemId string, quantity int64) (err error) {

	tx, err := r.db.Begin()

	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	if err = lockEditableOrder(tx, orderId); err != nil {
		return err
	}

	productId, currentQuantity, err := lockOrderItem(tx, orderId, orderItemId)

	if err != nil {
		return err
	}

	delta := quantity - currentQuantity

	if delta > 0 {
		if err = decreaseStock(tx, productId, delta); err != nil {
			return err
		}
	}

	if delta < 0 {
		if err = increaseStock(tx, productId, -delta); err != nil {
			return err
		}
	}

	if _, err = tx.Exec("UPDATE order_items SET quantity = ? WHERE id = ?", quantity, orderItemId); err != nil {
		return err
	}

	return tx.Commit()
}

// RemoveOrderItem remove o item e devolve a quantidade ao estoque
func (r *orderItemRepository) RemoveOrderItem(orderId string, orderItemId string) (err error) {

	tx, err := r.db.Begin()

	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	if err = lockEditableOrder(tx, orderId); err != nil {
		return err
	}

	productId, currentQuantity, err := lockOrderItem(tx, orderId, orderItemId)

	if err != nil {
		return err
	}

	if err = increaseStock(tx, productId, currentQuantity); err != nil {
		return err
	}

	if _, err = tx.Exec("DELETE FROM order_items WHERE id = ?", orderItemId); err != nil {
		return err
	}

	return tx.Commit()
}

// lockEditableOrder trava a linha do pedido e garante que ele ainda esta em PENDING_PAYMENT,
// evitando modificacao de itens apos pagamento aprovado ou concorrencia com mudanca de status
func lockEditableOrder(tx *sql.Tx, orderId string) error {

	var status string

	if err := tx.QueryRow("SELECT status FROM orders WHERE id = ? FOR UPDATE", orderId).Scan(&status); err != nil {

		if errors.Is(err, sql.ErrNoRows) {
			return apperrors.ErrNotFound
		}

		return err
	}

	if status != entitiesOrder.OrderStatusPendingPayment {
		return apperrors.ErrUnprocessableEntity
	}

	// Impede alteracao de itens apos cobranca ja gerada no gateway de pagamento (previne divergencia de valor)
	var activeGatewayPayments int
	if err := tx.QueryRow(
		"SELECT COUNT(*) FROM payments WHERE order_id = ? AND status = 'PENDING' AND external_payment_id IS NOT NULL AND external_payment_id <> ''",
		orderId,
	).Scan(&activeGatewayPayments); err != nil {
		return err
	}
	if activeGatewayPayments > 0 {
		return apperrors.ErrConflict
	}

	return nil
}

func lockOrderItem(tx *sql.Tx, orderId string, orderItemId string) (string, int64, error) {

	var productId string
	var quantity int64

	if err := tx.QueryRow("SELECT product_id, quantity FROM order_items WHERE id = ? AND order_id = ? FOR UPDATE",
		orderItemId, orderId).Scan(&productId, &quantity); err != nil {

		if errors.Is(err, sql.ErrNoRows) {
			return "", 0, apperrors.ErrNotFound
		}

		return "", 0, err
	}

	return productId, quantity, nil
}

func decreaseStock(tx *sql.Tx, productId string, quantity int64) error {

	result, err := tx.Exec("UPDATE products SET product_quantity = product_quantity - ? WHERE id = ? AND product_quantity >= ?",
		quantity, productId, quantity)

	if err != nil {
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

func increaseStock(tx *sql.Tx, productId string, quantity int64) error {

	_, err := tx.Exec("UPDATE products SET product_quantity = product_quantity + ? WHERE id = ?", quantity, productId)

	return err
}
