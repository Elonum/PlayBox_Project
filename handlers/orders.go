package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"server/models"
)

func CheckoutHandler(db *sql.DB, getUserID func(*http.Request) int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		userID := getUserID(r)
		var req models.CheckoutRequest
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
		var orderID int
		err = tx.QueryRow(`
			INSERT INTO orders (user_id, status_id)
			VALUES ($1, (SELECT status_id FROM order_statuses WHERE status_name='Новый'))
			RETURNING order_id
		`, userID).Scan(&orderID)
		if err != nil {
			http.Error(w, "DB error creating order", http.StatusInternalServerError)
			return
		}
		for _, it := range req.Items {
			if _, err := tx.Exec(`
				UPDATE products
				SET quantity_in_stock = GREATEST(quantity_in_stock - $1, 0)
				WHERE product_id = $2
			`, it.Quantity, it.ProductID); err != nil {
				http.Error(w, "DB error updating stock", http.StatusInternalServerError)
				return
			}
			if _, err := tx.Exec(`
				INSERT INTO order_items (order_id, product_id, quantity)
				VALUES ($1, $2, $3)
			`, orderID, it.ProductID, it.Quantity); err != nil {
				http.Error(w, "DB error inserting order_items", http.StatusInternalServerError)
				return
			}
			if _, err := tx.Exec(`
				DELETE FROM cart_items
				WHERE cart_id = (SELECT cart_id FROM carts WHERE user_id=$1)
				  AND product_id = $2
			`, userID, it.ProductID); err != nil {
				http.Error(w, "DB error clearing cart", http.StatusInternalServerError)
				return
			}
		}
		if err := tx.Commit(); err != nil {
			http.Error(w, "DB error commit", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func ListOrdersHandler(db *sql.DB, getUserID func(*http.Request) int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
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
		rows, err := db.Query(query, userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var orders []models.OrderSummary
		for rows.Next() {
			var order models.OrderSummary
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
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if err := json.Unmarshal([]byte(itemsJSON), &order.Items); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			orders = append(orders, order)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(orders); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
