-- Migration incremental: Fluxo de Pagamento Idempotente, Máquina de Estados e Estorno (Refund)
-- Compatível com o schema existente em sql/script.sql

USE ecommerce;

-- 1. Ampliar ENUM de status da tabela orders e adicionar colunas de rastreio e estorno
ALTER TABLE orders
    MODIFY COLUMN status ENUM(
        'PENDING_PAYMENT',
        'PREPARING',
        'SHIPPED',
        'DELIVERED',
        'CANCELED',
        'REFUND_REQUESTED',
        'REFUNDED'
    ) NOT NULL DEFAULT 'PENDING_PAYMENT',
    ADD COLUMN tracking_code VARCHAR(100) NULL AFTER zip_code,
    ADD COLUMN refund_reason TEXT NULL AFTER tracking_code,
    ADD COLUMN refund_rejection_reason TEXT NULL AFTER refund_reason;

-- 2. Ampliar ENUM de status do histórico de pedidos para espelhar orders.status
ALTER TABLE order_status_history
    MODIFY COLUMN status ENUM(
        'PENDING_PAYMENT',
        'PREPARING',
        'SHIPPED',
        'DELIVERED',
        'CANCELED',
        'REFUND_REQUESTED',
        'REFUNDED'
    ) NOT NULL;

-- 3. Ampliar ENUM de status da tabela payments e adicionar metadados de Pix, Cartão, Expiração e Estorno
ALTER TABLE payments
    MODIFY COLUMN status ENUM(
        'PENDING',
        'APPROVED',
        'FAILED',
        'EXPIRED',
        'REFUNDED'
    ) NOT NULL DEFAULT 'PENDING',
    ADD COLUMN pix_qr_code TEXT NULL AFTER status,
    ADD COLUMN pix_copy_paste TEXT NULL AFTER pix_qr_code,
    ADD COLUMN card_last4 VARCHAR(4) NULL AFTER pix_copy_paste,
    ADD COLUMN card_brand VARCHAR(30) NULL AFTER card_last4,
    ADD COLUMN expires_at TIMESTAMP NULL AFTER card_brand,
    ADD COLUMN refunded_at TIMESTAMP NULL AFTER paid_at;

-- 4. Índice auxiliar para consultas rápidas de pagamentos por pedido
CREATE INDEX idx_payments_order_id_created ON payments (order_id, created_at DESC);
