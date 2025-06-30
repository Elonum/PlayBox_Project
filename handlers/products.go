package handlers

import (
	"encoding/json"
	"net/http"

	"database/sql"
	"server/models"

	"github.com/lib/pq"
)

func ProductsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		var products []models.Product
		for rows.Next() {
			var p models.Product
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
}
