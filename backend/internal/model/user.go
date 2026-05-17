package model

type User struct {
	Email    string `json:"email"`
	Password string `json:"-"` // bcrypt hash, never serialized to JSON
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=4"`
}

type LoginResponse struct {
	Token string `json:"token"`
	Email string `json:"email"`
}

type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=4"`
}
