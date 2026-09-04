package dto

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string       `json:"token"`
	TokenType string       `json:"token_type"`
	ExpiresIn int64        `json:"expires_in"`
	User      UserResponse `json:"user"`
}
