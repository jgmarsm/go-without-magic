package memory

import (
	"context"
	"sync"

	"github.com/JoX23/go-without-magic/internal/domain"
)

// UserRepository es una implementación en memoria del dominio.
//
// Usos:
//   - Tests unitarios y de integración (sin base de datos real)
//   - Desarrollo local sin infraestructura
//
// Es seguro para uso concurrente (sync.RWMutex).
type UserRepository struct {
	mu      sync.RWMutex
	byID    map[string]*domain.User
	byEmail map[string]*domain.User
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		byID:    make(map[string]*domain.User),
		byEmail: make(map[string]*domain.User),
	}
}

// CreateIfNotExists verifica que el email no exista y crea de forma ATÓMICA.
// - Si el email ya existe: retorna ErrUserDuplicated (SIN crear)
// - Si no existe: crea el usuario en AMBOS índices (atómico)
//
// GARANTÍA DE CONCURRENCIA: Esta operación es thread-safe.
// No hay ventana entre check y write.
func (r *UserRepository) CreateIfNotExists(_ context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Verificar duplicado - MIENTRAS TENEMOS EL LOCK
	if _, exists := r.byEmail[user.Email]; exists {
		return domain.ErrUserDuplicated
	}

	// Crear - MIENTRAS TENEMOS EL LOCK (operación atómica)
	r.byID[user.ID.String()] = user
	r.byEmail[user.Email] = user

	return nil
}

func (r *UserRepository) Save(_ context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Guardar en ambos índices para búsquedas O(1)
	r.byID[user.ID.String()] = user
	r.byEmail[user.Email] = user

	return nil
}

func (r *UserRepository) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.byEmail[email]
	if !ok {
		return nil, domain.ErrUserNotFound
	}

	return user, nil
}

func (r *UserRepository) FindByID(_ context.Context, id string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.byID[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}

	return user, nil
}

func (r *UserRepository) List(_ context.Context) ([]*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	users := make([]*domain.User, 0, len(r.byID))
	for _, u := range r.byID {
		users = append(users, u)
	}

	return users, nil
}
