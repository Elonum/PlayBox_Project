package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"server/models"

	"golang.org/x/crypto/bcrypt"
)

func ChangePasswordHandler(db *sql.DB, getUserID func(*http.Request) int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		userID := getUserID(r)
		var req models.PasswordChangeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		var storedHash string
		if err := db.QueryRow(
			"SELECT password_hash FROM users WHERE user_id=$1", userID,
		).Scan(&storedHash); err != nil {
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
		if bcrypt.CompareHashAndPassword(
			[]byte(storedHash), []byte(req.OldPassword),
		) != nil {
			http.Error(w, "Old password is incorrect", http.StatusUnauthorized)
			return
		}
		newHash, err := bcrypt.GenerateFromPassword(
			[]byte(req.NewPassword), bcrypt.DefaultCost,
		)
		if err != nil {
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
		if _, err := db.Exec(
			"UPDATE users SET password_hash=$1 WHERE user_id=$2",
			string(newHash), userID,
		); err != nil {
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
