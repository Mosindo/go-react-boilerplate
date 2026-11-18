package users

type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required,max=128"`
}

// DeletedAccount lists what must be cleaned up outside the database.
type DeletedAccount struct {
	PhotoStorageKeys []string
	FormerMatchIDs   []string
}
