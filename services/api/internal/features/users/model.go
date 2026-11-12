package users

import (
	"errors"
	"time"
)

var ErrUserNotFound = errors.New("user not found")

// User is the minimal public directory entry. Emails are private and are
// never exposed to other members.
type User struct {
	ID        string    `json:"id"`
	FirstName string    `json:"firstName"`
	CreatedAt time.Time `json:"createdAt"`
}

type UsersResponse struct {
	Users []User `json:"users"`
}
