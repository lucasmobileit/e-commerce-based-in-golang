package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type Product struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Category string  `json:"category"`
	Stock    int     `json:"stock"`
}

type CreateProductRequest struct {
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Category string  `json:"category"`
	Stock    int     `json:"stock"`
}

type UpdateStockRequest struct {
	Stock int `json:"stock"`
}

type Coupon struct {
	ID            int     `json:"id"`
	Code          string  `json:"code"`
	DiscountType  string  `json:"discount_type"`
	DiscountValue float64 `json:"discount_value"`
	Active        bool    `json:"active"`
}

type CreateCouponRequest struct {
	Code          string  `json:"code"`
	DiscountType  string  `json:"discount_type"`
	DiscountValue float64 `json:"discount_value"`
}

type UserProfile struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	Role          string `json:"role"`
	AddressStreet string `json:"address_street"`
	AddressNumber string `json:"address_number"`
	AddressCity   string `json:"address_city"`
	AddressZip    string `json:"address_zip"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	AddressStreet string `json:"address_street"`
	AddressNumber string `json:"address_number"`
	AddressCity   string `json:"address_city"`
	AddressZip    string `json:"address_zip"`
}

type AddToCartRequest struct {
	ProductID int `json:"product_id"`
}

type ApplyCouponRequest struct {
	Code string `json:"code"`
}

type CheckoutRequest struct {
	CouponCode string `json:"coupon_code"`
}

type OrderItem struct {
	ID          int     `json:"id"`
	OrderID     int     `json:"order_id"`
	ProductID   int     `json:"product_id"`
	ProductName string  `json:"product_name"`
	UnitPrice   float64 `json:"unit_price"`
}

type Order struct {
	ID            int         `json:"id"`
	UserID        string      `json:"user_id"`
	Subtotal      float64     `json:"subtotal"`
	Discount      float64     `json:"discount"`
	Total         float64     `json:"total"`
	CouponApplied string      `json:"coupon_applied"`
	Status        string      `json:"status"`
	CreatedAt     time.Time   `json:"created_at"`
	Items         []OrderItem `json:"items,omitempty"`
}

type SecurityEvent struct {
	ID        int       `json:"id"`
	EventType string    `json:"event_type"`
	UserID    string    `json:"user_id"`
	IPAddress string    `json:"ip_address"`
	Details   string    `json:"details"`
	Severity  string    `json:"severity"`
	CreatedAt time.Time `json:"created_at"`
}

type CategorySales struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
	Count    int     `json:"count"`
}

type ProductStat struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Stock    int     `json:"stock"`
	SoldQty  int     `json:"sold_qty"`
	Revenue  float64 `json:"revenue"`
}

type TimelinePoint struct {
	Date          string  `json:"date"`
	ApprovedTotal float64 `json:"approved_total"`
	ApprovedCount int     `json:"approved_count"`
	FailedCount   int     `json:"failed_count"`
}

type KPISummary struct {
	GMV             float64         `json:"gmv"`
	AverageOrder    float64         `json:"average_order"`
	CompletedOrders int             `json:"completed_orders"`
	FailedOrders    int             `json:"failed_orders"`
	TotalOrders     int             `json:"total_orders"`
	ConversionRate  float64         `json:"conversion_rate"`
	TotalDiscounts  float64         `json:"total_discounts"`
	ActiveCoupons   int             `json:"active_coupons"`
	RegisteredUsers int             `json:"registered_users"`
	TopCategories   []CategorySales `json:"top_categories"`
	TopProducts     []ProductStat   `json:"top_products"`
	Timeline        []TimelinePoint `json:"timeline"`
}

type KRISummary struct {
	OverallRiskLevel     string          `json:"overall_risk_level"`
	OverallRiskScore     int             `json:"overall_risk_score"`
	PaymentFailureRate   float64         `json:"payment_failure_rate"`
	PaymentFailedValue   float64         `json:"payment_failed_value"`
	BruteForceAttempts   int             `json:"brute_force_attempts"`
	UnauthorizedAttempts int             `json:"unauthorized_attempts"`
	CouponAbuseAttempts  int             `json:"coupon_abuse_attempts"`
	OutOfStockCount      int             `json:"out_of_stock_count"`
	LowStockCount        int             `json:"low_stock_count"`
	CriticalStockItems   []Product       `json:"critical_stock_items"`
	AvgLatencyMs         float64         `json:"avg_latency_ms"`
	Error5xxRate         float64         `json:"error_5xx_rate"`
	RecentEvents         []SecurityEvent `json:"recent_events"`
	EventsByType         map[string]int  `json:"events_by_type"`
}

type ExecutiveMetricsResponse struct {
	Period string     `json:"period"`
	KPI    KPISummary `json:"kpi"`
	KRI    KRISummary `json:"kri"`
}

type Server struct {
	db        *sql.DB
	mu        sync.RWMutex
	userCarts map[string][]Product
}

func NewServer(db *sql.DB) *Server {
	return &Server{
		db:        db,
		userCarts: make(map[string][]Product),
	}
}

func setupDatabase(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		price REAL NOT NULL,
		category TEXT NOT NULL,
		stock INTEGER NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS coupons (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT NOT NULL UNIQUE,
		discount_type TEXT NOT NULL,
		discount_value REAL NOT NULL,
		active INTEGER NOT NULL DEFAULT 1
	);

	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		password TEXT NOT NULL,
		name TEXT NOT NULL,
		email TEXT NOT NULL,
		phone TEXT DEFAULT '',
		role TEXT NOT NULL DEFAULT 'customer',
		address_street TEXT DEFAULT '',
		address_number TEXT DEFAULT '',
		address_city TEXT DEFAULT '',
		address_zip TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS orders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id TEXT NOT NULL,
		subtotal REAL NOT NULL,
		discount REAL NOT NULL DEFAULT 0,
		total REAL NOT NULL,
		coupon_applied TEXT DEFAULT '',
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
	);

	CREATE TABLE IF NOT EXISTS security_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_type TEXT NOT NULL,
		user_id TEXT DEFAULT '',
		ip_address TEXT NOT NULL,
		details TEXT DEFAULT '',
		severity TEXT NOT NULL,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS http_metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		route TEXT NOT NULL,
		method TEXT NOT NULL,
		status_code INTEGER NOT NULL,
		duration_ms REAL NOT NULL,
		ip_address TEXT NOT NULL,
		created_at DATETIME NOT NULL
	);`

	if _, err := db.Exec(schema); err != nil {
		return err
	}

	var productCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM products").Scan(&productCount)

	if productCount == 0 {
		seedSQL, err := os.ReadFile("seed.sql")
		if err != nil {
			log.Println("Aviso: seed.sql não encontrado.")
			return nil
		}
		if _, err := db.Exec(string(seedSQL)); err != nil {
			return err
		}
		log.Println("Banco populado via seed.sql.")
	}

	return nil
}

func getClientIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return strings.TrimSpace(realIP)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func getUserID(r *http.Request) string {
	if cookie, err := r.Cookie("session_user"); err == nil && cookie.Value != "" {
		return strings.ToLower(strings.TrimSpace(cookie.Value))
	}
	if headerUser := r.Header.Get("X-User-ID"); headerUser != "" {
		return strings.ToLower(strings.TrimSpace(headerUser))
	}
	return ""
}

func (s *Server) logSecurityEvent(eventType, userID, ip, details, severity string) {
	go func() {
		_, err := s.db.Exec(
			"INSERT INTO security_events (event_type, user_id, ip_address, details, severity, created_at) VALUES (?, ?, ?, ?, ?, ?)",
			eventType, userID, ip, details, severity, time.Now(),
		)
		if err != nil {
			log.Printf("Aviso: Falha ao registrar evento de segurança: %v", err)
		}
	}()
}

func (s *Server) requireAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
		if userID == "" {
			http.Redirect(w, r, "/login.html", http.StatusSeeOther)
			return
		}

		var exists int
		err := s.db.QueryRow("SELECT 1 FROM users WHERE id = ?", userID).Scan(&exists)
		if err != nil || exists == 0 {
			http.SetCookie(w, &http.Cookie{Name: "session_user", Value: "", Path: "/", MaxAge: -1})
			http.Redirect(w, r, "/login.html", http.StatusSeeOther)
			return
		}

		next(w, r)
	}
}

func (s *Server) adminOnlyMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
		var role string
		err := s.db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&role)
		if err != nil || role != "admin" {
			clientIP := getClientIP(r)
			s.logSecurityEvent(
				"unauthorized_access",
				userID,
				clientIP,
				fmt.Sprintf("Tentativa de acesso não autorizado à rota restrita: %s", r.URL.Path),
				"CRITICAL",
			)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":   "Acesso Negado (403)",
				"message": "Apenas contas administrativas podem acessar este recurso.",
			})
			return
		}
		next(w, r)
	}
}

func processFakePayment(total float64) bool {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return r.Float32() < 0.80
}

// POST /login (Valida username + password)
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	username := strings.ToLower(strings.TrimSpace(req.Username))
	password := strings.TrimSpace(req.Password)

	if username == "" || password == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Usuário e senha são obrigatórios"})
		return
	}

	var user UserProfile
	var dbPass string
	err := s.db.QueryRow("SELECT id, password, name, email, role FROM users WHERE id = ?", username).
		Scan(&user.ID, &dbPass, &user.Name, &user.Email, &user.Role)

	if err == sql.ErrNoRows || dbPass != password {
		clientIP := getClientIP(r)
		var recentFails int
		_ = s.db.QueryRow("SELECT COUNT(*) FROM security_events WHERE event_type = 'login_failed' AND (user_id = ? OR ip_address = ?) AND created_at >= datetime('now', '-10 minutes')", username, clientIP).Scan(&recentFails)

		severity := "WARNING"
		details := fmt.Sprintf("Tentativa de login com credenciais inválidas para '%s'", username)
		if recentFails >= 2 {
			severity = "CRITICAL"
			details = fmt.Sprintf("Alerta de Força Bruta: %d falhas consecutivas para usuário '%s'", recentFails+1, username)
		}
		s.logSecurityEvent("login_failed", username, clientIP, details, severity)

		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "Usuário ou senha incorretos.",
		})
		return
	} else if err != nil {
		http.Error(w, "Erro no banco", http.StatusInternalServerError)
		return
	}

	s.logSecurityEvent("login_success", username, getClientIP(r), "Autenticação bem-sucedida", "INFO")

	http.SetCookie(w, &http.Cookie{
		Name:     "session_user",
		Value:    username,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 7,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Login realizado com sucesso",
		"user":    user,
	})
}

// POST /register
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	req.Username = strings.ToLower(strings.TrimSpace(req.Username))
	req.Password = strings.TrimSpace(req.Password)

	if req.Username == "" || req.Password == "" || req.Name == "" || req.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Usuário, senha, nome e e-mail são obrigatórios"})
		return
	}

	if req.Username == "admin" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Nome de usuário reservado"})
		return
	}

	_, err := s.db.Exec(`
		INSERT INTO users (id, password, name, email, phone, role, address_street, address_number, address_city, address_zip)
		VALUES (?, ?, ?, ?, ?, 'customer', ?, ?, ?, ?)`,
		req.Username, req.Password, req.Name, req.Email, req.Phone,
		req.AddressStreet, req.AddressNumber, req.AddressCity, req.AddressZip,
	)

	if err != nil {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Identificador ou e-mail já cadastrado"})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_user",
		Value:    req.Username,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 7,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Conta criada com sucesso",
		"user":    req,
	})
}

