package models

import (
	"github.com/google/uuid"
)

type User struct {
	ID              uuid.UUID `bson:"_id"`
	Email           string    `bson:"email"`
	Username        string    `bson:"username"`
	Password        string    `bson:"password"`
	Description     string    `bson:"description,omitempy"`
	ProfileImage    string    `bson:"profile_image,omitempty"`
	IsEmailVerified bool      `bson:"is_email_verified"`
	CreatedAt       int64     `bson:"created_at"`
	UpdatedAt       int64     `bson:"updated_at"`
}

type Image struct {
	ID   uuid.UUID
	Data []byte
}
