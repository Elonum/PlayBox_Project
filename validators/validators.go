package validators

import (
	"fmt"
	"regexp"
	"server/models"
	"unicode/utf8"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	nameRegex  = regexp.MustCompile(`^\p{L}+$`)
	phoneRegex = regexp.MustCompile(`^\d{11}$`)
	cardNumRe  = regexp.MustCompile(`^\d{4} \d{4} \d{4} \d{4}$`)
)

func ValidateString(field, val string, minLen, maxLen int) error {
	length := utf8.RuneCountInString(val)
	if length < minLen || length > maxLen {
		return fmt.Errorf("%s must be between %d and %d characters", field, minLen, maxLen)
	}
	return nil
}

func ValidateEmail(email string) error {
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

func ValidateName(field, val string) error {
	if !nameRegex.MatchString(val) {
		return fmt.Errorf("%s must contain only letters", field)
	}
	return nil
}

func ValidatePhone(phone string) error {
	if !phoneRegex.MatchString(phone) {
		return fmt.Errorf("phone must be exactly 11 digits")
	}
	return nil
}

func ValidateCard(req *models.PaymentCard) error {
	if !cardNumRe.MatchString(req.CardNumber) {
		return fmt.Errorf("card_number must be '0000 0000 0000 0000'")
	}
	if utf8.RuneCountInString(req.CardholderName) < 1 {
		return fmt.Errorf("cardholder_name cannot be empty")
	}
	if req.ExpMonth < 1 || req.ExpMonth > 12 {
		return fmt.Errorf("exp_month must be between 1 and 12")
	}
	if req.ExpYear < 0 || req.ExpYear > 99 {
		return fmt.Errorf("exp_year must be two digits")
	}
	return nil
}
