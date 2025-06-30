package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"server/models"
	"server/validators"
	"time"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func RegisterHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Метод не разрешён", http.StatusMethodNotAllowed)
			return
		}
		var creds models.Credentials
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			http.Error(w, "Неверный формат JSON", http.StatusBadRequest)
			return
		}
		if err := validators.ValidateString("first_name", creds.FirstName, 1, 50); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := validators.ValidateName("first_name", creds.FirstName); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := validators.ValidateString("last_name", creds.LastName, 1, 50); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := validators.ValidateName("last_name", creds.LastName); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := validators.ValidateEmail(creds.Email); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if utf8.RuneCountInString(creds.Password) < 6 {
			http.Error(w, "password must be at least 6 characters", http.StatusBadRequest)
			return
		}
		if creds.Phone != "" {
			if err := validators.ValidatePhone(creds.Phone); err != nil {
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
}

func LoginHandler(db *sql.DB, jwtKey []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Метод не разрешён", http.StatusMethodNotAllowed)
			return
		}
		var creds models.Credentials
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			http.Error(w, "Неверный формат JSON", http.StatusBadRequest)
			return
		}
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
		if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(creds.Password)); err != nil {
			http.Error(w, "Неверные учётные данные", http.StatusUnauthorized)
			return
		}
		expiration := time.Now().Add(24 * time.Hour)
		claims := &models.Claims{
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
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token":   tokenString,
			"user_id": userID,
		})
	}
}
