package domain

import "context"

// UserRepository define el contrato del puerto de salida.
//
// El dominio define la interfaz; la implementación vive en
// internal/repository/. Esto es el patrón Port & Adapter.
//
// REGLA: esta interfaz NO importa nada fuera del paquete domain.
type UserRepository interface {
	// CreateIfNotExists verifica que el email no exista y crea de forma ATÓMICA.
	// Retorna ErrUserDuplicated si el email ya existe.
	// RECOMENDADO para operaciones que necesitan protección contra duplicados.
	//
	// GARANTÍA: Thread-safe. Dos goroutines no pueden crear el mismo email simultáneamente.
	CreateIfNotExists(ctx context.Context, user *User) error

	// Save crea o actualiza un usuario (incondicionalmente).
	// Usa CreateIfNotExists si necesitas chequeo de duplicados.
	// NOTA: En memoria ignora duplicados. En postgres falla si hay UNIQUE constraint.
	Save(ctx context.Context, user *User) error

	// FindByEmail busca un usuario por su email.
	// Retorna ErrUserNotFound si no existe.
	FindByEmail(ctx context.Context, email string) (*User, error)

	// FindByID busca un usuario por su ID.
	// Retorna ErrUserNotFound si no existe.
	FindByID(ctx context.Context, id string) (*User, error)

	// List retorna todos los usuarios.
	List(ctx context.Context) ([]*User, error)
}
