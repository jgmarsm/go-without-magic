package domain

import "errors"

// Errores de dominio — tipados para que el handler los mapee
// correctamente a códigos HTTP o gRPC.
//
// REGLA: estos errores representan casos de negocio, NO errores técnicos.
var (
	ErrUserNotFound   = errors.New("user not found")
	ErrUserDuplicated = errors.New("user already exists")
	ErrInvalidEmail   = errors.New("email cannot be empty")
	ErrInvalidName    = errors.New("name cannot be empty")
)
