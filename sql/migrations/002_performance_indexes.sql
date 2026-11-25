-- Migration incremental: Indices de performance para listagens, ordenacao e consultas frequentes
-- Compativel com MySQL 8.0+ (schema base em sql/script.sql e sql/migrations/001_payment_refund_flow.sql)
-- No DBeaver, execute o arquivo completo como script (Alt+X) ou um bloco ALTER TABLE por vez (Ctrl+Enter).

USE ecommerce;

-- 1. Listagem paginada e ordenacao de produtos (vitrine e busca)
ALTER TABLE products
    ADD INDEX idx_products_created_id (created_at DESC, id DESC),
    ADD INDEX idx_products_name_id (product_name, id),
    ADD FULLTEXT INDEX idx_products_fulltext (product_name, product_description);

-- 2. Listagem de pedidos por usuario e por status sem filesort (Meus Pedidos e Painel Admin)
ALTER TABLE orders
    ADD INDEX idx_orders_user_status_created (user_id, status, created_at DESC),
    ADD INDEX idx_orders_user_created (user_id, created_at DESC),
    ADD INDEX idx_orders_status_created (status, created_at DESC),
    ADD INDEX idx_orders_created (created_at DESC);

-- 3. Historico cronologico de status de um pedido
ALTER TABLE order_status_history
    ADD INDEX idx_order_status_history_order_created (order_id, created_at ASC);

-- 4. Busca rapida de itens do carrinho por (cart_id, product_id)
ALTER TABLE cart_products
    ADD INDEX idx_cart_products_cart_product (cart_id, product_id);

-- 5. Listagem ordenada de usuarios no painel admin
ALTER TABLE users
    ADD INDEX idx_users_created (created_at DESC);
