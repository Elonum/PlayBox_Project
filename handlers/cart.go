package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"server/models"
	"strconv"
	"strings"

	"github.com/lib/pq"
)

func CartHandler(db *sql.DB, getUserID func(*http.Request) int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		userID := getUserID(r)
		var cartID int
		err := db.QueryRow("SELECT cart_id FROM carts WHERE user_id=$1", userID).Scan(&cartID)
		if err == sql.ErrNoRows {
			err = db.QueryRow("INSERT INTO carts(user_id) VALUES($1) RETURNING cart_id", userID).Scan(&cartID)
		} else if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
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
		var items []models.CartItem
		for rows.Next() {
			var ci models.CartItem
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
}

func AddOrUpdateItem(db *sql.DB, getUserID func(*http.Request) int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		userID := getUserID(r)
		var req models.CartRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		var cartID int
		if err := db.QueryRow("SELECT cart_id FROM carts WHERE user_id=$1", userID).Scan(&cartID); err == sql.ErrNoRows {
			db.QueryRow("INSERT INTO carts(user_id) VALUES($1) RETURNING cart_id", userID).Scan(&cartID)
		}
		if req.Quantity < 0 {
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
}

func RemoveItem(db *sql.DB, getUserID func(*http.Request) int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		var cartID int
		if err := db.QueryRow("SELECT cart_id FROM carts WHERE user_id=$1", userID).Scan(&cartID); err != nil {
			http.Error(w, "Cart not found", http.StatusNotFound)
			return
		}
		_, err = db.Exec("DELETE FROM cart_items WHERE cart_id=$1 AND product_id=$2", cartID, pid)
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
