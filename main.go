package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB

var jwtKey = []byte(os.Getenv("JWT_SECRET"))

type Product struct {
	ID              int      `json:"id"`
	Name            string   `json:"name"`
	Price           string   `json:"price"`
	Color           string   `json:"color"`
	WidthCm         int      `json:"width_cm"`
	HeightCm        int      `json:"height_cm"`
	WeightG         int      `json:"weight_g"`
	ImageURL        *string  `json:"image_url,omitempty"`
	Description     *string  `json:"description,omitempty"`
	QuantityInStock int      `json:"quantity_in_stock"`
	Categories      []string `json:"categories"`
}

type User struct {
	ID             int     `json:"id"`
	FirstName      string  `json:"first_name"`
	LastName       string  `json:"last_name"`
	Email          string  `json:"email"`
	Phone          string  `json:"phone"`
	ProfilePicture *string `json:"profile_picture_url,omitempty"`
	RegistrationTs string  `json:"registration_ts"`
}

type Credentials struct {
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Email     string `json:"email"`
	Phone     string `json:"phone,omitempty"`
	Password  string `json:"password"`
}

type Claims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

type CartItem struct {
	CartItemID int     `json:"cart_item_id"`
	Product    Product `json:"product"`
	Quantity   int     `json:"quantity"`
}

type CartRequest struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

type CheckoutRequest struct {
	// Список позиций: product_id и количество
	Items []CartRequest `json:"items"`
}

type PaymentCard struct {
	CardID         int    `json:"card_id"`
	UserID         int    `json:"user_id"`
	CardholderName string `json:"cardholder_name"`
	CardNumber     string `json:"card_number"`
	ExpMonth       int    `json:"exp_month"`
	ExpYear        int    `json:"exp_year"`
}

type PasswordChangeRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type OrderItem struct {
	ProductID   int64   `json:"product_id"`
	Quantity    int     `json:"quantity"`
	ProductName string  `json:"product_name"`
	Price       float64 `json:"price"`
}

type OrderSummary struct {
	OrderID     int         `json:"order_id"`
	Status      string      `json:"status"`
	OrderTS     time.Time   `json:"order_ts"`
	TotalItems  int         `json:"total_items"`
	TotalAmount string      `json:"total_amount"`
	UserID      int64       `json:"user_id"`
	Items       []OrderItem `json:"items"`
}

