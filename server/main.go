package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

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
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/login", loginHandler)

	// ===== 3) Запуск HTTP-сервера =====
	log.Println("Сервер запущен на порту 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
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
      p.description,
      p.quantity_in_stock,
      COALESCE(array_agg(c.name) FILTER (WHERE c.name IS NOT NULL), '{}') AS categories
    FROM products p
    LEFT JOIN products_categories pc ON pc.product_id = p.product_id
    LEFT JOIN categories c            ON c.category_id = pc.category_id
    GROUP BY
      p.product_id, p.name, p.price, p.color, p.width_cm,
      p.height_cm, p.weight_g, p.description, p.quantity_in_stock
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
			&p.WeightG, &p.Description, &p.QuantityInStock,
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
	// Хэшируем пароль
	hash, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}
	// Вставляем в БД. Пользователь по умолчанию получает роль user (id = 2)
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
	json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
}
