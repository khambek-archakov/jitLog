package configure

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func InitDB(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	DB, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("init postgres pool: %w", err)
	}

	if err := DB.Ping(ctx); err != nil {
		DB.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return DB, nil
}
