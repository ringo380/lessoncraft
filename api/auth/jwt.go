package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	// ErrInvalidToken is returned when the token is invalid
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken is returned when the token has expired
	ErrExpiredToken = errors.New("token has expired")
	// ErrInvalidClaims is returned when the token claims are invalid
	ErrInvalidClaims = errors.New("invalid token claims")
)

// JWTService handles JWT token generation and validation
type JWTService struct {
	secretKey     []byte
	issuer        string
	tokenDuration time.Duration
}

// NewJWTService creates a new JWT service
func NewJWTService(secretKey string, issuer string, tokenDuration time.Duration) *JWTService {
	return &JWTService{
		secretKey:     []byte(secretKey),
		issuer:        issuer,
		tokenDuration: tokenDuration,
	}
}

// GenerateToken generates a new JWT token for a user
func (s *JWTService) GenerateToken(userID, email string, roles []Role) (string, time.Time, error) {
	expirationTime := time.Now().Add(s.tokenDuration)

	claims := TokenClaims{
		UserID:    userID,
		Email:     email,
		Roles:     roles,
		ExpiresAt: expirationTime.Unix(),
		IssuedAt:  time.Now().Unix(),
		Issuer:    s.issuer,
		Subject:   userID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": claims.UserID,
		"email":   claims.Email,
		"roles":   claims.Roles,
		"exp":     claims.ExpiresAt,
		"iat":     claims.IssuedAt,
		"iss":     claims.Issuer,
		"sub":     claims.Subject,
	})

	tokenString, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expirationTime, nil
}

// GenerateRefreshToken generates a refresh token
func (s *JWTService) GenerateRefreshToken() (string, error) {
	// Generate a random UUID for the refresh token
	refreshToken := uuid.New().String()
	return refreshToken, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *JWTService) ValidateToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})

	if err != nil {
		// Check if the error is due to an expired token
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidClaims
	}

	// Extract and validate claims
	userID, ok := claims["user_id"].(string)
	if !ok {
		return nil, ErrInvalidClaims
	}

	email, ok := claims["email"].(string)
	if !ok {
		return nil, ErrInvalidClaims
	}

	// Extract roles
	rolesInterface, ok := claims["roles"].([]interface{})
	if !ok {
		return nil, ErrInvalidClaims
	}

	roles := make([]Role, len(rolesInterface))
	for i, r := range rolesInterface {
		roleStr, ok := r.(string)
		if !ok {
			return nil, ErrInvalidClaims
		}
		roles[i] = Role(roleStr)
	}

	// Extract standard claims
	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, ErrInvalidClaims
	}

	iat, ok := claims["iat"].(float64)
	if !ok {
		return nil, ErrInvalidClaims
	}

	iss, ok := claims["iss"].(string)
	if !ok {
		return nil, ErrInvalidClaims
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return nil, ErrInvalidClaims
	}

	return &TokenClaims{
		UserID:    userID,
		Email:     email,
		Roles:     roles,
		ExpiresAt: int64(exp),
		IssuedAt:  int64(iat),
		Issuer:    iss,
		Subject:   sub,
	}, nil
}
