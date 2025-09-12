package models

type User struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Password string `json:"-"` // Don't include password in JSON responses
}

type App struct {
	ID          int    `json:"id"`
	UserID      int    `json:"user_id"`
	Name        string `json:"name"`
	WingetID    string `json:"winget_id,omitempty"`
	DownloadURL string `json:"download_url,omitempty"`
	Args        string `json:"args,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type CreateAppRequest struct {
	Name        string `json:"name"`
	WingetID    string `json:"winget_id,omitempty"`
	DownloadURL string `json:"download_url,omitempty"`
	Args        string `json:"args,omitempty"`
}

type UpdateAppRequest struct {
	Name        string `json:"name"`
	WingetID    string `json:"winget_id,omitempty"`
	DownloadURL string `json:"download_url,omitempty"`
	Args        string `json:"args,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
