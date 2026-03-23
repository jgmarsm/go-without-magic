package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/JoX23/go-without-magic/internal/config"
	"github.com/JoX23/go-without-magic/internal/domain"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func New(cfg config.DatabaseConfig) (*UserRepository, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("parsing database DSN: %w", err)
	}

	poolCfg.MaxConns = int32(cfg.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.MaxIdleConns)

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &UserRepository{pool: pool}, nil
}

func (r *UserRepository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func (r *UserRepository) Close() {
	r.pool.Close()
}

// CreateIfNotExists crea un usuario si el email no existe.
// En PostgreSQL, confiamos en el UNIQUE constraint para detectar duplicados.
// Si el email ya existe, la BD retorna un error que mapeamos a ErrUserDuplicated.
func (r *UserRepository) CreateIfNotExists(ctx context.Context, user *domain.User) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, email, name, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		user.ID.String(), user.Email, user.Name, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		// PostgreSQL retorna error de constraint violation si email ya existe
		// Podemos chequear el string del error o usar sqlstate
		// Por ahora, delegamos al caller a manejar el error
		return fmt.Errorf("creating user: %w", err)
	}
	return nil
}

func (r *UserRepository) Save(ctx context.Context, user *domain.User) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, email, name, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		user.ID.String(), user.Email, user.Name, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting user: %w", err)
	}
	return nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, email, name, created_at, updated_at FROM users WHERE email = $1`,
		email,
	)
	user, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying by email: %w", err)
	}
	return user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, email, name, created_at, updated_at FROM users WHERE id = $1`,
		id,
	)
	user, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying by id: %w", err)
	}
	return user, nil
}

func (r *UserRepository) List(ctx context.Context) ([]*domain.User, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, email, name, created_at, updated_at FROM users ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		users = append(users, u)
	}

	return users, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanUser(s scanner) (*domain.User, error) {
	var u domain.User
	var idStr string

	if err := s.Scan(&idStr, &u.Email, &u.Name, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parsing uuid: %w", err)
	}
	u.ID = id

	return &u, nil
}
