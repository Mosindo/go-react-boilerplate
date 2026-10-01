package users

// DeleteAccountRequest requires the password again: a stolen access token
// alone must not be enough to erase an account.
type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required,max=72"`
}
