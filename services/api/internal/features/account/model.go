package account

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required,max=512"`
	NewPassword     string `json:"newPassword" binding:"required,max=512"`
}

type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required,max=512"`
}
