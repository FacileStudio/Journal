package auth

// RegisterRequest is the body of a password registration.
type RegisterRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

// LoginRequest is the body of a password login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UserResponse is a user shaped for the client.
type UserResponse struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	IsAdmin   bool   `json:"is_admin"`
	AvatarURL string `json:"avatar_url"`
	CreatedAt string `json:"created_at"`
}

// AuthResponse carries the session token issued to a successful login.
type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

// MeResponse is the signed-in user for /auth/me.
type MeResponse struct {
	User UserResponse `json:"user"`
}
