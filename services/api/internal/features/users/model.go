package users

import "time"

// User is the public directory entry of a member. Emails are private and
// are never exposed to other members.
type User struct {
	ID             string
	DisplayName    *string
	OrganizationID string
	CreatedAt      time.Time
}

type UserResponse struct {
	ID             string    `json:"id"`
	DisplayName    *string   `json:"displayName"`
	OrganizationID string    `json:"organizationId"`
	CreatedAt      time.Time `json:"createdAt"`
}

type UsersResponse struct {
	Users []UserResponse `json:"users"`
}
