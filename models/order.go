package models

import "time"

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
