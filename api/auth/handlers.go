package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/ringo380/lessoncraft/api/middleware"
	"github.com/ringo380/lessoncraft/pwd/types"
	"golang.org/x/crypto/bcrypt"
)

// UserStore defines the interface for user storage operations
type UserStore interface {
	// GetUserByEmail retrieves a user by email
	GetUserByEmail(email string) (*UserWithAuth, error)
	// GetUserByID retrieves a user by ID
	GetUserByID(id string) (*UserWithAuth, error)
	// CreateUser creates a new user
	CreateUser(user *UserWithAuth) error
	// UpdateUser updates an existing user
	UpdateUser(id string, user *UserWithAuth) error
	// DeleteUser deletes a user
	DeleteUser(id string) error
}

// AuthHandler handles HTTP requests related to authentication
type AuthHandler struct {
	userStore  UserStore
	jwtService *JWTService
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(userStore UserStore, jwtService *JWTService) *AuthHandler {
	return &AuthHandler{
		userStore:  userStore,
		jwtService: jwtService,
	}
}

// RegisterRoutes registers the authentication routes with the provided router
func (h *AuthHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/auth/register", h.Register).Methods("POST")
	r.HandleFunc("/api/auth/login", h.Login).Methods("POST")
	r.HandleFunc("/api/auth/refresh", h.RefreshToken).Methods("POST")

	// Protected routes that require authentication
	authMiddleware := AuthMiddleware(h.jwtService)

	r.Handle("/api/auth/me", authMiddleware(http.HandlerFunc(h.GetCurrentUser))).Methods("GET")
	r.Handle("/api/auth/logout", authMiddleware(http.HandlerFunc(h.Logout))).Methods("POST")
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(middleware.ErrorResponse{
			Error:     "InvalidRequest",
			Code:      http.StatusBadRequest,
			Message:   "Invalid request format",
			Details:   err.Error(),
			TimeStamp: time.Now(),
		})
		return
	}

	// Validate request
	if req.Email == "" || req.Password == "" || req.Name == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(middleware.ErrorResponse{
			Error:     "ValidationError",
			Code:      http.StatusBadRequest,
			Message:   "Email, password, and name are required",
			TimeStamp: time.Now(),
		})
		return
	}

	// Check if user already exists
	existingUser, err := h.userStore.GetUserByEmail(req.Email)
	if err == nil && existingUser != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(middleware.ErrorResponse{
			Error:     "UserAlreadyExists",
			Code:      http.StatusConflict,
			Message:   "A user with this email already exists",
			TimeStamp: time.Now(),
		})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(middleware.ErrorResponse{
			Error:     "InternalServerError",
			Code:      http.StatusInternalServerError,
			Message:   "Error hashing password",
			Details:   err.Error(),
			TimeStamp: time.Now(),
		})
		return
	}

	// Create user
	now := time.Now()
	user := &UserWithAuth{
		User: types.User{
			Id:       uuid.New().String(),
			Name:     req.Name,
			Email:    req.Email,
			Provider: "local",
		},
		PasswordHash:  string(hashedPassword),
		Roles:         []Role{RoleLearner}, // Default role is learner
		AccountStatus: "active",
		EmailVerified: false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := h.userStore.CreateUser(user); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(middleware.ErrorResponse{
			Error:     "DatabaseError",
			Code:      http.StatusInternalServerError,
			Message:   "Failed to create user",
			Details:   err.Error(),
			TimeStamp: time.Now(),
		})
		return
	}

	// Generate tokens
	token, expiresAt, err := h.jwtService.GenerateToken(user.Id, user.Email, user.Roles)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(middleware.ErrorResponse{
			Error:     "TokenGenerationError",
			Code:      http.StatusInternalServerError,
			Message:   "Failed to generate token",
			Details:   err.Error(),
			TimeStamp: time.Now(),
		})
		return
	}

	refreshToken, err := h.jwtService.GenerateRefreshToken()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(middleware.ErrorResponse{
			Error:     "TokenGenerationError",
			Code:      http.StatusInternalServerError,
			Message:   "Failed to generate refresh token",
			Details:   err.Error(),
			TimeStamp: time.Now(),
		})
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		User:         user.User,
	})
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(middleware.ErrorResponse{
			Error:     "InvalidRequest",
			Code:      http.StatusBadRequest,
			Message:   "Invalid request format",
			Details:   err.Error(),
			TimeStamp: time.Now(),
		})
		return
	}

	// Validate request
	if req.Email == "" || req.Password == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(middleware.ErrorResponse{
			Error:     "ValidationError",
			Code:      http.StatusBadRequest,
			Message:   "Email and password are required",
			TimeStamp: time.Now(),
		})
		return
	}

	// Get user by email
	user, err := h.userStore.GetUserByEmail(req.Email)
	if err != nil || user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(middleware.ErrorResponse{
			Error:     "InvalidCredentials",
			Code:      http.StatusUnauthorized,
			Message:   "Invalid email or password",
			TimeStamp: time.Now(),
		})
		return
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(middleware.ErrorResponse{
			Error:     "InvalidCredentials",
			Code:      http.StatusUnauthorized,
			Message:   "Invalid email or password",
			TimeStamp: time.Now(),
		})
		return
	}

	// Update last login time
	user.LastLogin = time.Now()
	if err := h.userStore.UpdateUser(user.Id, user); err != nil {
		// Log error but continue
		// TODO: Add proper logging
	}

	// Generate tokens
	token, expiresAt, err := h.jwtService.GenerateToken(user.Id, user.Email, user.Roles)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(middleware.ErrorResponse{
			Error:     "TokenGenerationError",
			Code:      http.StatusInternalServerError,
			Message:   "Failed to generate token",
			Details:   err.Error(),
			TimeStamp: time.Now(),
		})
		return
	}

	refreshToken, err := h.jwtService.GenerateRefreshToken()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(middleware.ErrorResponse{
			Error:     "TokenGenerationError",
			Code:      http.StatusInternalServerError,
			Message:   "Failed to generate refresh token",
			Details:   err.Error(),
			TimeStamp: time.Now(),
		})
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		User:         user.User,
	})
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement refresh token functionality
	// This would typically involve:
	// 1. Validating the refresh token
	// 2. Looking up the associated user
	// 3. Generating a new access token
	// 4. Optionally generating a new refresh token

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(middleware.ErrorResponse{
		Error:     "NotImplemented",
		Code:      http.StatusNotImplemented,
		Message:   "Refresh token functionality not implemented yet",
		TimeStamp: time.Now(),
	})
}

// GetCurrentUser returns the current authenticated user
func (h *AuthHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by AuthMiddleware)
	userID, ok := GetUserID(r)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(middleware.ErrorResponse{
			Error:     "Unauthorized",
			Code:      http.StatusUnauthorized,
			Message:   "User not authenticated",
			TimeStamp: time.Now(),
		})
		return
	}

	// Get user from store
	user, err := h.userStore.GetUserByID(userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(middleware.ErrorResponse{
			Error:     "DatabaseError",
			Code:      http.StatusInternalServerError,
			Message:   "Failed to retrieve user",
			Details:   err.Error(),
			TimeStamp: time.Now(),
		})
		return
	}

	// Return user information
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user.User)
}

// Logout handles user logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// In a stateless JWT authentication system, logout is typically handled client-side
	// by removing the token from storage. However, for security, we could implement
	// a token blacklist or revocation mechanism.

	// For now, just return a success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Logout successful",
	})
}
