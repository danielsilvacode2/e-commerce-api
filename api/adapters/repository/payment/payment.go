package repository

import (
	"database/sql"
	"e-commerce/core/apperrors"
	entitiesOrder "e-commerce/core/entities/order"
	entitiesPayment "e-commerce/core/entities/payment"
	"errors"
	"math"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

type paymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) entitiesPayment.PaymentRepository {
	return &paymentRepository{
		db: db,
	}
}

func (r *paymentRepository) CreatePayment(payment entitiesPayment.Payment) error {
	var externalId sql.NullString
	if payment.ExternalPaymentId != "" {
		externalId = sql.NullString{String: payment.ExternalPaymentId, Valid: true}
	}

	var pixQrCode sql.NullString
	if payment.PixQrCode != "" {
		pixQrCode = sql.NullString{String: payment.PixQrCode, Valid: true}
	}

	var pixCopyPaste sql.NullString
	if payment.PixCopyPaste != "" {
		pixCopyPaste = sql.NullString{String: payment.PixCopyPaste, Valid: true}
	}

	var cardLast4 sql.NullString
	if payment.CardLast4 != "" {
		cardLast4 = sql.NullString{String: payment.CardLast4, Valid: true}
	}

	var cardBrand sql.NullString
	if payment.CardBrand != "" {
		cardBrand = sql.NullString{String: payment.CardBrand, Valid: true}
	}

	_, err := r.db.Exec(`
		INSERT INTO payments (
			id, order_id, provider, method, external_payment_id, idempotency_key,
			amount, installments, status, pix_qr_code, pix_copy_paste,
			card_last4, card_brand, expires_at, paid_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	`,
		payment.Id,
		payment.OrderId,
		payment.Provider,
		payment.Method,
		externalId,
		payment.IdempotencyKey,
		payment.Amount,
		payment.Installments,
		payment.Status,
		pixQrCode,
		pixCopyPaste,
		cardLast4,
		cardBrand,
		payment.ExpiresAt,
		payment.PaidAt,
	)

	if err != nil {
		var sqlErr *mysql.MySQLError
		if errors.As(err, &sqlErr) && sqlErr.Number == 1062 {
			return apperrors.ErrConflict
		}
		return err
	}

	return nil
}

const paymentSelectColumns = `
	id, order_id, provider, method, external_payment_id, idempotency_key,
	amount, installments, status, pix_qr_code, pix_copy_paste,
	card_last4, card_brand, expires_at, paid_at, refunded_at, created_at, updated_at
`

func scanPaymentRow(scanner interface{ Scan(dest ...any) error }) (*entitiesPayment.Payment, error) {
	var p entitiesPayment.Payment
	var externalId, pixQrCode, pixCopyPaste, cardLast4, cardBrand sql.NullString
	var expiresAt, paidAt, refundedAt sql.NullTime

	if err := scanner.Scan(
		&p.Id,
		&p.OrderId,
		&p.Provider,
		&p.Method,
		&externalId,
		&p.IdempotencyKey,
		&p.Amount,
		&p.Installments,
		&p.Status,
		&pixQrCode,
		&pixCopyPaste,
		&cardLast4,
		&cardBrand,
		&expiresAt,
		&paidAt,
		&refundedAt,
		&p.CreatedAt,
		&p.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}

	p.ExternalPaymentId = externalId.String
	p.PixQrCode = pixQrCode.String
	p.PixCopyPaste = pixCopyPaste.String
	p.CardLast4 = cardLast4.String
	p.CardBrand = cardBrand.String
	if expiresAt.Valid {
		t := expiresAt.Time
		p.ExpiresAt = &t
	}
	if paidAt.Valid {
		t := paidAt.Time
		p.PaidAt = &t
	}
	if refundedAt.Valid {
		t := refundedAt.Time
		p.RefundedAt = &t
	}

	return &p, nil
}

func (r *paymentRepository) ReadPaymentById(id string) (*entitiesPayment.Payment, error) {
	row := r.db.QueryRow("SELECT "+paymentSelectColumns+" FROM payments WHERE id = ?", id)
	return scanPaymentRow(row)
}

func (r *paymentRepository) ReadPaymentByOrderId(orderId string) (*entitiesPayment.Payment, error) {
	row := r.db.QueryRow(
		"SELECT "+paymentSelectColumns+" FROM payments WHERE order_id = ? ORDER BY created_at DESC LIMIT 1",
		orderId,
	)
	return scanPaymentRow(row)
}

func (r *paymentRepository) ReadPaymentByIdempotencyKey(key string) (*entitiesPayment.Payment, error) {
	row := r.db.QueryRow(
		"SELECT "+paymentSelectColumns+" FROM payments WHERE idempotency_key = ?",
		key,
	)
	return scanPaymentRow(row)
}

func (r *paymentRepository) ReadPaymentByExternalId(externalId string) (*entitiesPayment.Payment, error) {
	row := r.db.QueryRow(
		"SELECT "+paymentSelectColumns+" FROM payments WHERE external_payment_id = ?",
		externalId,
	)
	return scanPaymentRow(row)
}