// POST /logout
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_user",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Sessão finalizada"})
}

// GET /auth/me
func (s *Server) handleGetMe(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		http.Error(w, "Não autenticado", http.StatusUnauthorized)
		return
	}

	var u UserProfile
	err := s.db.QueryRow("SELECT id, name, email, phone, role, address_street, address_number, address_city, address_zip FROM users WHERE id = ?", userID).
		Scan(&u.ID, &u.Name, &u.Email, &u.Phone, &u.Role, &u.AddressStreet, &u.AddressNumber, &u.AddressCity, &u.AddressZip)

	if err != nil {
		http.Error(w, "Usuário não encontrado", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(u)
}

func (s *Server) handleServeIndexPage(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		http.Redirect(w, r, "/login.html", http.StatusSeeOther)
		return
	}
	http.ServeFile(w, r, "./public/index.html")
}

func (s *Server) handleServeProfilePage(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		http.Redirect(w, r, "/login.html", http.StatusSeeOther)
		return
	}
	http.ServeFile(w, r, "./public/profile.html")
}

func (s *Server) handleServeAdminPage(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		http.Redirect(w, r, "/login.html?unauthorized=true", http.StatusSeeOther)
		return
	}

	var role string
	err := s.db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&role)
	if err != nil || role != "admin" {
		http.Redirect(w, r, "/login.html?unauthorized=true", http.StatusSeeOther)
		return
	}
	http.ServeFile(w, r, "./public/admin.html")
}

func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	var u UserProfile
	query := "SELECT id, name, email, phone, role, address_street, address_number, address_city, address_zip FROM users WHERE id = ?"
	err := s.db.QueryRow(query, userID).Scan(
		&u.ID, &u.Name, &u.Email, &u.Phone, &u.Role,
		&u.AddressStreet, &u.AddressNumber, &u.AddressCity, &u.AddressZip,
	)

	if err != nil {
		http.Error(w, "Perfil não encontrado", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(u)
}

func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	var req UserProfile
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	query := `
	UPDATE users SET
		name = ?, email = ?, phone = ?, address_street = ?, address_number = ?, address_city = ?, address_zip = ?
	WHERE id = ?`

	_, err := s.db.Exec(query, req.Name, req.Email, req.Phone, req.AddressStreet, req.AddressNumber, req.AddressCity, req.AddressZip, userID)
	if err != nil {
		http.Error(w, "Erro ao salvar perfil no banco", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"message": "Perfil atualizado com sucesso!"})
}

