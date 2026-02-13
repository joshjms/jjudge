package types

import "time"

// User represents an account in the system.
// It contains identity, role, and audit metadata.
type User struct {
	// ID is the unique identifier of the user.
	ID int `json:"id" db:"id"`

	// Username is the unique login name chosen by the user.
	Username string `json:"username" db:"username"`

	// Email is the user's email address.
	Email string `json:"email" db:"email"`

	// Name is the user's display or full name.
	Name string `json:"name" db:"name"`

	// Role indicates the user's authorization level or role
	// within the system (e.g., "admin", "user").
	Role string `json:"role" db:"role"`

	// PasswordHash stores the hashed representation of the user's password.
	// This field is never exposed in API responses.
	PasswordHash string `json:"-" db:"password_hash"`

	// CreatedAt is the timestamp when the user account was created.
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// UpdatedAt is the timestamp of the most recent update to the user account.
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
