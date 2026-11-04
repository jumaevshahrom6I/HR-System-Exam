package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"hr-system/internal/config"
	"hr-system/pkg/postgres"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		os.Exit(1)
	}
}

func run() error {
	email := flag.String("email", "hr@company.local", "email of the first HR")
	flag.Parse()
	login := strings.ToLower(strings.TrimSpace(*email))
	if login == "" {
		return fmt.Errorf("email is empty")
	}

	fmt.Print("Password for the first HR (8-72 bytes): ")
	password, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return fmt.Errorf("read password: %w", err)
	}
	password = strings.TrimRight(password, "\r\n")
	if l := len(password); l < 8 || l > 72 {
		return fmt.Errorf("password must be 8-72 bytes")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, postgres.Config{
		Host:     cfg.Postgres.Host,
		Port:     cfg.Postgres.Port,
		User:     cfg.Postgres.User,
		Password: cfg.Postgres.Password,
		Database: cfg.Postgres.Database,
		SSLMode:  cfg.Postgres.SSLMode,
	})
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer pool.Close()

	tag, err := pool.Exec(ctx,
		`INSERT INTO users (role_id, email, password_hash)
		 SELECT id, $1, $2 FROM roles WHERE code = 'hr'
		 ON CONFLICT (email) DO NOTHING`,
		login, string(hash))
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		fmt.Println("User with this email already exists (or role hr is missing). Nothing changed.")
		return nil
	}
	fmt.Printf("First HR created: %s\n", login)
	return nil
}
