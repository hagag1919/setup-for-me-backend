package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"setupforme/models"
	"setupforme/utils"

	"github.com/lib/pq"
)

type AuthHandler struct {
	db *sql.DB
}

func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{db: db}
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req models.SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Normalize email
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	// Validate email format
	if !isValidEmail(req.Email) {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid email format")
		return
	}

	// Validate password strength
	if len(req.Password) < 8 {
		writeErrorResponse(w, http.StatusBadRequest, "Password must be at least 8 characters long")
		return
	}

	// Check if user already exists
	var existingID int
	err := h.db.QueryRow("SELECT id FROM users WHERE email = $1", req.Email).Scan(&existingID)
	if err == nil {
		writeErrorResponse(w, http.StatusConflict, "User already exists")
		return
	} else if err != sql.ErrNoRows {
		writeErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	// Insert user and get ID (PostgreSQL: use RETURNING id)
	var userID int64
	err = h.db.QueryRow("INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id", req.Email, hashedPassword).Scan(&userID)
	if err != nil {
		// Handle unique violation just in case of race condition
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			writeErrorResponse(w, http.StatusConflict, "User already exists")
			return
		}
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Generate JWT token
	token, err := utils.GenerateJWT(int(userID), req.Email)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	user := models.User{
		ID:    int(userID),
		Email: req.Email,
	}

	response := models.AuthResponse{
		Token: token,
		User:  user,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Normalize email
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	// Get user from database
	var user models.User
	var hashedPassword string
	err := h.db.QueryRow("SELECT id, email, password FROM users WHERE email = $1", req.Email).
		Scan(&user.ID, &user.Email, &hashedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			writeErrorResponse(w, http.StatusUnauthorized, "Invalid credentials")
		} else {
			writeErrorResponse(w, http.StatusInternalServerError, "Database error")
		}
		return
	}

	// Check password
	if !utils.CheckPasswordHash(req.Password, hashedPassword) {
		writeErrorResponse(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Generate JWT token
	token, err := utils.GenerateJWT(user.ID, user.Email)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	response := models.AuthResponse{
		Token: token,
		User:  user,
	}

	json.NewEncoder(w).Encode(response)
}

func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(strings.ToLower(email))
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(models.ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	})
}
