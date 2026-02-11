package auth

import (
	"strconv"
	"time"
)

type User struct {
	ID           UserID    `db:"id" json:"id"`
	Name         string    `db:"name" json:"name"`
	Email        string    `db:"email" json:"email"`
	PasswordHash []byte    `db:"password_hash" json:"-"`
	Role         Role      `db:"role" json:"role"`
	CreatedBy    *UserID   `db:"created_by" json:"created_by,omitempty"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type UserID int64

func (id UserID) Int64() int64 {
	return int64(id)
}

func (id UserID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

type Role string

func (r Role) String() string {
	return string(r)
}

const AdminRole Role = "admin"
const UserRole Role = "user"
