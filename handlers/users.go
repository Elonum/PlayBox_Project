package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"server/models"
	"strconv"
	"strings"
	"time"
)

func UserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Метод не разрешён", http.StatusMethodNotAllowed)
			return
		}
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
		row := db.QueryRow(`
			SELECT user_id, first_name, last_name, email, phone, profile_picture_url, registration_ts
			FROM users
			WHERE user_id = $1
		`, id)
		var u models.User
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
}
