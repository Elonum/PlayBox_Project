package models

type CartItem struct {
	CartItemID int     `json:"cart_item_id"`
	Product    Product `json:"product"`
	Quantity   int     `json:"quantity"`
}

type CartRequest struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}