// ApprovePaymentAndOrder aprova o pagamento e transiciona o pedido de PENDING_PAYMENT para PREPARING atomicamente.
func (r *paymentRepository) ApprovePaymentAndOrder(payment entitiesPayment.Payment, orderId string) (err error) {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var orderStatus string
	if err = tx.QueryRow("SELECT status FROM orders WHERE id = ? FOR UPDATE", orderId).Scan(&orderStatus); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = apperrors.ErrNotFound
		}
		return err
	}

	if orderStatus != entitiesOrder.OrderStatusPendingPayment {
		err = apperrors.ErrConflict
		return err
	}

	// Garante integridade financeira: o valor pago deve coincidir com o total atual dos itens do pedido
	var currentOrderTotal float64
	if err = tx.QueryRow(
		"SELECT COALESCE(SUM(quantity * unit_price), 0) FROM order_items WHERE order_id = ?",
		orderId,
	).Scan(&currentOrderTotal); err != nil {
		return err
	}
	if math.Abs(currentOrderTotal-payment.Amount) > 0.01 {
		err = apperrors.ErrConflict
		return err
	}

	now := time.Now()
	if payment.PaidAt != nil {
		now = *payment.PaidAt
	}

	if _, err = tx.Exec(
		"UPDATE payments SET status = ?, paid_at = ? WHERE id = ?",
		entitiesPayment.PaymentStatusApproved,
		now,
		payment.Id,
	); err != nil {
		return err
	}

	if _, err = tx.Exec(
		"UPDATE orders SET status = ? WHERE id = ?",
		entitiesOrder.OrderStatusPreparing,
		orderId,
	); err != nil {
		return err
	}

	if _, err = tx.Exec(
		"INSERT INTO order_status_history (id, order_id, status) VALUES (?,?,?)",
		uuid.New(),
		orderId,
		entitiesOrder.OrderStatusPreparing,
	); err != nil {
		return err
	}

	return tx.Commit()
}

// FailOrExpirePaymentAndCancelOrder marca o pagamento como FAILED ou EXPIRED, cancela o pedido e restaura estoque.
func (r *paymentRepository) FailOrExpirePaymentAndCancelOrder(payment entitiesPayment.Payment, orderId string) (err error) {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var orderStatus string
	if err = tx.QueryRow("SELECT status FROM orders WHERE id = ? FOR UPDATE", orderId).Scan(&orderStatus); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = apperrors.ErrNotFound
		}
		return err
	}

	if orderStatus != entitiesOrder.OrderStatusPendingPayment {
		err = apperrors.ErrConflict
		return err
	}

	if _, err = tx.Exec(
		"UPDATE payments SET status = ? WHERE id = ?",
		payment.Status,
		payment.Id,
	); err != nil {
		return err
	}

	if _, err = tx.Exec(`
		UPDATE products product
		INNER JOIN (
			SELECT product_id, SUM(quantity) AS quantity
			FROM order_items
			WHERE order_id = ?
			GROUP BY product_id
		) orderitems ON orderitems.product_id = product.id
		SET product.product_quantity = product.product_quantity + orderitems.quantity
	`, orderId); err != nil {
		return err
	}

	if _, err = tx.Exec(
		"UPDATE orders SET status = ? WHERE id = ?",
		entitiesOrder.OrderStatusCanceled,
		orderId,
	); err != nil {
		return err
	}

	if _, err = tx.Exec(
		"INSERT INTO order_status_history (id, order_id, status) VALUES (?,?,?)",
		uuid.New(),
		orderId,
		entitiesOrder.OrderStatusCanceled,
	); err != nil {
		return err
	}

	return tx.Commit()
}

// RefundPaymentAndOrder efetiva o estorno do pagamento aprovado e transiciona o pedido para REFUNDED atomicamente.
func (r *paymentRepository) RefundPaymentAndOrder(
	orderId string,
	expectedOrderStatus string,
	restoreStock bool,
	refundReason *string,
) (refundedPayment *entitiesPayment.Payment, err error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var orderStatus string
	if err = tx.QueryRow("SELECT status FROM orders WHERE id = ? FOR UPDATE", orderId).Scan(&orderStatus); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = apperrors.ErrNotFound
		}
		return nil, err
	}

	if orderStatus != expectedOrderStatus {
		err = apperrors.ErrConflict
		return nil, err
	}

	row := tx.QueryRow(
		"SELECT "+paymentSelectColumns+" FROM payments WHERE order_id = ? AND status = ? ORDER BY created_at DESC LIMIT 1 FOR UPDATE",
		orderId,
		entitiesPayment.PaymentStatusApproved,
	)
	p, err := scanPaymentRow(row)
	if err != nil && !errors.Is(err, apperrors.ErrNotFound) {
		return nil, err
	}

	if p != nil {
		now := time.Now()
		if _, err = tx.Exec(
			"UPDATE payments SET status = ?, refunded_at = ? WHERE id = ?",
			entitiesPayment.PaymentStatusRefunded,
			now,
			p.Id,
		); err != nil {
			return nil, err
		}
		p.Status = entitiesPayment.PaymentStatusRefunded
		p.RefundedAt = &now
	}

	if restoreStock {
		if _, err = tx.Exec(`
			UPDATE products product
			INNER JOIN (
				SELECT product_id, SUM(quantity) AS quantity
				FROM order_items
				WHERE order_id = ?
				GROUP BY product_id
			) orderitems ON orderitems.product_id = product.id
			SET product.product_quantity = product.product_quantity + orderitems.quantity
		`, orderId); err != nil {
			return nil, err
		}
	}

	if refundReason != nil {
		if _, err = tx.Exec(
			"UPDATE orders SET status = ?, refund_reason = ? WHERE id = ?",
			entitiesOrder.OrderStatusRefunded,
			*refundReason,
			orderId,
		); err != nil {
			return nil, err
		}
	} else {
		if _, err = tx.Exec(
			"UPDATE orders SET status = ? WHERE id = ?",
			entitiesOrder.OrderStatusRefunded,
			orderId,
		); err != nil {
			return nil, err
		}
	}

	if _, err = tx.Exec(
		"INSERT INTO order_status_history (id, order_id, status) VALUES (?,?,?)",
		uuid.New(),
		orderId,
		entitiesOrder.OrderStatusRefunded,
	); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return p, nil
}
