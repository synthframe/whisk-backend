package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(context.Background()); err != nil {
		return nil, err
	}
	return pool, nil
}

func Migrate(pool *pgxpool.Pool) error {
	_, err := pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			name VARCHAR(255) NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
		CREATE TABLE IF NOT EXISTS generated_images (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			storage_key VARCHAR(500) NOT NULL,
			subject_prompt TEXT DEFAULT '',
			scene_prompt TEXT DEFAULT '',
			style_prompt TEXT DEFAULT '',
			style_preset VARCHAR(50) DEFAULT '',
			width INT DEFAULT 1024,
			height INT DEFAULT 1024,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
	`)
	if err != nil {
		return err
	}
	// Add columns to existing tables if not present (idempotent migration)
	_, _ = pool.Exec(context.Background(), `
		ALTER TABLE generated_images
			ADD COLUMN IF NOT EXISTS subject_prompt TEXT DEFAULT '',
			ADD COLUMN IF NOT EXISTS scene_prompt TEXT DEFAULT '',
			ADD COLUMN IF NOT EXISTS style_prompt TEXT DEFAULT '',
			ADD COLUMN IF NOT EXISTS style_preset VARCHAR(50) DEFAULT '',
			ADD COLUMN IF NOT EXISTS width INT DEFAULT 1024,
			ADD COLUMN IF NOT EXISTS height INT DEFAULT 1024;
	`)
	return nil
}
