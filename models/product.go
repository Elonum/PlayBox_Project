package models

type Product struct {
	ID              int      `json:"id"`
	Name            string   `json:"name"`
	Price           string   `json:"price"`
	Color           string   `json:"color"`
	WidthCm         int      `json:"width_cm"`
	HeightCm        int      `json:"height_cm"`
	WeightG         int      `json:"weight_g"`
	ImageURL        *string  `json:"image_url,omitempty"`
	Description     *string  `json:"description,omitempty"`
	QuantityInStock int      `json:"quantity_in_stock"`
	Categories      []string `json:"categories"`
}
