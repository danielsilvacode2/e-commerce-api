package repository

import (
	"database/sql"
	entitiesAddress "e-commerce/core/entities/address"
	entitiesOrder "e-commerce/core/entities/order"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

func readMigrationFile(t *testing.T, relativePath string) string {
	t.Helper()
	fullPath := filepath.Join("..", "..", "..", "..", relativePath)
	raw, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("erro ao ler arquivo de migration %s: %v", fullPath, err)
	}
	return string(raw)
}

func splitSQLStatements(sqlContent string) []string {
	lines := strings.Split(sqlContent, "\n")
	var cleaned []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		cleaned = append(cleaned, line)
	}
	joined := strings.Join(cleaned, "\n")
	parts := strings.Split(joined, ";")
	var stmts []string
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s != "" {
			stmts = append(stmts, s)
		}
	}
	return stmts
}

// simulateLegacyCepNormalization espelha a expressao SQL da migration 004:
// LEFT(REPLACE(REPLACE(TRIM(COALESCE(o.zip_code, '')), '-', ''), ' ', ''), 8)
func simulateLegacyCepNormalization(rawZipCode string) string {
	cleaned := strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(rawZipCode), "-", ""), " ", "")
	if len(cleaned) > 8 {
		return cleaned[:8]
	}
	return cleaned
}

func TestMigration004OrderAddressesStructureAndDataRules(t *testing.T) {
	content := readMigrationFile(t, filepath.Join("sql", "migrations", "004_order_addresses.sql"))
	stmts := splitSQLStatements(content)

	expectedSnippets := []string{
		"CREATE TABLE IF NOT EXISTS addresses",
		"cep VARCHAR(8) NOT NULL",
		"street VARCHAR(200) NOT NULL DEFAULT ''",
		"number VARCHAR(30) NOT NULL DEFAULT ''",
		"complement VARCHAR(120) NULL",
		"neighborhood VARCHAR(120) NULL",
		"city VARCHAR(120) NOT NULL DEFAULT ''",
		"state CHAR(2) NOT NULL DEFAULT ''",
		"ADD COLUMN address_id VARCHAR(50) NULL",
		"SET address_id = UUID()",
		"INSERT INTO addresses",
		"MODIFY COLUMN address_id VARCHAR(50) NOT NULL",
		"UNIQUE (address_id)",
		"FOREIGN KEY (address_id) REFERENCES addresses(id)",
		"DROP COLUMN zip_code",
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(content, snippet) {
			t.Errorf("migration 004 deve conter trecho obrigatorio %q", snippet)
		}
	}

	// Garante a ordem critica: criar tabela -> adicionar address_id -> preencher UUID -> migrar dados -> aplicar NOT NULL/FK e remover zip_code
	posCreate := strings.Index(content, "CREATE TABLE IF NOT EXISTS addresses")
	posAddCol := strings.Index(content, "ADD COLUMN address_id")
	posUpdateUUID := strings.Index(content, "SET address_id = UUID()")
	posInsertData := strings.Index(content, "INSERT INTO addresses")
	posDropZip := strings.Index(content, "DROP COLUMN zip_code")

	if !(posCreate < posAddCol && posAddCol < posUpdateUUID && posUpdateUUID < posInsertData && posInsertData < posDropZip) {
		t.Fatalf("ordem das operacoes da migration 004 invalida: create=%d add=%d update=%d insert=%d drop=%d",
			posCreate, posAddCol, posUpdateUUID, posInsertData, posDropZip)
	}

	if len(stmts) < 5 {
		t.Fatalf("esperava ao menos 5 instrucoes SQL na migration 004, obteve %d", len(stmts))
	}

	// Valida a regra de transformacao dos dados legados (incluindo CEPs duplicados entre pedidos distintos)
	type legacyOrder struct {
		id        string
		zipCode   string
		createdAt time.Time
		updatedAt time.Time
	}
	now := time.Now().UTC().Truncate(time.Second)
	legacyOrders := []legacyOrder{
		{id: uuid.NewString(), zipCode: "01001-000", createdAt: now.Add(-2 * time.Hour), updatedAt: now.Add(-1 * time.Hour)},
		{id: uuid.NewString(), zipCode: "01001-000", createdAt: now.Add(-1 * time.Hour), updatedAt: now},
		{id: uuid.NewString(), zipCode: "20040020", createdAt: now, updatedAt: now},
		{id: uuid.NewString(), zipCode: " 30130-010 ", createdAt: now, updatedAt: now},
	}

	migratedAddresses := make(map[string]entitiesAddress.Address)
	orderToAddressID := make(map[string]string)

	for _, lo := range legacyOrders {
		addrID := uuid.New()
		orderToAddressID[lo.id] = addrID.String()
		migratedAddresses[addrID.String()] = entitiesAddress.Address{
			Id:       addrID,
			Cep:      simulateLegacyCepNormalization(lo.zipCode),
			Street:   "",
			Number:   "",
			City:     "",
			State:    "",
			CreateAt: lo.createdAt,
			UpdateAt: lo.updatedAt,
		}
	}

	if len(migratedAddresses) != len(legacyOrders) {
		t.Fatalf("cada pedido legado deve possuir um endereco 1:1 dedicado: pedidos=%d enderecos=%d",
			len(legacyOrders), len(migratedAddresses))
	}

	// Dois pedidos legados com o mesmo CEP devem receber registros distintos (1:1)
	if orderToAddressID[legacyOrders[0].id] == orderToAddressID[legacyOrders[1].id] {
		t.Fatal("pedidos distintos com o mesmo CEP nao podem compartilhar o mesmo address_id em relacao 1:1")
	}

	wantCeps := []string{"01001000", "01001000", "20040020", "30130010"}
	for i, lo := range legacyOrders {
		addr := migratedAddresses[orderToAddressID[lo.id]]
		if addr.Cep != wantCeps[i] {
			t.Errorf("pedido %d: cep migrado = %q, esperado %q", i, addr.Cep, wantCeps[i])
		}
		if addr.Street != "" || addr.Number != "" || addr.City != "" || addr.State != "" {
			t.Errorf("pedido %d: campos legados desconhecidos devem permanecer vazios, obteve %+v", i, addr)
		}
	}
}

