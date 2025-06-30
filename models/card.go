package models

type PaymentCard struct {
	CardID         int    `json:"card_id"`
	UserID         int    `json:"user_id"`
	CardholderName string `json:"cardholder_name"`
	CardNumber     string `json:"card_number"`
	ExpMonth       int    `json:"exp_month"`
	ExpYear        int    `json:"exp_year"`
}
