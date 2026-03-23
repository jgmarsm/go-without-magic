package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/JoX23/go-without-magic/internal/domain"
	"github.com/JoX23/go-without-magic/internal/repository/memory"
	"github.com/JoX23/go-without-magic/internal/service"
)

// newTestService crea un servicio con repositorio en memoria.
// No necesita mocks — el repo en memoria ES la implementación de test.
func newTestService(t *testing.T) *service.UserService {
	t.Helper()
	repo := memory.NewUserRepository()
	logger := zap.NewNop() // logger silencioso en tests
	return service.NewUserService(repo, logger)
}

func TestCreateUser_Success(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	user, err := svc.CreateUser(ctx, "alice@example.com", "Alice")

	require.NoError(t, err)
	assert.Equal(t, "alice@example.com", user.Email)
	assert.Equal(t, "Alice", user.Name)
	assert.NotEmpty(t, user.ID)
	assert.False(t, user.CreatedAt.IsZero())
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// Crear primero
	_, err := svc.CreateUser(ctx, "bob@example.com", "Bob")
	require.NoError(t, err)

	// Intentar crear con el mismo email
	_, err = svc.CreateUser(ctx, "bob@example.com", "Bob Clone")

	assert.ErrorIs(t, err, domain.ErrUserDuplicated)
}

func TestCreateUser_InvalidEmail(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.CreateUser(ctx, "", "Alice")

	assert.ErrorIs(t, err, domain.ErrInvalidEmail)
}

func TestCreateUser_InvalidName(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.CreateUser(ctx, "alice@example.com", "")

	assert.ErrorIs(t, err, domain.ErrInvalidName)
}

func TestGetByID_NotFound(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, "non-existent-id")

	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestListUsers_Empty(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	users, err := svc.ListUsers(ctx)

	require.NoError(t, err)
	assert.Empty(t, users)
}

func TestListUsers_AfterCreate(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.CreateUser(ctx, "a@example.com", "A")
	require.NoError(t, err)

	_, err = svc.CreateUser(ctx, "b@example.com", "B")
	require.NoError(t, err)

	users, err := svc.ListUsers(ctx)
	require.NoError(t, err)
	assert.Len(t, users, 2)
}
