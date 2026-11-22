package users

import (
	"errors"
	"time"
)

var ErrUserNotFound = errors.New("user not found")

// User is the minimal public identity of another member. E-mail addresses
// are private and are only ever returned to their owner (GET /me).
type User struct {
	ID        string    `json:"id"`
	FirstName string    `json:"firstName"`
	CreatedAt time.Time `json:"createdAt"`
}

type UsersResponse struct {
	Users []User `json:"users"`
}
