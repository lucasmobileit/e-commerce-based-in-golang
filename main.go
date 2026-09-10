package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type Product struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Category string  `json:"category"`
}

type AddToCartRequest struct {
	ProductID int `json:"product_id"`
}

type OrderItem struct {
	ID          int     `json:"id"`
	OrderID     int     `json:"order_id"`
	ProductID   int     `json:"product_id"`
	ProductName string  `json:"product_name"`
	UnitPrice   float64 `json:"unit_price"`
}

type Order struct {
	ID        int         `json:"id"`
	UserID    string      `json:"user_id"`
	Total     float64     `json:"total"`
	Status    string      `json:"status"` // "completed" ou "failed"
	CreatedAt time.Time   `json:"created_at"`
	Items     []OrderItem `json:"items,omitempty"`
}

type Server struct {
	db        *sql.DB
	mu        sync.RWMutex
	userCarts map[string][]Product
}

type MetricsResponse struct {
	TotalOrders    int     `json:"total_orders"`
	CompletedSales int     `json:"completed_sales"`
	FailedSales    int     `json:"failed_sales"`
	Revenue        float64 `json:"revenue"`
	AverageTicket  float64 `json:"average_ticket"`
	ConversionRate float64 `json:"conversion_rate"`
}

func (s *Server) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	var m MetricsResponse

	// Query única consolidada para extrair todas as métricas em uma única viagem ao banco
	query := `
	SELECT 
		COUNT(*),
		COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = 'completed' THEN total ELSE 0 END), 0),
		COALESCE(AVG(CASE WHEN status = 'completed' THEN total ELSE NULL END), 0)
	FROM orders;
	`

	err := s.db.QueryRow(query).Scan(
		&m.TotalOrders,
		&m.CompletedSales,
		&m.FailedSales,
		&m.Revenue,
		&m.AverageTicket,
	)
	if err != nil {
		http.Error(w, "Erro ao calcular métricas", http.StatusInternalServerError)
		return
	}

	if m.TotalOrders > 0 {
		m.ConversionRate = (float64(m.CompletedSales) / float64(m.TotalOrders)) * 100
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(m)
}

func NewServer(db *sql.DB) *Server {
	return &Server{
		db:        db,
		userCarts: make(map[string][]Product),
	}
}

