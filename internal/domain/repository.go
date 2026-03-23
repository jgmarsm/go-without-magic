package domain

import "context"

// UserRepository define el contrato del puerto de salida.
//
// El dominio define la interfaz; la implementación vive en
// internal/repository/. Esto es el patrón Port & Adapter.
//
// REGLA: esta interfaz NO importa nada fuera del paquete domain.
type UserRepository interface {
	Save(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	List(ctx context.Context) ([]*User, error)
}
