package main

import (
	"database/sql"
	"encoding/json"
	"log"
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

type Order struct {
	ID        int       `json:"id"`
	UserID    string    `json:"user_id"`
	Items     []Product `json:"items"`
	Total     float64   `json:"total"`
	CreatedAt time.Time `json:"created_at"`
}

type Server struct {
	db          *sql.DB
	mu          sync.RWMutex
	userCarts   map[string][]Product
	orders      []Order
	nextOrderID int
}

func NewServer(db *sql.DB) *Server {
	return &Server{
		db:          db,
		userCarts:   make(map[string][]Product),
		orders:      make([]Order, 0),
		nextOrderID: 1,
	}
}

// Cria a tabela e popula via arquivo externo se estiver vazia
func setupDatabase(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		price REAL NOT NULL,
		category TEXT NOT NULL
	);`

	if _, err := db.Exec(query); err != nil {
		return err
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM products").Scan(&count); err != nil {
		return err
	}

	// Se não tem produtos, executa o script SQL externo
	if count == 0 {
		seedSQL, err := os.ReadFile("seed.sql")
		if err != nil {
			log.Println("Aviso: seed.sql não encontrado. Banco iniciado sem dados.")
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

// GET /products: Lê estritamente da tabela no SQLite
func (s *Server) handleGetProducts(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query("SELECT id, name, price, category FROM products")
	if err != nil {
		http.Error(w, "Erro ao buscar produtos", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	products := []Product{}
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Category); err != nil {
			http.Error(w, "Erro ao processar dados", http.StatusInternalServerError)
			return
		}
		products = append(products, p)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(products)
}

// POST /cart: Valida se o produto existe direto no banco
func (s *Server) handleAddToCart(w http.ResponseWriter, r *http.Request) {
	var req AddToCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	var p Product
	query := "SELECT id, name, price, category FROM products WHERE id = ?"
	err := s.db.QueryRow(query, req.ProductID).Scan(&p.ID, &p.Name, &p.Price, &p.Category)

	if err == sql.ErrNoRows {
		http.Error(w, "Produto não encontrado", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Erro de banco de dados", http.StatusInternalServerError)
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
		"user_id": userID,
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

func (s *Server) handleCheckout(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	s.mu.Lock()
	defer s.mu.Unlock()

	cartItems, exists := s.userCarts[userID]
	if !exists || len(cartItems) == 0 {
		http.Error(w, "Carrinho vazio", http.StatusBadRequest)
		return
	}

	var total float64
	for _, item := range cartItems {
		total += item.Price
	}

	order := Order{
		ID:        s.nextOrderID,
		UserID:    userID,
		Items:     cartItems,
		Total:     total,
		CreatedAt: time.Now(),
	}
	s.nextOrderID++

	s.orders = append(s.orders, order)
	delete(s.userCarts, userID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(order)
}

func (s *Server) handleGetOrders(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	s.mu.RLock()
	defer s.mu.RUnlock()

	userOrders := make([]Order, 0)
	for _, order := range s.orders {
		if order.UserID == userID {
			userOrders = append(userOrders, order)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(userOrders)
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
		log.Fatalf("Erro ao inicializar schema do banco: %v", err)
	}

	server := NewServer(db)
	mux := http.NewServeMux()

	mux.HandleFunc("GET /products", server.handleGetProducts)
	mux.HandleFunc("POST /cart", server.handleAddToCart)
	mux.HandleFunc("GET /cart", server.handleGetCart)
	mux.HandleFunc("POST /checkout", server.handleCheckout)
	mux.HandleFunc("GET /orders", server.handleGetOrders)

	// 2. Servir arquivos estáticos do front-end (public/index.html, etc.)
	mux.Handle("GET /", http.FileServer(http.Dir("./public")))

	loggedMux := loggingMiddleware(mux)

	log.Println("Servidor rodando em http://localhost:8080...")
	if err := http.ListenAndServe(":8080", loggedMux); err != nil {
		log.Fatal(err)
	}
}