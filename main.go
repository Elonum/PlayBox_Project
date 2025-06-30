package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"

	"server/config"
	"server/handlers"
	"server/models"
)

func main() {
	cfg := config.LoadConfig()
	if len(cfg.JWTSecret) == 0 {
		log.Fatal("JWT_SECRET is not set in environment")
	}

	db, err := sql.Open("postgres", cfg.DBConnStr)
	if err != nil {
		log.Fatalf("Не удалось открыть подключение к БД: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Не удалось пинговать БД: %v", err)
	}
	log.Println("Успешно подключились к БД")

	auth := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			token, err := jwt.ParseWithClaims(tokenStr, &models.Claims{}, func(t *jwt.Token) (interface{}, error) {
				return cfg.JWTSecret, nil
			})
			if err != nil || !token.Valid {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			claims := token.Claims.(*models.Claims)
			ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
			next(w, r.WithContext(ctx))
		}
	}

	getUserID := func(r *http.Request) int {
		return r.Context().Value("user_id").(int)
	}

	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		w.Write([]byte("pong"))
	})

	http.HandleFunc("/products", handlers.ProductsHandler(db))
	http.HandleFunc("/users/", handlers.UserHandler(db))
	http.HandleFunc("/register", handlers.RegisterHandler(db))
	http.HandleFunc("/login", handlers.LoginHandler(db, cfg.JWTSecret))

	http.HandleFunc("/cart", auth(handlers.CartHandler(db, getUserID)))
	http.HandleFunc("/cart/items", auth(handlers.AddOrUpdateItem(db, getUserID)))
	http.HandleFunc("/cart/items/", auth(handlers.RemoveItem(db, getUserID)))

	http.HandleFunc("/cards", auth(handlers.CardsHandler(db, getUserID)))
	http.HandleFunc("/cards/", auth(handlers.DeleteCard(db, getUserID)))

	http.HandleFunc("/checkout", auth(handlers.CheckoutHandler(db, getUserID)))
	http.HandleFunc("/orders", auth(handlers.ListOrdersHandler(db, getUserID)))
	http.HandleFunc("/users/password", auth(handlers.ChangePasswordHandler(db, getUserID)))

	log.Printf("Сервер запущен на порту %s", cfg.ServerPort)
	log.Fatal(http.ListenAndServe(":"+cfg.ServerPort, nil))
}
