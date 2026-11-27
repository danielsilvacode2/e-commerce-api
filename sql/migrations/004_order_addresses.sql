-- Migration incremental: Extracao do endereco do pedido para entidade propria `addresses` (relacao 1:1)
-- Compativel com MySQL 8.0+ (schema base em sql/script.sql + migrations 001..003)
--
-- Estrategia de migracao de dados:
--   SQL puro e deterministico. Para pedidos legados que possuem apenas `zip_code`,
--   e criado um registro dedicado em `addresses` com `cep` normalizado (somente digitos, ate 8 chars)
--   e os demais campos vazios/nulos, preservando created_at/updated_at do pedido.
--   Tradeoff: evita acoplamento de rede/rate-limit com a BrasilAPI durante a migration e
--   garante que pedidos antigos continuem funcionando mesmo com CEPs de teste ou sem numero.

USE ecommerce;

-- 1. Criar tabela addresses
CREATE TABLE IF NOT EXISTS addresses (
    id VARCHAR(50) PRIMARY KEY,
    cep VARCHAR(8) NOT NULL,
    street VARCHAR(200) NOT NULL DEFAULT '',
    number VARCHAR(30) NOT NULL DEFAULT '',
    complement VARCHAR(120) NULL,
    neighborhood VARCHAR(120) NULL,
    city VARCHAR(120) NOT NULL DEFAULT '',
    state CHAR(2) NOT NULL DEFAULT '',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_addresses_cep (cep)
);

-- 2. Adicionar coluna address_id (temporariamente NULL para migracao dos pedidos existentes)
ALTER TABLE orders
    ADD COLUMN address_id VARCHAR(50) NULL AFTER status;

-- 3. Gerar um UUID de endereco dedicado (1:1) para cada pedido existente
UPDATE orders
SET address_id = UUID()
WHERE address_id IS NULL;

-- 4. Migrar o zip_code legado para a nova tabela addresses (somente digitos)
INSERT INTO addresses (
    id,
    cep,
    street,
    number,
    complement,
    neighborhood,
    city,
    state,
    created_at,
    updated_at
)
SELECT
    o.address_id,
    LEFT(REPLACE(REPLACE(TRIM(COALESCE(o.zip_code, '')), '-', ''), ' ', ''), 8),
    '',
    '',
    NULL,
    NULL,
    '',
    '',
    o.created_at,
    o.updated_at
FROM orders o
LEFT JOIN addresses a ON a.id = o.address_id
WHERE a.id IS NULL;

-- 5. Tornar address_id obrigatorio, unico (1:1), adicionar FK e remover zip_code solto
ALTER TABLE orders
    MODIFY COLUMN address_id VARCHAR(50) NOT NULL,
    ADD CONSTRAINT uq_orders_address_id UNIQUE (address_id),
    ADD CONSTRAINT fk_orders_address FOREIGN KEY (address_id) REFERENCES addresses(id) ON DELETE RESTRICT,
    DROP COLUMN zip_code;
