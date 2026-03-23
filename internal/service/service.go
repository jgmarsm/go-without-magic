package service

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/JoX23/go-without-magic/internal/domain"
)

// UserService contiene SOLO lógica de negocio.
// No sabe nada de HTTP, gRPC, bases de datos ni frameworks.
type UserService struct {
	repo   domain.UserRepository
	logger *zap.Logger
}

func NewUserService(repo domain.UserRepository, logger *zap.Logger) *UserService {
	return &UserService{
		repo:   repo,
		logger: logger,
	}
}

// CreateUser orquesta la creación de un usuario.
//
// Flujo:
//  1. Construir entidad (valida invariantes del dominio)
//  2. Persistir de forma atómica (check-then-act en repo)
//
// SEGURIDAD DE CONCURRENCIA:
// Esta función es segura para ser llamada simultáneamente por múltiples goroutines.
// La protección ocurre en repo.CreateIfNotExists() que es operación atómica.
func (s *UserService) CreateUser(ctx context.Context, email, name string) (*domain.User, error) {
	// Paso 1: el constructor valida las invariantes del dominio
	user, err := domain.NewUser(email, name)
	if err != nil {
		return nil, err // ErrInvalidEmail o ErrInvalidName
	}

	// Paso 2: persistir DE FORMA ATÓMICA (check + create en una sola operación)
	// CreateIfNotExists maneja la verificación de duplicados SIN ventanas de race condition
	if err := s.repo.CreateIfNotExists(ctx, user); err != nil {
		if errors.Is(err, domain.ErrUserDuplicated) {
			// Email ya existe - error de negocio esperado
			return nil, err
		}
		// Error técnico
		s.logger.Error("failed to create user",
			zap.String("email", email),
			zap.Error(err),
		)
		return nil, fmt.Errorf("creating user: %w", err)
	}

	s.logger.Info("user created",
		zap.String("id", user.ID.String()),
		zap.String("email", email),
	)

	return user, nil
}

// GetByID busca un usuario por su ID.
func (s *UserService) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("finding user: %w", err)
	}

	return user, nil
}

// ListUsers retorna todos los usuarios.
func (s *UserService) ListUsers(ctx context.Context) ([]*domain.User, error) {
	users, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	return users, nil
}
