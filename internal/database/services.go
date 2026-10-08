package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Azula19-b/incidenthub/internal/models"
)

func (s *Store) ListServices(ctx context.Context, status string) ([]models.Service, error) {
	query := `SELECT id, name, description, owner_team, status, created_at, updated_at FROM services`
	args := []any{}
	if status != "" {
		query += ` WHERE status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY name`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}
	defer rows.Close()

	var services []models.Service
	for rows.Next() {
		service, scanErr := scanService(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		services = append(services, service)
	}
	return services, rows.Err()
}

func (s *Store) GetService(ctx context.Context, id int64) (models.Service, error) {
	row := s.db.QueryRowContext(ctx, `
        SELECT id, name, description, owner_team, status, created_at, updated_at
        FROM services WHERE id = ?`, id)
	service, err := scanService(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Service{}, ErrNotFound
	}
	return service, err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanService(row rowScanner) (models.Service, error) {
	var service models.Service
	var created, updated string
	if err := row.Scan(&service.ID, &service.Name, &service.Description, &service.OwnerTeam,
		&service.Status, &created, &updated); err != nil {
		return models.Service{}, err
	}
	service.CreatedAt, _ = time.Parse(time.RFC3339, created)
	service.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return service, nil
}