func main() {
	if len(jwtKey) == 0 {
		log.Fatal("JWT_SECRET is not set in environment")
	}

	// ===== 1) Подключение к БД =====
	connStr := `host=localhost port=5432 user=postgres password=2006Hjvfy! dbname=playboxdb sslmode=disable`
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Не удалось открыть подключение к БД: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Не удалось пинговать БД: %v", err)
	}
	log.Println("Успешно подключились к БД")

	// ===== 2) Регистрация эндпойнтов =====
	http.HandleFunc("/ping", pingHandler)
	http.HandleFunc("/products", productsHandler)
	http.HandleFunc("/users/", userHandler)
	http.HandleFunc("/users/password", authMiddleware(changePasswordHandler))
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/login", loginHandler)

	http.HandleFunc("/cart", authMiddleware(cartHandler))           // GET /cart
	http.HandleFunc("/cart/items", authMiddleware(addOrUpdateItem)) // POST /cart/items
	http.HandleFunc("/cart/items/", authMiddleware(removeItem))     // DELETE /cart/items/{product_id}

	http.HandleFunc("/cards", authMiddleware(cardsHandler)) // GET, POST
	http.HandleFunc("/cards/", authMiddleware(deleteCard))  // DELETE

	http.HandleFunc("/checkout", authMiddleware(checkoutHandler))
	http.HandleFunc("/orders", authMiddleware(listOrdersHandler))

	// ===== 3) Запуск HTTP-сервера =====
	log.Println("Сервер запущен на порту 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func checkoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := getUserID(r)

	var req CheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	if len(req.Items) == 0 {
		http.Error(w, "No items", http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// 1) Вставляем заказ
	var orderID int
	err = tx.QueryRow(`
        INSERT INTO orders (user_id, status_id)
        VALUES ($1,
            (SELECT status_id FROM order_statuses WHERE status_name='Новый')
        )
        RETURNING order_id
    `, userID).Scan(&orderID)
	if err != nil {
		http.Error(w, "DB error creating order", http.StatusInternalServerError)
		return
	}

	// 2) Обрабатываем каждую позицию
	for _, it := range req.Items {
		// 2.1) Уменьшаем остаток
		if _, err := tx.Exec(`
            UPDATE products
            SET quantity_in_stock = GREATEST(quantity_in_stock - $1, 0)
            WHERE product_id = $2
        `, it.Quantity, it.ProductID); err != nil {
			http.Error(w, "DB error updating stock", http.StatusInternalServerError)
			return
		}
		// 2.2) Добавляем в order_items
		if _, err := tx.Exec(`
            INSERT INTO order_items (order_id, product_id, quantity)
            VALUES ($1, $2, $3)
        `, orderID, it.ProductID, it.Quantity); err != nil {
			http.Error(w, "DB error inserting order_items", http.StatusInternalServerError)
			return
		}
		// 2.3) Удаляем из корзины
		if _, err := tx.Exec(`
            DELETE FROM cart_items
            WHERE cart_id = (SELECT cart_id FROM carts WHERE user_id=$1)
              AND product_id = $2
        `, userID, it.ProductID); err != nil {
			http.Error(w, "DB error clearing cart", http.StatusInternalServerError)
			return
		}
	}

	// 3) Коммитим
	if err := tx.Commit(); err != nil {
		http.Error(w, "DB error commit", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func listOrdersHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	query := `
        WITH order_totals AS (
            SELECT 
                o.order_id,
                COUNT(oi.order_item_id) as total_items,
                CAST(SUM(CAST(p.price AS DECIMAL(10,2)) * oi.quantity) AS VARCHAR) as total_amount
            FROM orders o
            LEFT JOIN order_items oi ON o.order_id = oi.order_id
            LEFT JOIN products p ON oi.product_id = p.product_id
            GROUP BY o.order_id
        )
        SELECT 
            o.order_id,
            os.status_name as status,
            o.order_ts,
            COALESCE(ot.total_items, 0) as total_items,
            COALESCE(ot.total_amount, '0') as total_amount,
            o.user_id,
            json_agg(
                json_build_object(
                    'product_id', p.product_id,
                    'quantity', oi.quantity,
                    'product_name', p.name,
                    'price', p.price
                )
            ) as items
        FROM orders o
        LEFT JOIN order_statuses os ON o.status_id = os.status_id
        LEFT JOIN order_totals ot ON o.order_id = ot.order_id
        LEFT JOIN order_items oi ON o.order_id = oi.order_id
        LEFT JOIN products p ON oi.product_id = p.product_id
        WHERE o.user_id = $1
        GROUP BY o.order_id, os.status_name, o.order_ts, ot.total_items, ot.total_amount, o.user_id
        ORDER BY o.order_ts DESC`

	// Execute query
	rows, err := db.Query(query, userID)
	if err != nil {
		log.Printf("Error querying orders: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var orders []OrderSummary
	for rows.Next() {
		var order OrderSummary
		var itemsJSON string
		err := rows.Scan(
			&order.OrderID,
			&order.Status,
			&order.OrderTS,
			&order.TotalItems,
			&order.TotalAmount,
			&order.UserID,
			&itemsJSON,
		)
		if err != nil {
			log.Printf("Error scanning order row: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Parse items JSON into struct
		if err := json.Unmarshal([]byte(itemsJSON), &order.Items); err != nil {
			log.Printf("Error parsing items JSON: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		orders = append(orders, order)
	}

	// Return orders as JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(orders); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	if err := db.Ping(); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	fmt.Fprintln(w, "pong")
}

func productsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не разрешён", http.StatusMethodNotAllowed)
		return
	}

	query := `
    SELECT
      p.product_id,
      p.name,
      p.price::text,
      p.color,
      p.width_cm,
      p.height_cm,
      p.weight_g,
      p.image_url,
      p.description,
      p.quantity_in_stock,
      COALESCE(array_agg(c.name) FILTER (WHERE c.name IS NOT NULL), '{}') AS categories
    FROM products p
    LEFT JOIN products_categories pc ON pc.product_id = p.product_id
    LEFT JOIN categories c            ON c.category_id = pc.category_id
    GROUP BY
      p.product_id, p.name, p.price, p.color, p.width_cm,
      p.height_cm, p.weight_g, p.image_url, p.description, p.quantity_in_stock
    `

	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, "Ошибка чтения из БД", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		var cats pq.StringArray
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Price,
			&p.Color, &p.WidthCm, &p.HeightCm,
			&p.WeightG, &p.ImageURL, &p.Description, &p.QuantityInStock,
			&cats,
		); err != nil {
			http.Error(w, "Ошибка обработки данных", http.StatusInternalServerError)
			return
		}
		p.Categories = []string(cats)
		products = append(products, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не разрешён", http.StatusMethodNotAllowed)
		return
	}
	// URL ожидается вида /users/123
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 2 {
		http.Error(w, "Неверный путь", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(parts[1])
	if err != nil {
		http.Error(w, "ID должен быть числом", http.StatusBadRequest)
		return
	}

	// Запросим поля профиля
	row := db.QueryRow(`
        SELECT user_id, first_name, last_name, email, phone, profile_picture_url, registration_ts
        FROM users
        WHERE user_id = $1
    `, id)

	var u User
	var ts sql.NullTime
	err = row.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.Phone, &u.ProfilePicture, &ts)
	if err == sql.ErrNoRows {
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}
	u.RegistrationTs = ts.Time.Format(time.RFC3339)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(u)
}

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	nameRegex  = regexp.MustCompile(`^\p{L}+$`)
	phoneRegex = regexp.MustCompile(`^\d{11}$`)
	cardNumRe  = regexp.MustCompile(`^\d{4} \d{4} \d{4} \d{4}$`)
)

func validateString(field, val string, minLen, maxLen int) error {
	length := utf8.RuneCountInString(val)
	if length < minLen || length > maxLen {
		return fmt.Errorf("%s must be between %d and %d characters", field, minLen, maxLen)
	}
	return nil
}

func validateEmail(email string) error {
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

func validateName(field, val string) error {
	if !nameRegex.MatchString(val) {
		return fmt.Errorf("%s must contain only letters", field)
	}
	return nil
}

func validatePhone(phone string) error {
	if !phoneRegex.MatchString(phone) {
		return fmt.Errorf("phone must be exactly 11 digits")
	}
	return nil
}

func validateCard(req *PaymentCard) error {
	if !cardNumRe.MatchString(req.CardNumber) {
		return fmt.Errorf("card_number must be '0000 0000 0000 0000'")
	}
	if utf8.RuneCountInString(req.CardholderName) < 1 {
		return fmt.Errorf("cardholder_name cannot be empty")
	}
	if req.ExpMonth < 1 || req.ExpMonth > 12 {
		return fmt.Errorf("exp_month must be between 1 and 12")
	}
	if req.ExpYear < 0 || req.ExpYear > 99 {
		return fmt.Errorf("exp_year must be two digits")
	}
	return nil
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", http.StatusMethodNotAllowed)
		return
	}
	var creds Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Неверный формат JSON", http.StatusBadRequest)
		return
	}

	// — ВАЛИДАЦИИ —

	// 1) Имя и фамилия: не пустые, только буквы, до 50 символов
	if err := validateString("first_name", creds.FirstName, 1, 50); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validateName("first_name", creds.FirstName); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validateString("last_name", creds.LastName, 1, 50); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validateName("last_name", creds.LastName); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 2) Email
	if err := validateEmail(creds.Email); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 3) Пароль (минимум 6 символов)
	if utf8.RuneCountInString(creds.Password) < 6 {
		http.Error(w, "password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	// 4) Телефон (если указан) — ровно 11 цифр
	if creds.Phone != "" {
		if err := validatePhone(creds.Phone); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}
	var userID int
	err = db.QueryRow(`
        INSERT INTO users (role_id, first_name, last_name, email, phone, password_hash)
        VALUES (2, $1, $2, $3, $4, $5) RETURNING user_id
    `, creds.FirstName, creds.LastName, creds.Email, creds.Phone, string(hash)).Scan(&userID)
	if err != nil {
		http.Error(w, "Email или телефон уже заняты", http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int{"user_id": userID})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", http.StatusMethodNotAllowed)
		return
	}
	var creds Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Неверный формат JSON", http.StatusBadRequest)
		return
	}
	// Достаём пользователя и хэш из БД
	var storedHash string
	var userID int
	err := db.QueryRow(`
    SELECT user_id, password_hash FROM users WHERE email = $1
  `, creds.Email).Scan(&userID, &storedHash)
	if err == sql.ErrNoRows {
		http.Error(w, "Неверные учётные данные", http.StatusUnauthorized)
		return
	} else if err != nil {
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}
	// Проверяем пароль
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(creds.Password)); err != nil {
		http.Error(w, "Неверные учётные данные", http.StatusUnauthorized)
		return
	}
	// Создаём JWT
	expiration := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID: userID,
		Email:  creds.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiration),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		http.Error(w, "Ошибка создания токена", http.StatusInternalServerError)
		return
	}
	// Отдаём токен
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":   tokenString,
		"user_id": userID,
	})
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		claims := token.Claims.(*Claims)
		// кладём userID в контекст запроса
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		next(w, r.WithContext(ctx))
	}
}