// TestMigration004OrderAddressesMySQL executa a migration real contra um banco MySQL temporario
// quando a variavel TEST_MYSQL_DSN estiver configurada (ex.: root:root@tcp(127.0.0.1:3306)/?parseTime=true&multiStatements=true).
func TestMigration004OrderAddressesMySQL(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("TEST_MYSQL_DSN"))
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN nao configurada; pulando teste de integracao MySQL da migration 004")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("erro ao abrir conexao MySQL: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("erro no ping MySQL: %v", err)
	}

	dbName := "ecommerce_mig_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	if _, err := db.Exec("CREATE DATABASE " + dbName); err != nil {
		t.Fatalf("erro ao criar banco temporario %s: %v", dbName, err)
	}
	defer func() {
		_, _ = db.Exec("DROP DATABASE IF EXISTS " + dbName)
	}()

	if _, err := db.Exec("USE " + dbName); err != nil {
		t.Fatalf("erro ao selecionar banco temporario: %v", err)
	}

	// Cria schema legado minimo compativel com script.sql + migration 001
	legacyDDL := []string{
		`CREATE TABLE users (
			id VARCHAR(50) PRIMARY KEY,
			user_name VARCHAR(100) NOT NULL,
			user_email VARCHAR(100) NOT NULL UNIQUE,
			user_password VARCHAR(255) NOT NULL,
			user_role ENUM('admin', 'user') NOT NULL DEFAULT 'user',
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE products (
			id VARCHAR(50) PRIMARY KEY,
			product_name VARCHAR(100) NOT NULL,
			product_description VARCHAR(100) NOT NULL,
			product_sku VARCHAR(255) NOT NULL UNIQUE,
			product_price DECIMAL(10, 2) NOT NULL,
			product_image VARCHAR(200) NOT NULL,
			product_quantity INT NOT NULL,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE orders (
			id VARCHAR(50) PRIMARY KEY,
			user_id VARCHAR(50) NOT NULL,
			status ENUM('PENDING_PAYMENT','PREPARING','SHIPPED','DELIVERED','CANCELED','REFUND_REQUESTED','REFUNDED') NOT NULL DEFAULT 'PENDING_PAYMENT',
			zip_code VARCHAR(9) NOT NULL,
			tracking_code VARCHAR(100) NULL,
			refund_reason TEXT NULL,
			refund_rejection_reason TEXT NULL,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT
		)`,
		`CREATE TABLE order_items (
			id VARCHAR(50) PRIMARY KEY,
			order_id VARCHAR(50) NOT NULL,
			product_id VARCHAR(50) NOT NULL,
			quantity INT NOT NULL CHECK (quantity > 0),
			unit_price DECIMAL(10, 2) NOT NULL,
			FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
			FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE RESTRICT
		)`,
		`CREATE TABLE order_status_history (
			id VARCHAR(50) PRIMARY KEY,
			order_id VARCHAR(50) NOT NULL,
			status ENUM('PENDING_PAYMENT','PREPARING','SHIPPED','DELIVERED','CANCELED','REFUND_REQUESTED','REFUNDED') NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
		)`,
	}

	for _, stmt := range legacyDDL {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("erro ao aplicar schema legado: %v", err)
		}
	}

	userID := uuid.New()
	order1ID := uuid.New()
	order2ID := uuid.New()

	if _, err := db.Exec(`INSERT INTO users (id, user_name, user_email, user_password) VALUES (?, 'Cliente Teste', 'cliente@teste.com', 'hash')`, userID); err != nil {
		t.Fatalf("erro ao inserir usuario: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO orders (id, user_id, status, zip_code) VALUES (?, ?, 'DELIVERED', '01001-000')`, order1ID, userID); err != nil {
		t.Fatalf("erro ao inserir pedido legado 1: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO orders (id, user_id, status, zip_code) VALUES (?, ?, 'PREPARING', '20040020')`, order2ID, userID); err != nil {
		t.Fatalf("erro ao inserir pedido legado 2: %v", err)
	}

	// Executa migration 004 e 005 no banco temporario (substituindo USE ecommerce pelo banco temporario)
	for _, migPath := range []string{
		filepath.Join("sql", "migrations", "004_order_addresses.sql"),
		filepath.Join("sql", "migrations", "005_user_saved_addresses.sql"),
	} {
		migSQL := readMigrationFile(t, migPath)
		for _, stmt := range splitSQLStatements(migSQL) {
			if strings.HasPrefix(strings.ToUpper(stmt), "USE ") {
				continue
			}
			if _, err := db.Exec(stmt); err != nil {
				t.Fatalf("erro ao executar statement de %s (%s): %v", migPath, stmt, err)
			}
		}
	}

	// Verifica leitura dos pedidos legados pelo OrderRepository atualizado
	repo := NewOrderRepository(db)

	o1, err := repo.ReadOrderById(order1ID.String())
	if err != nil {
		t.Fatalf("erro ao ler pedido legado 1 apos migration: %v", err)
	}
	if o1.Address == nil || o1.Address.Cep != "01001000" || o1.Address.Street != "" || o1.Address.UserId != userID {
		t.Fatalf("endereco migrado inesperado para pedido 1: %+v", o1.Address)
	}

	o2, err := repo.ReadOrderById(order2ID.String())
	if err != nil {
		t.Fatalf("erro ao ler pedido legado 2 apos migration: %v", err)
	}
	if o2.Address == nil || o2.Address.Cep != "20040020" || o2.Address.Id == o1.Address.Id || o2.Address.UserId != userID {
		t.Fatalf("endereco migrado inesperado para pedido 2: %+v", o2.Address)
	}

	// Verifica atualizacao de endereco criando nova instancia e vinculando ao pedido 2
	newAddrID := uuid.New()
	err = repo.UpdateOrderAddress(entitiesAddress.Address{
		Id:           newAddrID,
		UserId:       userID,
		Cep:          "20040020",
		Street:       "Rua da Assembleia",
		Number:       "10",
		Complement:   "Sala 5",
		Neighborhood: "Centro",
		City:         "Rio de Janeiro",
		State:        "RJ",
	}, order2ID.String(), false)
	if err != nil {
		t.Fatalf("erro ao atualizar endereco de pedido migrado: %v", err)
	}

	o2Updated, err := repo.ReadOrderById(order2ID.String())
	if err != nil {
		t.Fatalf("erro ao reler pedido 2 atualizado: %v", err)
	}
	if o2Updated.Address.Id != newAddrID || o2Updated.Address.Street != "Rua da Assembleia" || o2Updated.Address.Number != "10" || o2Updated.Address.State != "RJ" {
		t.Fatalf("endereco atualizado inesperado: %+v", o2Updated.Address)
	}

	// Verifica reutilizacao do mesmo endereco em outro pedido (N:1)
	err = repo.UpdateOrderAddress(entitiesAddress.Address{Id: newAddrID}, order1ID.String(), true)
	if err != nil {
		t.Fatalf("erro ao reutilizar endereco salvo em relacao N:1: %v", err)
	}

	// Verifica listagem paginada
	paginated, err := repo.ReadAllOrders(entitiesOrder.OrderFilter{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("erro em ReadAllOrders apos migration: %v", err)
	}
	if len(paginated.Data) != 2 {
		t.Fatalf("esperava 2 pedidos, obteve %d", len(paginated.Data))
	}
	fmt.Printf("[INFO] Migrations 004 e 005 validadas com sucesso no MySQL temporario %s\n", dbName)
}

func assertSingleOpAlterTables(t *testing.T, migrationName string, stmts []string) {
	t.Helper()
	for _, stmt := range stmts {
		upper := strings.ToUpper(stmt)
		if !strings.HasPrefix(upper, "ALTER TABLE ") {
			continue
		}
		// Evita multiplas operacoes no mesmo ALTER TABLE (erro TiDB 8200: Unsupported operate same column)
		ops := 0
		for _, keyword := range []string{
			"ADD COLUMN",
			"MODIFY COLUMN",
			"CHANGE COLUMN",
			"DROP COLUMN",
			"ADD CONSTRAINT",
			"ADD INDEX",
			"DROP INDEX",
			"DROP FOREIGN KEY",
		} {
			ops += strings.Count(upper, keyword)
		}
		if ops > 1 {
			t.Errorf("%s contem ALTER TABLE com multiplas operacoes (%d) — divida em instrucoes separadas para compatibilidade com TiDB/MySQL:\n%s",
				migrationName, ops, stmt)
		}
	}
}

func TestMigration005UserSavedAddressesStructure(t *testing.T) {
	content004 := readMigrationFile(t, filepath.Join("sql", "migrations", "004_order_addresses.sql"))
	assertSingleOpAlterTables(t, "004_order_addresses.sql", splitSQLStatements(content004))

	content := readMigrationFile(t, filepath.Join("sql", "migrations", "005_user_saved_addresses.sql"))
	assertSingleOpAlterTables(t, "005_user_saved_addresses.sql", splitSQLStatements(content))

	for _, snippet := range []string{
		"ADD COLUMN user_id VARCHAR(50) NULL",
		"SET a.user_id = o.user_id",
		"MODIFY COLUMN user_id VARCHAR(50) NOT NULL",
		"FOREIGN KEY (user_id) REFERENCES users(id)",
		"ADD INDEX idx_orders_address_id (address_id)",
		"DROP INDEX uq_orders_address_id",
	} {
		if !strings.Contains(content, snippet) {
			t.Errorf("migration 005 deve conter trecho obrigatorio %q", snippet)
		}
	}
}
