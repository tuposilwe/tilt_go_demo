package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func initializeDatabase(ctx context.Context, db *sql.DB) error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS restaurants (
	id BIGSERIAL PRIMARY KEY,
	name TEXT NOT NULL,
	cuisine TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	address TEXT NOT NULL,
	city TEXT NOT NULL,
	state TEXT NOT NULL DEFAULT '',
	postal_code TEXT NOT NULL DEFAULT '',
	country TEXT NOT NULL,
	phone TEXT NOT NULL DEFAULT '',
	email TEXT NOT NULL DEFAULT '',
	website TEXT NOT NULL DEFAULT '',
	price_level SMALLINT NOT NULL CHECK (price_level BETWEEN 1 AND 4),
	rating NUMERIC(3, 2) NOT NULL CHECK (rating BETWEEN 0 AND 5),
	seed_key TEXT UNIQUE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);`,
		`CREATE INDEX IF NOT EXISTS restaurants_city_idx ON restaurants (city)`,
		`CREATE INDEX IF NOT EXISTS restaurants_cuisine_idx ON restaurants (cuisine)`,
		`ALTER TABLE restaurants ALTER COLUMN rating TYPE NUMERIC(3, 2)`,
		`CREATE TABLE IF NOT EXISTS restaurant_seed_state (
			seed_name TEXT PRIMARY KEY,
			seeded_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
	}
	for _, statement := range schema {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("create restaurant schema: %w", err)
		}
	}
	if err := seedRestaurants(ctx, db); err != nil {
		return fmt.Errorf("seed restaurants: %w", err)
	}
	return nil
}

func openDatabase(ctx context.Context) (*sql.DB, error) {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	name := os.Getenv("DB_NAME")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")

	for _, field := range []struct {
		name  string
		value string
	}{
		{"DB_HOST", host},
		{"DB_PORT", port},
		{"DB_NAME", name},
		{"DB_USER", user},
		{"DB_PASSWORD", password},
	} {
		if field.value == "" {
			return nil, fmt.Errorf("%s must be set", field.name)
		}
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil {
		return nil, fmt.Errorf("DB_PORT must be a number: %w", err)
	}
	if portNumber < 1 || portNumber > 65535 {
		return nil, fmt.Errorf("DB_PORT must be between 1 and 65535")
	}

	dsn := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, port),
		Path:   name,
	}
	query := dsn.Query()
	query.Set("sslmode", "disable")
	dsn.RawQuery = query.Encode()

	db, err := sql.Open("pgx", dsn.String())
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}

func seedRestaurants(ctx context.Context, db *sql.DB) error {
	type citySeed struct {
		city, state, postalCode, country, cuisine string
	}
	names := []string{
		"Juniper Table", "Copper Spoon", "Olive & Oak", "Bluebird Kitchen", "Ember House",
		"Golden Fig", "Harbor & Hearth", "Maple Room", "Saffron Garden", "The Daily Platter",
	}
	cities := []citySeed{
		{"Portland", "Oregon", "97201", "United States", "Pacific Northwest"},
		{"Austin", "Texas", "78701", "United States", "Tex-Mex"},
		{"Chicago", "Illinois", "60601", "United States", "Italian"},
		{"Seattle", "Washington", "98101", "United States", "Japanese"},
		{"Miami", "Florida", "33101", "United States", "Caribbean"},
		{"Denver", "Colorado", "80202", "United States", "American"},
		{"Boston", "Massachusetts", "02108", "United States", "Seafood"},
		{"Nashville", "Tennessee", "37201", "United States", "Southern"},
		{"San Diego", "California", "92101", "United States", "Mexican"},
		{"New York", "New York", "10001", "United States", "Mediterranean"},
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var seeded bool
	if err := tx.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT 1 FROM restaurant_seed_state WHERE seed_name = 'demo-restaurants-v1')",
	).Scan(&seeded); err != nil {
		return err
	}
	if seeded {
		return tx.Commit()
	}

	const insert = `
INSERT INTO restaurants
	(name, cuisine, description, address, city, state, postal_code, country, phone, email, website, price_level, rating, seed_key)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
ON CONFLICT (seed_key) DO NOTHING`
	for i, name := range names {
		for j, city := range cities {
			seedKey := fmt.Sprintf("demo-%02d-%02d", i+1, j+1)
			restaurantName := fmt.Sprintf("%s %s", name, city.city)
			rating := 3.8 + float64((i*3+j*2)%13)/10
			priceLevel := (i+j)%4 + 1
			phone := fmt.Sprintf("+1-555-%03d-%04d", 200+i, 1000+j)
			slug := strings.ToLower(strings.NewReplacer("&", "and", " ", "-").Replace(name))
			email := fmt.Sprintf("hello@%s.example", slug)
			website := fmt.Sprintf("https://%s.example", slug)
			_, err := tx.ExecContext(ctx, insert,
				restaurantName, city.cuisine,
				fmt.Sprintf("A neighborhood favorite serving %s-inspired dishes in %s.", strings.ToLower(city.cuisine), city.city),
				fmt.Sprintf("%d %s Avenue", 100+i*10+j, []string{"Market", "Oak", "River", "Park"}[(i+j)%4]),
				city.city, city.state, city.postalCode, city.country,
				phone, email, website, priceLevel, rating, seedKey)
			if err != nil {
				return fmt.Errorf("insert seed restaurant %q: %w", restaurantName, err)
			}
		}
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO restaurant_seed_state (seed_name) VALUES ('demo-restaurants-v1') ON CONFLICT DO NOTHING",
	); err != nil {
		return err
	}
	return tx.Commit()
}
