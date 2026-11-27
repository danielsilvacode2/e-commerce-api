-- Migration incremental: Enderecos salvos por usuario (User 1:N Address e Order N:1 Address)
-- Compativel com MySQL 8.0+ (requer migration 004_order_addresses.sql aplicada)
--
-- Permite que um usuario possua multiplas instancias de endereco cadastradas e
-- reutilize enderecos existentes em novos pedidos (sem alterar pedidos historicos).

USE ecommerce;

-- 1. Vincular cada endereco ao usuario proprietario
ALTER TABLE addresses
    ADD COLUMN user_id VARCHAR(50) NULL AFTER id;

-- 2. Preencher user_id dos enderecos existentes a partir dos pedidos associados
UPDATE addresses a
INNER JOIN orders o ON o.address_id = a.id
SET a.user_id = o.user_id
WHERE a.user_id IS NULL;

-- 3. Tornar user_id obrigatorio, adicionar FK e indice de listagem por usuario
ALTER TABLE addresses
    MODIFY COLUMN user_id VARCHAR(50) NOT NULL,
    ADD CONSTRAINT fk_addresses_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    ADD INDEX idx_addresses_user_created (user_id, created_at DESC);

-- 4. Evoluir relacao Order -> Address de 1:1 para N:1 (varios pedidos podem referenciar o mesmo endereco salvo)
--    Adiciona indice nao-unico antes de remover a constraint UNIQUE que sustenta fk_orders_address.
ALTER TABLE orders
    ADD INDEX idx_orders_address_id (address_id),
    DROP INDEX uq_orders_address_id;
