package models

type User struct {
	ID             int     `json:"id"`
	FirstName      string  `json:"first_name"`
	LastName       string  `json:"last_name"`
	Email          string  `json:"email"`
	Phone          string  `json:"phone"`
	ProfilePicture *string `json:"profile_picture_url,omitempty"`
	RegistrationTs string  `json:"registration_ts"`
}
