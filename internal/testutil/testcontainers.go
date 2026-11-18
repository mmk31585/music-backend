//go:build integration

package testutil

import (
	"context"
)

// TestContainer represents a test container environment with Postgres and Redis.
// Real implementation requires testcontainers-go (not yet a dependency).
// Add with: go get github.com/testcontainers/testcontainers-go
type TestContainer struct {
	PostgresDSN string
	RedisAddr   string
}

// NewTestContainer creates a new test container environment.
// Requires testcontainers-go dependency.
func NewTestContainer(ctx context.Context) (*TestContainer, error) {
	// TODO: implement with testcontainers-go when dependency is added
	return &TestContainer{
		PostgresDSN: "postgres://postgres:postgres@localhost:5432/musicapp_test?sslmode=disable",
		RedisAddr:   "localhost:6379",
	}, nil
}

// Start starts the test containers and ensures they're ready
func (tc *TestContainer) Start(ctx context.Context) error {
	// TODO: implement with testcontainers-go
	return nil
}

// Stop stops the test containers
func (tc *TestContainer) Stop() error {
	// TODO: implement with testcontainers-go
	return nil
}