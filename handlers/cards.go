package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"server/models"
	"server/validators"
	"strconv"
	"strings"
)

func ListCards(db *sql.DB, getUserID func(*http.Request) int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		var cards []models.PaymentCard
		for rows.Next() {
			var c models.PaymentCard
			if err := rows.Scan(&c.CardID, &c.UserID, &c.CardholderName, &c.CardNumber, &c.ExpMonth, &c.ExpYear); err != nil {
				http.Error(w, "DB error", http.StatusInternalServerError)
				return
			}
			cards = append(cards, c)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cards)
	}
}

func AddCard(db *sql.DB, getUserID func(*http.Request) int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.PaymentCard
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad JSON", http.StatusBadRequest)
			return
		}
		if err := validators.ValidateCard(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		req.UserID = getUserID(r)
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
}

func DeleteCard(db *sql.DB, getUserID func(*http.Request) int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
}

func CardsHandler(db *sql.DB, getUserID func(*http.Request) int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			ListCards(db, getUserID)(w, r)
		case http.MethodPost:
			AddCard(db, getUserID)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}