func getUserID(r *http.Request) int {
	return r.Context().Value("user_id").(int)
}

func cartHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := getUserID(r)
	// Убедимся, что есть корзина
	var cartID int
	err := db.QueryRow("SELECT cart_id FROM carts WHERE user_id=$1", userID).Scan(&cartID)
	if err == sql.ErrNoRows {
		// создаём пустую корзину
		err = db.QueryRow("INSERT INTO carts(user_id) VALUES($1) RETURNING cart_id", userID).Scan(&cartID)
	} else if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	// Собираем элементы
	rows, err := db.Query(`
        SELECT ci.cart_item_id, ci.quantity,
               p.product_id,p.name,p.price::text,p.color,p.width_cm,p.height_cm,p.weight_g,p.image_url,p.description,p.quantity_in_stock,
               COALESCE(array_agg(c.name) FILTER (WHERE c.name IS NOT NULL), '{}')
        FROM cart_items ci
        JOIN products p ON p.product_id=ci.product_id
        LEFT JOIN products_categories pc ON pc.product_id=p.product_id
        LEFT JOIN categories c ON c.category_id=pc.category_id
        WHERE ci.cart_id=$1
        GROUP BY ci.cart_item_id, ci.quantity,
                 p.product_id,p.name,p.price,p.color,p.width_cm,p.height_cm,p.weight_g,p.image_url,p.description,p.quantity_in_stock
    `, cartID)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []CartItem
	for rows.Next() {
		var ci CartItem
		var cats pq.StringArray
		if err := rows.Scan(
			&ci.CartItemID, &ci.Quantity,
			&ci.Product.ID, &ci.Product.Name, &ci.Product.Price,
			&ci.Product.Color, &ci.Product.WidthCm, &ci.Product.HeightCm,
			&ci.Product.WeightG, &ci.Product.ImageURL,
			&ci.Product.Description, &ci.Product.QuantityInStock,
			&cats,
		); err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		ci.Product.Categories = []string(cats)
		items = append(items, ci)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func addOrUpdateItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := getUserID(r)
	var req CartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// получаем или создаём корзину
	var cartID int
	if err := db.QueryRow("SELECT cart_id FROM carts WHERE user_id=$1", userID).Scan(&cartID); err == sql.ErrNoRows {
		db.QueryRow("INSERT INTO carts(user_id) VALUES($1) RETURNING cart_id", userID).Scan(&cartID)
	}

	// Проверка на отрицательное количество
	if req.Quantity < 0 {
		// Уменьшаем количество товара, но не допускаем, чтобы оно стало отрицательным
		_, err := db.Exec(`
            UPDATE cart_items
            SET quantity = GREATEST(quantity + $1, 0)
            WHERE cart_id = $2 AND product_id = $3
        `, req.Quantity, cartID, req.ProductID)
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
	} else {
		// upsert: если уже есть — обновим количество, иначе вставим
		_, err := db.Exec(`
            INSERT INTO cart_items(cart_id, product_id, quantity)
            VALUES($1, $2, $3)
            ON CONFLICT (cart_id, product_id) DO
            UPDATE SET quantity = cart_items.quantity + EXCLUDED.quantity
        `, cartID, req.ProductID, req.Quantity)
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func removeItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := getUserID(r)
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	pid, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		http.Error(w, "Bad product id", http.StatusBadRequest)
		return
	}
	// найдём cart_id
	var cartID int
	if err := db.QueryRow("SELECT cart_id FROM carts WHERE user_id=$1", userID).Scan(&cartID); err != nil {
		http.Error(w, "Cart not found", http.StatusNotFound)
		return
	}
	// удаляем
	_, err = db.Exec("DELETE FROM cart_items WHERE cart_id=$1 AND product_id=$2", cartID, pid)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET  /cards        — список карт пользователя
func listCards(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	rows, err := db.Query(`
		SELECT card_id, user_id, cardholder_name, card_number, exp_month, exp_year
		FROM payment_cards
		WHERE user_id=$1
	`, userID)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var cards []PaymentCard
	for rows.Next() {
		var c PaymentCard
		if err := rows.Scan(&c.CardID, &c.UserID, &c.CardholderName, &c.CardNumber, &c.ExpMonth, &c.ExpYear); err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		cards = append(cards, c)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cards)
}

// POST /cards        — добавить новую карту
func addCard(w http.ResponseWriter, r *http.Request) {
	var req PaymentCard
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad JSON", http.StatusBadRequest)
		return
	}
	// валидация полей
	if err := validateCard(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.UserID = getUserID(r)

	// вставка
	err := db.QueryRow(`
		INSERT INTO payment_cards (user_id, cardholder_name, card_number, exp_month, exp_year)
		VALUES ($1,$2,$3,$4,$5) RETURNING card_id
	`, req.UserID, req.CardholderName, req.CardNumber, req.ExpMonth, req.ExpYear).
		Scan(&req.CardID)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(req)
}

// DELETE /cards/{card_id} — удалить карту
func deleteCard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	id, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		http.Error(w, "Bad card_id", http.StatusBadRequest)
		return
	}
	// проверяем, что карта принадлежит пользователю
	userID := getUserID(r)
	res, err := db.Exec(`
		DELETE FROM payment_cards
		WHERE card_id=$1 AND user_id=$2
	`, id, userID)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	if cnt, _ := res.RowsAffected(); cnt == 0 {
		http.Error(w, "Not found or forbidden", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func cardsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listCards(w, r)
	case http.MethodPost:
		addCard(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func changePasswordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := getUserID(r)

	var req PasswordChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Получаем старый хэш из БД
	var storedHash string
	if err := db.QueryRow(
		"SELECT password_hash FROM users WHERE user_id=$1", userID,
	).Scan(&storedHash); err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Сравниваем старый пароль
	if bcrypt.CompareHashAndPassword(
		[]byte(storedHash), []byte(req.OldPassword),
	) != nil {
		http.Error(w, "Old password is incorrect", http.StatusUnauthorized)
		return
	}

	// Хэшируем новый пароль
	newHash, err := bcrypt.GenerateFromPassword(
		[]byte(req.NewPassword), bcrypt.DefaultCost,
	)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Обновляем БД
	if _, err := db.Exec(
		"UPDATE users SET password_hash=$1 WHERE user_id=$2",
		string(newHash), userID,
	); err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
