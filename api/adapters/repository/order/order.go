package repository

import (
	"database/sql"
	"e-commerce/core/apperrors"
	entitiesOrder "e-commerce/core/entities/order"
	entitiesOrderItem "e-commerce/core/entities/order_item"
	entitiesProduct "e-commerce/core/entities/product"
	entitiesUser "e-commerce/core/entities/user"
	"errors"
	"math"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

type orderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) entitiesOrder.OrderRepository {
	return &orderRepository{
		db: db,
	}
}

// CreateOrder persiste o pedido, os itens, baixa o estoque, grava o historico
// e remove os itens do carrinho em uma unica transacao.
func (r *orderRepository) CreateOrder(order entitiesOrder.Order, cartProductIds []string) (err error) {

	tx, err := r.db.Begin()

	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	if _, err = tx.Exec("INSERT INTO orders (id, user_id, status, zip_code) VALUES (?,?,?,?)",
		order.Id,
		order.User.ID,
		order.Status,
		order.ZipCode,
	); err != nil {

		var sqlErr *mysql.MySQLError

		if errors.As(err, &sqlErr) && sqlErr.Number == 1062 {
			return apperrors.ErrConflict
		}

		return err
	}

	for _, item := range order.Items {

		if err = decreaseStock(tx, item.Product.Id.String(), item.Quantity); err != nil {
			return err
		}

		if _, err = tx.Exec("INSERT INTO order_items (id, order_id, product_id, quantity, unit_price) VALUES (?,?,?,?,?)",
			item.Id,
			order.Id,
			item.Product.Id,
			item.Quantity,
			item.UnitPrice,
		); err != nil {
			return err
		}
	}

	if err = insertStatusHistory(tx, order.Id.String(), order.Status); err != nil {
		return err
	}

	if len(cartProductIds) > 0 {

		placeholders, args := inClause(cartProductIds)

		if _, err = tx.Exec("DELETE FROM cart_products WHERE id IN ("+placeholders+")", args...); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *orderRepository) ReadOrderById(ID string) (*entitiesOrder.Order, error) {

	row, err := r.db.Query(`
        SELECT
            orders.id,
            orders.status,
            orders.zip_code,
            orders.tracking_code,
            orders.refund_reason,
            orders.refund_rejection_reason,
            orders.updated_at,
            orders.created_at,
            users.id,
            users.user_name,
            users.user_email
        FROM orders
        INNER JOIN users ON orders.user_id = users.id
        WHERE orders.id = ?
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

	order := entitiesOrder.Order{
		User: &entitiesUser.User{},
	}

	var trackingCode, refundReason, refundRejectionReason sql.NullString

	if err = row.Scan(
		&order.Id,
		&order.Status,
		&order.ZipCode,
		&trackingCode,
		&refundReason,
		&refundRejectionReason,
		&order.UpdateAt,
		&order.CreateAt,
		&order.User.ID,
		&order.User.Name,
		&order.User.Email,
	); err != nil {
		return nil, err
	}

	order.TrackingCode = trackingCode.String
	order.RefundReason = refundReason.String
	order.RefundRejectionReason = refundRejectionReason.String

	mapItems, err := r.readItemsByOrderIds([]string{order.Id.String()})

	if err != nil {
		return nil, err
	}

	order.Items = mapItems[order.Id.String()]

	return &order, nil
}

func (r *orderRepository) ReadAllOrders(filter entitiesOrder.OrderFilter) (*entitiesOrder.PaginatedOrders, error) {

	page := filter.Page
	limit := filter.Limit

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

	where := " WHERE 1 = 1"
	var args []interface{}

	if filter.Status != "" {
		where += " AND orders.status = ?"
		args = append(args, filter.Status)
	}

	if filter.UserId != "" {
		where += " AND orders.user_id = ?"
		args = append(args, filter.UserId)
	}

	var totalRows int64

	if err := r.db.QueryRow("SELECT COUNT(*) FROM orders"+where, args...).Scan(&totalRows); err != nil {
		return nil, err
	}

	query := "SELECT orders.id, orders.status, orders.zip_code, orders.tracking_code, orders.refund_reason, orders.refund_rejection_reason, orders.updated_at, orders.created_at," +
		" users.id, users.user_name, users.user_email" +
		" FROM orders" +
		" INNER JOIN users ON orders.user_id = users.id" +
		where +
		" ORDER BY orders.created_at DESC LIMIT ? OFFSET ?"

	rows, err := r.db.Query(query, append(args, limit, offset)...)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var listOrders []entitiesOrder.Order
	var listIds []string

	for rows.Next() {

		order := entitiesOrder.Order{
			User: &entitiesUser.User{},
		}

		var trackingCode, refundReason, refundRejectionReason sql.NullString

		if err = rows.Scan(
			&order.Id,
			&order.Status,
			&order.ZipCode,
			&trackingCode,
			&refundReason,
			&refundRejectionReason,
			&order.UpdateAt,
			&order.CreateAt,
			&order.User.ID,
			&order.User.Name,
			&order.User.Email,
		); err != nil {
			return nil, err
		}

		order.TrackingCode = trackingCode.String
		order.RefundReason = refundReason.String
		order.RefundRejectionReason = refundRejectionReason.String

		listOrders = append(listOrders, order)
		listIds = append(listIds, order.Id.String())
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if len(listIds) > 0 {

		mapItems, err := r.readItemsByOrderIds(listIds)

		if err != nil {
			return nil, err
		}

		for i := range listOrders {
			listOrders[i].Items = mapItems[listOrders[i].Id.String()]
		}
	}

	totalPages := int(math.Ceil(float64(totalRows) / float64(limit)))

	return &entitiesOrder.PaginatedOrders{
		Data:       listOrders,
		Page:       page,
		Limit:      limit,
		TotalRows:  totalRows,
		TotalPages: totalPages,
	}, nil
}

func (r *orderRepository) UpdateOrderZipCode(zipCode string, ID string) error {

	statement, err := r.db.Prepare("UPDATE orders SET zip_code = ? WHERE id = ?")

	if err != nil {
		return err
	}

	defer statement.Close()

	if _, err = statement.Exec(zipCode, ID); err != nil {
		return err
	}

	return nil
}

// UpdateOrderStatus altera o status com lock na linha do pedido, grava o historico
// e devolve o estoque quando restoreStock = true. Retorna ErrConflict se o status
// atual mudou desde a leitura feita pelo caso de uso.
func (r *orderRepository) UpdateOrderStatus(ID string, currentStatus string, newStatus string, restoreStock bool) error {
	return r.UpdateOrderStatusWithDetails(ID, currentStatus, newStatus, restoreStock, nil, nil, nil)
}

// UpdateOrderStatusWithDetails altera o status e campos opcionais (tracking_code, refund_reason, refund_rejection_reason)
// na mesma transacao com lock FOR UPDATE.
func (r *orderRepository) UpdateOrderStatusWithDetails(
	ID string,
	currentStatus string,
	newStatus string,
	restoreStock bool,
	trackingCode *string,
	refundReason *string,
	refundRejectionReason *string,
) (err error) {

	tx, err := r.db.Begin()

	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var status string

	if err = tx.QueryRow("SELECT status FROM orders WHERE id = ? FOR UPDATE", ID).Scan(&status); err != nil {

		if errors.Is(err, sql.ErrNoRows) {
			err = apperrors.ErrNotFound
		}

		return err
	}

	if status != currentStatus {
		err = apperrors.ErrConflict
		return err
	}

	if restoreStock {
		if err = restoreOrderStock(tx, ID); err != nil {
			return err
		}
	}

	setClauses := []string{"status = ?"}
	args := []interface{}{newStatus}

	if trackingCode != nil {
		setClauses = append(setClauses, "tracking_code = ?")
		args = append(args, *trackingCode)
	}
	if refundReason != nil {
		setClauses = append(setClauses, "refund_reason = ?")
		args = append(args, *refundReason)
	}
	if refundRejectionReason != nil {
		setClauses = append(setClauses, "refund_rejection_reason = ?")
		args = append(args, *refundRejectionReason)
	}
	args = append(args, ID)

	if _, err = tx.Exec("UPDATE orders SET "+strings.Join(setClauses, ", ")+" WHERE id = ?", args...); err != nil {
		return err
	}

	if err = insertStatusHistory(tx, ID, newStatus); err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteOrder remove o pedido (itens e historico via ON DELETE CASCADE),
// devolvendo o estoque quando restoreStock = true.
func (r *orderRepository) DeleteOrder(ID string, restoreStock bool) (err error) {

	tx, err := r.db.Begin()

	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	if restoreStock {
		if err = restoreOrderStock(tx, ID); err != nil {
			return err
		}
	}

	if _, err = tx.Exec("DELETE FROM orders WHERE id = ?", ID); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *orderRepository) readItemsByOrderIds(orderIds []string) (map[string][]entitiesOrderItem.OrderItem, error) {

	placeholders, args := inClause(orderIds)

	rows, err := r.db.Query(`
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
        WHERE orderitems.order_id IN (`+placeholders+`)
    `, args...)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	mapItems := make(map[string][]entitiesOrderItem.OrderItem)

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

		mapItems[item.OrderId.String()] = append(mapItems[item.OrderId.String()], item)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return mapItems, nil
}

// decreaseStock baixa o estoque somente se houver quantidade suficiente
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

// restoreOrderStock devolve ao estoque as quantidades de todos os itens do pedido
func restoreOrderStock(tx *sql.Tx, orderId string) error {

	_, err := tx.Exec(`
        UPDATE products product
        INNER JOIN (
            SELECT product_id, SUM(quantity) AS quantity
            FROM order_items
            WHERE order_id = ?
            GROUP BY product_id
        ) orderitems ON orderitems.product_id = product.id
        SET product.product_quantity = product.product_quantity + orderitems.quantity
    `, orderId)

	return err
}

func insertStatusHistory(tx *sql.Tx, orderId string, status string) error {

	_, err := tx.Exec("INSERT INTO order_status_history (id, order_id, status) VALUES (?,?,?)",
		uuid.New(), orderId, status)

	return err
}

func inClause(values []string) (string, []interface{}) {

	args := make([]interface{}, len(values))

	for i, value := range values {
		args[i] = value
	}

	return strings.TrimSuffix(strings.Repeat("?,", len(values)), ","), args
}