// Cria tabelas de produtos, pedidos e itens de pedidos
func setupDatabase(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		price REAL NOT NULL,
		category TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS orders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id TEXT NOT NULL,
		total REAL NOT NULL,
		status TEXT NOT NULL,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS order_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		order_id INTEGER NOT NULL,
		product_id INTEGER NOT NULL,
		product_name TEXT NOT NULL,
		unit_price REAL NOT NULL,
		FOREIGN KEY (order_id) REFERENCES orders(id)
	);`

	if _, err := db.Exec(schema); err != nil {
		return err
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM products").Scan(&count); err != nil {
		return err
	}

	if count == 0 {
		seedSQL, err := os.ReadFile("seed.sql")
		if err != nil {
			log.Println("Aviso: seed.sql não encontrado. Banco iniciado sem produtos.")
			return nil
		}
		if _, err := db.Exec(string(seedSQL)); err != nil {
			return err
		}
		log.Println("Banco populado com sucesso a partir do seed.sql.")
	}

	return nil
}

func getUserID(r *http.Request) string {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		return "guest-default"
	}
	return userID
}

// Simulação de Gateway de Pagamento (ex: Stripe, Cielo, Mercado Pago)
// 80% de chance de aprovação, 20% de rejeição (cartão recusado, timeout, etc.)
func processFakePayment(total float64) bool {
	// Semente de tempo para variar a cada requisição
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return r.Float32() < 0.80
}

func (s *Server) handleGetProducts(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query("SELECT id, name, price, category FROM products")
	if err != nil {
		http.Error(w, "Erro ao buscar produtos", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Category); err != nil {
			http.Error(w, "Erro ao ler produto", http.StatusInternalServerError)
			return
		}
		products = append(products, p)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(products)
}

func (s *Server) handleAddToCart(w http.ResponseWriter, r *http.Request) {
	var req AddToCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	var p Product
	err := s.db.QueryRow("SELECT id, name, price, category FROM products WHERE id = ?", req.ProductID).
		Scan(&p.ID, &p.Name, &p.Price, &p.Category)

	if err == sql.ErrNoRows {
		http.Error(w, "Produto não encontrado", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Erro no banco", http.StatusInternalServerError)
		return
	}

	userID := getUserID(r)

	s.mu.Lock()
	s.userCarts[userID] = append(s.userCarts[userID], p)
	currentCart := s.userCarts[userID]
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Produto adicionado",
		"cart":    currentCart,
	})
}

func (s *Server) handleGetCart(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	s.mu.RLock()
	userItems := s.userCarts[userID]
	s.mu.RUnlock()

	var total float64
	for _, item := range userItems {
		total += item.Price
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"user_id": userID,
		"count":   len(userItems),
		"total":   total,
		"items":   userItems,
	})
}

// POST /checkout com Pagamento Fake e Transação ACID
func (s *Server) handleCheckout(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	s.mu.Lock()
	cartItems, exists := s.userCarts[userID]
	if !exists || len(cartItems) == 0 {
		s.mu.Unlock()
		http.Error(w, "Carrinho vazio", http.StatusBadRequest)
		return
	}
	s.mu.Unlock()

	// 1. Calcula o total da transação
	var total float64
	for _, item := range cartItems {
		total += item.Price
	}

	// 2. Gateway de Pagamento Fake
	paymentApproved := processFakePayment(total)
	status := "completed"
	if !paymentApproved {
		status = "failed"
	}

	// 3. Persistência atômica no SQLite via Transação (BEGIN TRANSACTION)
	tx, err := s.db.Begin()
	if err != nil {
		http.Error(w, "Erro ao iniciar transação", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback() // Seguro: se já houve Commit(), o Rollback é no-op

	now := time.Now()
	res, err := tx.Exec(
		"INSERT INTO orders (user_id, total, status, created_at) VALUES (?, ?, ?, ?)",
		userID, total, status, now,
	)
	if err != nil {
		http.Error(w, "Erro ao criar pedido", http.StatusInternalServerError)
		return
	}

	orderID64, _ := res.LastInsertId()
	orderID := int(orderID64)

	var recordedItems []OrderItem
	for _, item := range cartItems {
		_, err := tx.Exec(
			"INSERT INTO order_items (order_id, product_id, product_name, unit_price) VALUES (?, ?, ?, ?)",
			orderID, item.ID, item.Name, item.Price,
		)
		if err != nil {
			http.Error(w, "Erro ao salvar itens do pedido", http.StatusInternalServerError)
			return
		}
		recordedItems = append(recordedItems, OrderItem{
			OrderID:     orderID,
			ProductID:   item.ID,
			ProductName: item.Name,
			UnitPrice:   item.Price,
		})
	}

	// Comita no banco de dados de fato
	if err := tx.Commit(); err != nil {
		http.Error(w, "Erro ao comitar pedido", http.StatusInternalServerError)
		return
	}

	// 4. Regra de Negócio: Esvazia o carrinho APENAS se o pagamento foi aprovado
	if paymentApproved {
		s.mu.Lock()
		delete(s.userCarts, userID)
		s.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message":  "Checkout finalizado com sucesso!",
			"order_id": orderID,
			"status":   "completed",
			"total":    total,
			"items":    recordedItems,
		})
	} else {
		// Retorna 402 Payment Required para sinalizar falha financeira
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPaymentRequired)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message":  "Pagamento recusado pela operadora. O pedido foi registrado como failed e os itens continuam no seu carrinho.",
			"order_id": orderID,
			"status":   "failed",
			"total":    total,
		})
	}
}

// GET /orders: Busca histórico direto do banco com os itens relacionados
func (s *Server) handleGetOrders(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	rows, err := s.db.Query(
		"SELECT id, user_id, total, status, created_at FROM orders WHERE user_id = ? ORDER BY id DESC",
		userID,
	)
	if err != nil {
		http.Error(w, "Erro ao buscar pedidos", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		var createdAtStr string
		if err := rows.Scan(&o.ID, &o.UserID, &o.Total, &o.Status, &createdAtStr); err != nil {
			http.Error(w, "Erro ao processar pedidos", http.StatusInternalServerError)
			return
		}
		o.CreatedAt, _ = time.Parse("2006-01-02 15:04:05.999999999-07:00", createdAtStr)

		// Busca os itens específicos deste pedido
		itemRows, err := s.db.Query(
			"SELECT id, order_id, product_id, product_name, unit_price FROM order_items WHERE order_id = ?",
			o.ID,
		)
		if err == nil {
			for itemRows.Next() {
				var it OrderItem
				_ = itemRows.Scan(&it.ID, &it.OrderID, &it.ProductID, &it.ProductName, &it.UnitPrice)
				o.Items = append(o.Items, it)
			}
			itemRows.Close()
		}

		orders = append(orders, o)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(orders)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s | %s | %v", r.Method, r.URL.Path, r.RemoteAddr, time.Since(start))
	})
}

func main() {
	db, err := sql.Open("sqlite", "ecommerce.db")
	if err != nil {
		log.Fatalf("Erro ao abrir banco: %v", err)
	}
	defer db.Close()

	if err := setupDatabase(db); err != nil {
		log.Fatalf("Erro no schema do banco: %v", err)
	}

	server := NewServer(db)
	mux := http.NewServeMux()

	mux.HandleFunc("GET /products", server.handleGetProducts)
	mux.HandleFunc("POST /cart", server.handleAddToCart)
	mux.HandleFunc("GET /cart", server.handleGetCart)
	mux.HandleFunc("POST /checkout", server.handleCheckout)
	mux.HandleFunc("GET /orders", server.handleGetOrders)
	mux.HandleFunc("GET /metrics", server.handleGetMetrics)

	mux.Handle("GET /", http.FileServer(http.Dir("./public")))

	loggedMux := loggingMiddleware(mux)

	log.Println("Servidor rodando em http://localhost:8080...")
	if err := http.ListenAndServe(":8080", loggedMux); err != nil {
		log.Fatal(err)
	}
}