func (s *Server) handleGetProducts(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query("SELECT id, name, price, category, stock FROM products ORDER BY id DESC")
	if err != nil {
		http.Error(w, "Erro ao buscar produtos", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Category, &p.Stock); err != nil {
			http.Error(w, "Erro ao ler produto", http.StatusInternalServerError)
			return
		}
		products = append(products, p)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(products)
}

func (s *Server) handleCreateProduct(w http.ResponseWriter, r *http.Request) {
	var req CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	res, err := s.db.Exec(
		"INSERT INTO products (name, price, category, stock) VALUES (?, ?, ?, ?)",
		req.Name, req.Price, req.Category, req.Stock,
	)
	if err != nil {
		http.Error(w, "Erro ao cadastrar produto", http.StatusInternalServerError)
		return
	}

	id, _ := res.LastInsertId()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(Product{
		ID:       int(id),
		Name:     req.Name,
		Price:    req.Price,
		Category: req.Category,
		Stock:    req.Stock,
	})
}

func (s *Server) handleUpdateStock(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	prodID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	var req UpdateStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	res, err := s.db.Exec("UPDATE products SET stock = ? WHERE id = ?", req.Stock, prodID)
	if err != nil {
		http.Error(w, "Erro ao atualizar estoque", http.StatusInternalServerError)
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		http.Error(w, "Produto não encontrado", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"message": "Estoque atualizado", "id": prodID, "new_stock": req.Stock})
}

func (s *Server) handleAddToCart(w http.ResponseWriter, r *http.Request) {
	var req AddToCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	var p Product
	err := s.db.QueryRow("SELECT id, name, price, category, stock FROM products WHERE id = ?", req.ProductID).
		Scan(&p.ID, &p.Name, &p.Price, &p.Category, &p.Stock)

	if err == sql.ErrNoRows {
		http.Error(w, "Produto não encontrado", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Erro no banco", http.StatusInternalServerError)
		return
	}

	userID := getUserID(r)

	s.mu.Lock()
	defer s.mu.Unlock()

	countInCart := 0
	for _, item := range s.userCarts[userID] {
		if item.ID == p.ID {
			countInCart++
		}
	}

	if countInCart >= p.Stock {
		http.Error(w, "Estoque máximo atingido no carrinho", http.StatusBadRequest)
		return
	}

	s.userCarts[userID] = append(s.userCarts[userID], p)
	currentCart := s.userCarts[userID]

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"message": "Produto adicionado", "cart": currentCart})
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

func (s *Server) handleGetCoupons(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query("SELECT id, code, discount_type, discount_value, active FROM coupons ORDER BY id DESC")
	if err != nil {
		http.Error(w, "Erro ao buscar cupons", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var coupons []Coupon
	for rows.Next() {
		var c Coupon
		var activeInt int
		if err := rows.Scan(&c.ID, &c.Code, &c.DiscountType, &c.DiscountValue, &activeInt); err != nil {
			http.Error(w, "Erro ao processar cupom", http.StatusInternalServerError)
			return
		}
		c.Active = activeInt == 1
		coupons = append(coupons, c)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(coupons)
}

func (s *Server) handleCreateCoupon(w http.ResponseWriter, r *http.Request) {
	var req CreateCouponRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	req.Code = strings.ToUpper(strings.TrimSpace(req.Code))
	res, err := s.db.Exec("INSERT INTO coupons (code, discount_type, discount_value, active) VALUES (?, ?, ?, 1)", req.Code, req.DiscountType, req.DiscountValue)
	if err != nil {
		http.Error(w, "Cupom já existe", http.StatusConflict)
		return
	}

	id, _ := res.LastInsertId()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(Coupon{ID: int(id), Code: req.Code, DiscountType: req.DiscountType, DiscountValue: req.DiscountValue, Active: true})
}

func (s *Server) handleToggleCoupon(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	couponID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	_, err = s.db.Exec("UPDATE coupons SET active = CASE WHEN active = 1 THEN 0 ELSE 1 END WHERE id = ?", couponID)
	if err != nil {
		http.Error(w, "Erro ao alterar cupom", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Status atualizado"})
}

func (s *Server) handleApplyCoupon(w http.ResponseWriter, r *http.Request) {
	var req ApplyCouponRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	code := strings.ToUpper(strings.TrimSpace(req.Code))

	var c Coupon
	var activeInt int
	err := s.db.QueryRow("SELECT id, code, discount_type, discount_value, active FROM coupons WHERE UPPER(code) = ?", code).
		Scan(&c.ID, &c.Code, &c.DiscountType, &c.DiscountValue, &activeInt)

	if err == sql.ErrNoRows || activeInt == 0 {
		s.logSecurityEvent("coupon_invalid", getUserID(r), getClientIP(r), fmt.Sprintf("Tentativa de aplicar cupom inválido ou inativo: '%s'", code), "WARNING")
		http.Error(w, "Cupom inválido", http.StatusNotFound)
		return
	}

	c.Active = (activeInt == 1)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(c)
}

func (s *Server) handleCheckout(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	var req CheckoutRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	s.mu.Lock()
	cartItems, exists := s.userCarts[userID]
	if !exists || len(cartItems) == 0 {
		s.mu.Unlock()
		http.Error(w, "Carrinho vazio", http.StatusBadRequest)
		return
	}
	s.mu.Unlock()

	itemCounts := make(map[int]int)
	var subtotal float64
	for _, item := range cartItems {
		itemCounts[item.ID]++
		subtotal += item.Price
	}

	for prodID, requiredQty := range itemCounts {
		var availableStock int
		var prodName string
		err := s.db.QueryRow("SELECT name, stock FROM products WHERE id = ?", prodID).Scan(&prodName, &availableStock)
		if err != nil || availableStock < requiredQty {
			http.Error(w, "Estoque insuficiente para: "+prodName, http.StatusConflict)
			return
		}
	}

	var discount float64
	appliedCouponCode := ""
	if req.CouponCode != "" {
		code := strings.ToUpper(strings.TrimSpace(req.CouponCode))
		var c Coupon
		var activeInt int
		err := s.db.QueryRow("SELECT id, code, discount_type, discount_value, active FROM coupons WHERE UPPER(code) = ?", code).
			Scan(&c.ID, &c.Code, &c.DiscountType, &c.DiscountValue, &activeInt)

		if err == nil && activeInt == 1 {
			appliedCouponCode = c.Code
			if c.DiscountType == "percentage" {
				discount = subtotal * (c.DiscountValue / 100.0)
			} else if c.DiscountType == "fixed" {
				discount = c.DiscountValue
			}
			if discount > subtotal {
				discount = subtotal
			}
		}
	}

	total := subtotal - discount
	paymentApproved := processFakePayment(total)
	status := "completed"
	if !paymentApproved {
		status = "failed"
	}

	tx, err := s.db.Begin()
	if err != nil {
		http.Error(w, "Erro ao iniciar transação", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	now := time.Now()
	res, err := tx.Exec(
		"INSERT INTO orders (user_id, subtotal, discount, total, coupon_applied, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		userID, subtotal, discount, total, appliedCouponCode, status, now,
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
			http.Error(w, "Erro ao salvar itens", http.StatusInternalServerError)
			return
		}
		recordedItems = append(recordedItems, OrderItem{OrderID: orderID, ProductID: item.ID, ProductName: item.Name, UnitPrice: item.Price})
	}

	if paymentApproved {
		for prodID, qty := range itemCounts {
			_, err := tx.Exec("UPDATE products SET stock = stock - ? WHERE id = ? AND stock >= ?", qty, prodID, qty)
			if err != nil {
				http.Error(w, "Erro ao debitar estoque", http.StatusInternalServerError)
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Erro ao comitar pedido", http.StatusInternalServerError)
		return
	}

	if paymentApproved {
		s.logSecurityEvent("checkout_completed", userID, getClientIP(r), fmt.Sprintf("Pedido #%d aprovado com sucesso no valor de R$ %.2f", orderID, total), "INFO")
		s.mu.Lock()
		delete(s.userCarts, userID)
		s.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message":        "Checkout aprovado!",
			"order_id":       orderID,
			"status":         "completed",
			"subtotal":       subtotal,
			"discount":       discount,
			"total":          total,
			"coupon_applied": appliedCouponCode,
			"items":          recordedItems,
		})
	} else {
		s.logSecurityEvent("payment_failed", userID, getClientIP(r), fmt.Sprintf("Risco de Pagamento: Transação recusada para o pedido #%d no valor de R$ %.2f", orderID, total), "WARNING")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPaymentRequired)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message":  "Pagamento recusado.",
			"order_id": orderID,
			"status":   "failed",
			"total":    total,
		})
	}
}

func (s *Server) handleGetOrders(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	s.fetchOrdersResponse(w, "WHERE user_id = ? ORDER BY id DESC", userID)
}

func (s *Server) handleAdminGetOrders(w http.ResponseWriter, r *http.Request) {
	s.fetchOrdersResponse(w, "ORDER BY id DESC")
}

func (s *Server) fetchOrdersResponse(w http.ResponseWriter, querySuffix string, args ...any) {
	query := "SELECT id, user_id, subtotal, discount, total, coupon_applied, status, created_at FROM orders " + querySuffix
	rows, err := s.db.Query(query, args...)
	if err != nil {
		http.Error(w, "Erro ao buscar pedidos", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		var createdAtStr string
		if err := rows.Scan(&o.ID, &o.UserID, &o.Subtotal, &o.Discount, &o.Total, &o.CouponApplied, &o.Status, &createdAtStr); err != nil {
			http.Error(w, "Erro ao processar pedidos", http.StatusInternalServerError)
			return
		}
		o.CreatedAt, _ = time.Parse("2006-01-02 15:04:05.999999999-07:00", createdAtStr)

		itemRows, err := s.db.Query("SELECT id, order_id, product_id, product_name, unit_price FROM order_items WHERE order_id = ?", o.ID)
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

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (s *Server) telemetryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		duration := time.Since(start)

		path := r.URL.Path
		// Ignora assets estáticos para não poluir o banco de métricas
		if !strings.HasSuffix(path, ".css") && !strings.HasSuffix(path, ".js") && !strings.HasSuffix(path, ".png") && !strings.HasSuffix(path, ".ico") {
			clientIP := getClientIP(r)
			durationMs := float64(duration.Microseconds()) / 1000.0
			go func(route, method string, code int, ms float64, ip string, t time.Time) {
				_, _ = s.db.Exec(
					"INSERT INTO http_metrics (route, method, status_code, duration_ms, ip_address, created_at) VALUES (?, ?, ?, ?, ?, ?)",
					route, method, code, ms, ip, t.Format("2006-01-02 15:04:05"),
				)
			}(path, r.Method, wrapped.statusCode, durationMs, clientIP, start)
		}
		log.Printf("[%s] %s | %d | %s | %v", r.Method, path, wrapped.statusCode, getClientIP(r), duration)
	})
}

func (s *Server) handleServeDashboardPage(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		http.Redirect(w, r, "/login.html?unauthorized=true", http.StatusSeeOther)
		return
	}

	var role string
	err := s.db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&role)
	if err != nil || role != "admin" {
		s.logSecurityEvent("unauthorized_access", userID, getClientIP(r), "Tentativa de abrir /dashboard.html sem perfil de administrador", "CRITICAL")
		http.Redirect(w, r, "/login.html?unauthorized=true", http.StatusSeeOther)
		return
	}
	http.ServeFile(w, r, "./public/dashboard.html")
}

func (s *Server) handleGetAdminMetrics(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "all"
	}

	now := time.Now()
	var startTime time.Time
	hasStart := false

	switch period {
	case "today":
		startTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		hasStart = true
	case "7days":
		startTime = now.AddDate(0, 0, -7)
		hasStart = true
	case "30days":
		startTime = now.AddDate(0, 0, -30)
		hasStart = true
	default:
		hasStart = false
	}

	var startTimeStr string
	if hasStart {
		startTimeStr = startTime.Format("2006-01-02 15:04:05")
	}

	var kpi KPISummary
	var kri KRISummary

	// 1. Consulta de Pedidos (GMV, AOV, Conversão, Descontos)
	queryOrders := "SELECT status, total, discount FROM orders"
	var rows *sql.Rows
	var err error
	if hasStart {
		queryOrders += " WHERE created_at >= ?"
		rows, err = s.db.Query(queryOrders, startTimeStr)
	} else {
		rows, err = s.db.Query(queryOrders)
	}

	if err == nil {
		for rows.Next() {
			var status string
			var total, discount float64
			if err := rows.Scan(&status, &total, &discount); err == nil {
				kpi.TotalOrders++
				if status == "completed" {
					kpi.CompletedOrders++
					kpi.GMV += total
					kpi.TotalDiscounts += discount
				} else if status == "failed" {
					kpi.FailedOrders++
					kri.PaymentFailedValue += total
				}
			}
		}
		rows.Close()
	}

	if kpi.CompletedOrders > 0 {
		kpi.AverageOrder = kpi.GMV / float64(kpi.CompletedOrders)
	}
	if kpi.TotalOrders > 0 {
		kpi.ConversionRate = (float64(kpi.CompletedOrders) / float64(kpi.TotalOrders)) * 100.0
		kri.PaymentFailureRate = (float64(kpi.FailedOrders) / float64(kpi.TotalOrders)) * 100.0
	}

	_ = s.db.QueryRow("SELECT COUNT(*) FROM coupons WHERE active = 1").Scan(&kpi.ActiveCoupons)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'customer'").Scan(&kpi.RegisteredUsers)

	// Top Categorias
	catQuery := `
		SELECT p.category, COALESCE(SUM(oi.unit_price), 0), COUNT(oi.id)
		FROM order_items oi
		JOIN orders o ON oi.order_id = o.id
		JOIN products p ON oi.product_id = p.id
		WHERE o.status = 'completed'`
	if hasStart {
		catQuery += " AND o.created_at >= ? GROUP BY p.category ORDER BY SUM(oi.unit_price) DESC"
		rows, err = s.db.Query(catQuery, startTimeStr)
	} else {
		catQuery += " GROUP BY p.category ORDER BY SUM(oi.unit_price) DESC"
		rows, err = s.db.Query(catQuery)
	}
	if err == nil {
		for rows.Next() {
			var cs CategorySales
			if err := rows.Scan(&cs.Category, &cs.Total, &cs.Count); err == nil {
				kpi.TopCategories = append(kpi.TopCategories, cs)
			}
		}
		rows.Close()
	}

	// Top Produtos
	prodQuery := `
		SELECT p.id, p.name, p.category, p.stock,
		       COALESCE(COUNT(oi.id), 0) as sold_qty,
		       COALESCE(SUM(oi.unit_price), 0) as revenue
		FROM products p
		LEFT JOIN order_items oi ON p.id = oi.product_id
		LEFT JOIN orders o ON oi.order_id = o.id AND o.status = 'completed'
	`
	if hasStart {
		prodQuery += " AND o.created_at >= ? GROUP BY p.id ORDER BY sold_qty DESC, revenue DESC LIMIT 5"
		rows, err = s.db.Query(prodQuery, startTimeStr)
	} else {
		prodQuery += " GROUP BY p.id ORDER BY sold_qty DESC, revenue DESC LIMIT 5"
		rows, err = s.db.Query(prodQuery)
	}
	if err == nil {
		for rows.Next() {
			var ps ProductStat
			if err := rows.Scan(&ps.ID, &ps.Name, &ps.Category, &ps.Stock, &ps.SoldQty, &ps.Revenue); err == nil {
				kpi.TopProducts = append(kpi.TopProducts, ps)
			}
		}
		rows.Close()
	}

	// Timeline temporal de pedidos e receita
	timelineQuery := `
		SELECT substr(created_at, 1, 10) as day,
		       COALESCE(SUM(CASE WHEN status = 'completed' THEN total ELSE 0 END), 0),
		       SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END),
		       SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END)
		FROM orders
	`
	if hasStart {
		timelineQuery += " WHERE created_at >= ? GROUP BY day ORDER BY day ASC"
		rows, err = s.db.Query(timelineQuery, startTimeStr)
	} else {
		timelineQuery += " GROUP BY day ORDER BY day ASC"
		rows, err = s.db.Query(timelineQuery)
	}
	if err == nil {
		for rows.Next() {
			var tp TimelinePoint
			if err := rows.Scan(&tp.Date, &tp.ApprovedTotal, &tp.ApprovedCount, &tp.FailedCount); err == nil {
				kpi.Timeline = append(kpi.Timeline, tp)
			}
		}
		rows.Close()
	}

	// 2. Indicadores de Risco (KRI)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM products WHERE stock = 0").Scan(&kri.OutOfStockCount)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM products WHERE stock > 0 AND stock <= 2").Scan(&kri.LowStockCount)

	critRows, err := s.db.Query("SELECT id, name, price, category, stock FROM products WHERE stock <= 2 ORDER BY stock ASC, id ASC LIMIT 10")
	if err == nil {
		for critRows.Next() {
			var p Product
			if err := critRows.Scan(&p.ID, &p.Name, &p.Price, &p.Category, &p.Stock); err == nil {
				kri.CriticalStockItems = append(kri.CriticalStockItems, p)
			}
		}
		critRows.Close()
	}

	// Eventos de Segurança por Tipo
	kri.EventsByType = make(map[string]int)
	secQuery := "SELECT event_type, COUNT(*) FROM security_events"
	if hasStart {
		secQuery += " WHERE created_at >= ? GROUP BY event_type"
		rows, err = s.db.Query(secQuery, startTimeStr)
	} else {
		secQuery += " GROUP BY event_type"
		rows, err = s.db.Query(secQuery)
	}
	if err == nil {
		for rows.Next() {
			var evType string
			var count int
			if err := rows.Scan(&evType, &count); err == nil {
				kri.EventsByType[evType] = count
				switch evType {
				case "login_failed":
					kri.BruteForceAttempts = count
				case "unauthorized_access":
					kri.UnauthorizedAttempts = count
				case "coupon_invalid":
					kri.CouponAbuseAttempts = count
				}
			}
		}
		rows.Close()
	}

	// Últimos 15 eventos de segurança para o feed de auditoria
	evRows, err := s.db.Query("SELECT id, event_type, user_id, ip_address, details, severity, created_at FROM security_events ORDER BY id DESC LIMIT 15")
	if err == nil {
		for evRows.Next() {
			var se SecurityEvent
			var createdStr string
			if err := evRows.Scan(&se.ID, &se.EventType, &se.UserID, &se.IPAddress, &se.Details, &se.Severity, &createdStr); err == nil {
				if t, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", createdStr); err == nil {
					se.CreatedAt = t
				} else if t, err := time.Parse("2006-01-02 15:04:05", createdStr); err == nil {
					se.CreatedAt = t
				} else if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
					se.CreatedAt = t
				} else {
					se.CreatedAt = time.Now()
				}
				kri.RecentEvents = append(kri.RecentEvents, se)
			}
		}
		evRows.Close()
	}

	// Telemetria de Infraestrutura (Latência Média & Erros 5xx)
	var totalHttp, count5xx int
	if hasStart {
		_ = s.db.QueryRow("SELECT COALESCE(AVG(duration_ms), 0), COUNT(*), COALESCE(SUM(CASE WHEN status_code >= 500 THEN 1 ELSE 0 END), 0) FROM http_metrics WHERE created_at >= ?", startTimeStr).
			Scan(&kri.AvgLatencyMs, &totalHttp, &count5xx)
	} else {
		_ = s.db.QueryRow("SELECT COALESCE(AVG(duration_ms), 0), COUNT(*), COALESCE(SUM(CASE WHEN status_code >= 500 THEN 1 ELSE 0 END), 0) FROM http_metrics").
			Scan(&kri.AvgLatencyMs, &totalHttp, &count5xx)
	}
	if totalHttp > 0 {
		kri.Error5xxRate = (float64(count5xx) / float64(totalHttp)) * 100.0
	}

	// Cálculo da Matriz de Risco Global (Score de 0 a 100)
	riskScore := 0
	if kri.PaymentFailureRate >= 30.0 {
		riskScore += 25
	} else if kri.PaymentFailureRate >= 15.0 {
		riskScore += 12
	}

	if kri.BruteForceAttempts >= 8 {
		riskScore += 30
	} else if kri.BruteForceAttempts >= 3 {
		riskScore += 15
	}

	if kri.UnauthorizedAttempts >= 4 {
		riskScore += 25
	} else if kri.UnauthorizedAttempts >= 1 {
		riskScore += 12
	}

	if kri.OutOfStockCount >= 3 {
		riskScore += 20
	} else if kri.OutOfStockCount >= 1 || kri.LowStockCount >= 2 {
		riskScore += 10
	}

	if kri.Error5xxRate >= 5.0 || kri.AvgLatencyMs >= 400.0 {
		riskScore += 20
	}

	if riskScore > 100 {
		riskScore = 100
	}
	kri.OverallRiskScore = riskScore

	if riskScore >= 60 {
		kri.OverallRiskLevel = "CRÍTICO"
	} else if riskScore >= 35 {
		kri.OverallRiskLevel = "ALTO"
	} else if riskScore >= 15 {
		kri.OverallRiskLevel = "MÉDIO"
	} else {
		kri.OverallRiskLevel = "BAIXO"
	}

	resp := ExecutiveMetricsResponse{
		Period: period,
		KPI:    kpi,
		KRI:    kri,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleSimulateTraffic(w http.ResponseWriter, r *http.Request) {
	productsRows, err := s.db.Query("SELECT id, name, price, stock FROM products WHERE stock > 0")
	var prods []Product
	if err == nil {
		for productsRows.Next() {
			var p Product
			if err := productsRows.Scan(&p.ID, &p.Name, &p.Price, &p.Stock); err == nil {
				prods = append(prods, p)
			}
		}
		productsRows.Close()
	}

	simUsers := []string{"elaine", "lucas", "admin"}
	now := time.Now()

	if len(prods) > 0 {
		// 1. Simular 4 pedidos aprovados
		for i := 0; i < 4; i++ {
			u := simUsers[rand.Intn(len(simUsers))]
			p := prods[rand.Intn(len(prods))]
			subtotal := p.Price
			discount := 0.0
			couponCode := ""
			if i%2 == 0 {
				couponCode = "GO10"
				discount = subtotal * 0.10
			}
			total := subtotal - discount
			orderTime := now.Add(-time.Duration(rand.Intn(120)) * time.Minute)

			res, err := s.db.Exec(
				"INSERT INTO orders (user_id, subtotal, discount, total, coupon_applied, status, created_at) VALUES (?, ?, ?, ?, ?, 'completed', ?)",
				u, subtotal, discount, total, couponCode, orderTime.Format("2006-01-02 15:04:05"),
			)
			if err == nil {
				orderID, _ := res.LastInsertId()
				_, _ = s.db.Exec(
					"INSERT INTO order_items (order_id, product_id, product_name, unit_price) VALUES (?, ?, ?, ?)",
					orderID, p.ID, p.Name, p.Price,
				)
				_, _ = s.db.Exec("UPDATE products SET stock = MAX(stock - 1, 0) WHERE id = ?", p.ID)
			}
		}

		// 2. Simular 2 pedidos com pagamento recusado
		for i := 0; i < 2; i++ {
			u := simUsers[rand.Intn(len(simUsers))]
			p := prods[rand.Intn(len(prods))]
			orderTime := now.Add(-time.Duration(rand.Intn(60)) * time.Minute)
			res, err := s.db.Exec(
				"INSERT INTO orders (user_id, subtotal, discount, total, coupon_applied, status, created_at) VALUES (?, ?, 0, ?, '', 'failed', ?)",
				u, p.Price, p.Price, orderTime.Format("2006-01-02 15:04:05"),
			)
			if err == nil {
				orderID, _ := res.LastInsertId()
				_, _ = s.db.Exec(
					"INSERT INTO order_items (order_id, product_id, product_name, unit_price) VALUES (?, ?, ?, ?)",
					orderID, p.ID, p.Name, p.Price,
				)
				s.logSecurityEvent("payment_failed", u, "177.18.29."+strconv.Itoa(rand.Intn(200)+10), fmt.Sprintf("Simulação: Cartão recusado no pedido #%d", orderID), "WARNING")
			}
		}
	}

	// 3. Simular tentativas de força bruta (Brute Force Attack)
	suspiciousIPs := []string{"185.220.101.5", "194.26.29.112", "45.145.22.8", "103.149.28.19"}
	for i := 0; i < 6; i++ {
		targetUser := "admin"
		if i%3 == 0 {
			targetUser = "root"
		}
		ip := suspiciousIPs[rand.Intn(len(suspiciousIPs))]
		s.logSecurityEvent("login_failed", targetUser, ip, fmt.Sprintf("Ataque de Força Bruta Simulado: senha incorreta testada para '%s'", targetUser), "CRITICAL")
	}

	// 4. Simular tentativa de acesso administrativo não autorizado
	s.logSecurityEvent("unauthorized_access", "lucas", "177.135.2.14", "Simulação: Cliente comum tentou acessar rota restrita /admin.html", "CRITICAL")

	// 5. Simular abuso de cupons inválidos
	s.logSecurityEvent("coupon_invalid", "elaine", "189.40.11.88", "Simulação: Tentativa de aplicação de cupom inexistente 'DESCONTO99'", "WARNING")
	s.logSecurityEvent("coupon_invalid", "lucas", "189.40.11.88", "Simulação: Tentativa com cupom expirado 'PROMOBLACK'", "WARNING")

	// 6. Simular tráfego HTTP para telemetria
	routes := []string{"/products", "/cart", "/login", "/checkout"}
	for i := 0; i < 12; i++ {
		r := routes[rand.Intn(len(routes))]
		status := 200
		if i == 4 {
			status = 401
		} else if i == 9 {
			status = 500
		}
		dur := 18.0 + rand.Float64()*65.0
		_, _ = s.db.Exec(
			"INSERT INTO http_metrics (route, method, status_code, duration_ms, ip_address, created_at) VALUES (?, 'GET', ?, ?, '127.0.0.1', ?)",
			r, status, dur, now.Add(-time.Duration(i*2)*time.Minute).Format("2006-01-02 15:04:05"),
		)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Cenário de teste gerado com sucesso! Vendas, recusas, força bruta e telemetria foram injetados.",
		"status":  "ok",
	})
}

