package auth

// UserResponse is a user shaped for the client.
type UserResponse struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	IsAdmin   bool   `json:"is_admin"`
	AvatarURL string `json:"avatar_url"`
	Locale    string `json:"locale"`
	CreatedAt string `json:"created_at"`
}

// MeResponse is the signed-in user for /auth/me.
type MeResponse struct {
	User UserResponse `json:"user"`
}
