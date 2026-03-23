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
//  1. Validar y construir entidad (dominio)
//  2. Verificar que el email no exista (regla de negocio)
//  3. Persistir
func (s *UserService) CreateUser(ctx context.Context, email, name string) (*domain.User, error) {
	// Paso 1: el constructor valida las invariantes del dominio
	user, err := domain.NewUser(email, name)
	if err != nil {
		return nil, err // ErrInvalidEmail o ErrInvalidName
	}

	// Paso 2: verificar duplicado
	existing, err := s.repo.FindByEmail(ctx, email)
	switch {
	case err == nil && existing != nil:
		// Email ya existe → error de negocio
		return nil, domain.ErrUserDuplicated
	case errors.Is(err, domain.ErrUserNotFound):
		// No existe → podemos continuar
	case err != nil:
		// Error técnico del repositorio
		return nil, fmt.Errorf("checking existing user: %w", err)
	}

	// Paso 3: persistir
	if err := s.repo.Save(ctx, user); err != nil {
		s.logger.Error("failed to save user",
			zap.String("email", email),
			zap.Error(err),
		)
		return nil, fmt.Errorf("saving user: %w", err)
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