func main() {
	db, err := sql.Open("sqlite", "ecommerce.db")
	if err != nil {
		log.Fatalf("Erro ao abrir banco: %v", err)
	}
	defer db.Close()

	if err := setupDatabase(db); err != nil {
		log.Fatalf("Erro no banco: %v", err)
	}

	server := NewServer(db)
	mux := http.NewServeMux()

	// 1. Acesso Público
	mux.HandleFunc("POST /login", server.handleLogin)
	mux.HandleFunc("POST /register", server.handleRegister)
	mux.HandleFunc("POST /logout", server.handleLogout)
	mux.HandleFunc("GET /login.html", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./public/login.html")
	})
	mux.HandleFunc("GET /style.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./public/style.css")
	})
	mux.HandleFunc("GET /app.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./public/app.js")
	})

	// 2. Proteção Server-side de Páginas
	mux.HandleFunc("GET /{$}", server.handleServeIndexPage)
	mux.HandleFunc("GET /index.html", server.handleServeIndexPage)
	mux.HandleFunc("GET /profile.html", server.handleServeProfilePage)
	mux.HandleFunc("GET /admin.html", server.handleServeAdminPage)
	mux.HandleFunc("GET /dashboard.html", server.handleServeDashboardPage)

	// 3. APIs Protegidas (Clientes)
	mux.HandleFunc("GET /auth/me", server.requireAuthMiddleware(server.handleGetMe))
	mux.HandleFunc("GET /products", server.requireAuthMiddleware(server.handleGetProducts))
	mux.HandleFunc("POST /cart", server.requireAuthMiddleware(server.handleAddToCart))
	mux.HandleFunc("GET /cart", server.requireAuthMiddleware(server.handleGetCart))
	mux.HandleFunc("POST /coupons/apply", server.requireAuthMiddleware(server.handleApplyCoupon))
	mux.HandleFunc("POST /checkout", server.requireAuthMiddleware(server.handleCheckout))
	mux.HandleFunc("GET /orders", server.requireAuthMiddleware(server.handleGetOrders))
	mux.HandleFunc("GET /profile", server.requireAuthMiddleware(server.handleGetProfile))
	mux.HandleFunc("PUT /profile", server.requireAuthMiddleware(server.handleUpdateProfile))

	// 4. APIs Backoffice & Inteligência Executiva (Admin)
	mux.HandleFunc("POST /products", server.adminOnlyMiddleware(server.handleCreateProduct))
	mux.HandleFunc("PATCH /products/{id}/stock", server.adminOnlyMiddleware(server.handleUpdateStock))
	mux.HandleFunc("GET /coupons", server.adminOnlyMiddleware(server.handleGetCoupons))
	mux.HandleFunc("POST /coupons", server.adminOnlyMiddleware(server.handleCreateCoupon))
	mux.HandleFunc("PATCH /coupons/{id}/toggle", server.adminOnlyMiddleware(server.handleToggleCoupon))
	mux.HandleFunc("GET /admin/orders", server.adminOnlyMiddleware(server.handleAdminGetOrders))
	mux.HandleFunc("GET /api/admin/metrics", server.adminOnlyMiddleware(server.handleGetAdminMetrics))
	mux.HandleFunc("POST /api/admin/simulate", server.adminOnlyMiddleware(server.handleSimulateTraffic))

	loggedMux := server.telemetryMiddleware(mux)

	log.Println("Servidor MercadoGO rodando em http://localhost:8080...")
	log.Println("Loja: http://localhost:8080/ | Login: http://localhost:8080/login.html")
	log.Println("Backoffice: http://localhost:8080/admin.html | Dashboard Executivo: http://localhost:8080/dashboard.html")
	if err := http.ListenAndServe(":8080", loggedMux); err != nil {
		log.Fatal(err)
	}
}
