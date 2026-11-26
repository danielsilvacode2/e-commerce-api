-- Migration incremental: Unicidade por (cart_id, product_id) e validacao de quantidade positiva em cart_products
-- Compativel com MySQL 8.0+ (schema base em sql/script.sql)
-- Garante que INSERT ... ON DUPLICATE KEY UPDATE atualize o item existente em vez de criar linhas duplicadas.

USE ecommerce;

-- 1. Normaliza eventuais registros com quantidade invalida (<= 0)
UPDATE cart_products
SET quantity = 1
WHERE quantity < 1;

-- 2. Consolida eventuais linhas duplicadas com o mesmo (cart_id, product_id),
--    preservando a linha mais recente com a maior quantidade e selecao ativa
UPDATE cart_products cp
INNER JOIN (
    SELECT
        cart_id,
        product_id,
        MAX(quantity) AS max_quantity,
        MAX(isSelect) AS any_selected,
        SUBSTRING_INDEX(GROUP_CONCAT(id ORDER BY updated_at DESC, id DESC), ',', 1) AS keep_id
    FROM cart_products
    GROUP BY cart_id, product_id
    HAVING COUNT(*) > 1
) dup ON cp.id = dup.keep_id
SET
    cp.quantity = dup.max_quantity,
    cp.isSelect = dup.any_selected;

DELETE cp
FROM cart_products cp
INNER JOIN (
    SELECT
        cart_id,
        product_id,
        SUBSTRING_INDEX(GROUP_CONCAT(id ORDER BY updated_at DESC, id DESC), ',', 1) AS keep_id
    FROM cart_products
    GROUP BY cart_id, product_id
    HAVING COUNT(*) > 1
) dup
    ON cp.cart_id = dup.cart_id
   AND cp.product_id = dup.product_id
   AND cp.id <> dup.keep_id;

-- 3. Impede novas duplicidades e quantidades menores que 1 no nivel do banco
ALTER TABLE cart_products
    ADD CONSTRAINT uq_cart_products_cart_product UNIQUE (cart_id, product_id),
    ADD CONSTRAINT chk_cart_products_quantity_positive CHECK (quantity > 0);
