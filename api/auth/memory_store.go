package auth

import (
	"errors"
	"sync"
)

var (
	// ErrUserNotFound is returned when a user is not found
	ErrUserNotFound = errors.New("user not found")
	// ErrUserAlreadyExists is returned when a user with the same email already exists
	ErrUserAlreadyExists = errors.New("user with this email already exists")
)

// MemoryUserStore is an in-memory implementation of the UserStore interface
// It is primarily used for testing purposes
type MemoryUserStore struct {
	users  map[string]*UserWithAuth // Map of user ID to user
	emails map[string]string        // Map of email to user ID
	mu     sync.RWMutex             // Mutex to protect concurrent access
}

// NewMemoryUserStore creates a new in-memory user store
func NewMemoryUserStore() *MemoryUserStore {
	return &MemoryUserStore{
		users:  make(map[string]*UserWithAuth),
		emails: make(map[string]string),
	}
}

// GetUserByEmail retrieves a user by email
func (s *MemoryUserStore) GetUserByEmail(email string) (*UserWithAuth, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userID, ok := s.emails[email]
	if !ok {
		return nil, ErrUserNotFound
	}

	user, ok := s.users[userID]
	if !ok {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (s *MemoryUserStore) GetUserByID(id string) (*UserWithAuth, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// CreateUser creates a new user
func (s *MemoryUserStore) CreateUser(user *UserWithAuth) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if user with this email already exists
	if _, ok := s.emails[user.Email]; ok {
		return ErrUserAlreadyExists
	}

	// Store user
	s.users[user.Id] = user
	s.emails[user.Email] = user.Id

	return nil
}

// UpdateUser updates an existing user
func (s *MemoryUserStore) UpdateUser(id string, user *UserWithAuth) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if user exists
	existingUser, ok := s.users[id]
	if !ok {
		return ErrUserNotFound
	}

	// Check if email is being changed
	if existingUser.Email != user.Email {
		// Check if new email is already in use
		if _, ok := s.emails[user.Email]; ok {
			return ErrUserAlreadyExists
		}

		// Update email mapping
		delete(s.emails, existingUser.Email)
		s.emails[user.Email] = id
	}

	// Update user
	s.users[id] = user

	return nil
}

// DeleteUser deletes a user
func (s *MemoryUserStore) DeleteUser(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if user exists
	user, ok := s.users[id]
	if !ok {
		return ErrUserNotFound
	}

	// Delete user
	delete(s.users, id)
	delete(s.emails, user.Email)

	return nil
}
