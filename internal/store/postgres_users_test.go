//go:build integration

package store

import (
	"context"
	"errors"
	"testing"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/config"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/db"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/testutil"
)

func TestPostgresUserStore(t *testing.T) {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	pool, err := db.NewPool(ctx, cfg.Postgres)
	if err != nil {
		testutil.SkipOnServiceError(t, err, "connect postgres")
	}
	defer db.Close(pool)

	store := NewPostgresUserStore(pool)

	user := domain.User{
		ID:           "00000000-0000-0000-0000-000000000101",
		Email:        "store-user@example.com",
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=1$c2FsdHNhbHRzYWx0c2FsdA$aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}

	_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = $1 OR email = $2`, user.ID, user.Email)

	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	got, err := store.GetUserByEmail(ctx, user.Email)
	if err != nil {
		t.Fatalf("GetUserByEmail failed: %v", err)
	}

	if got.ID != user.ID {
		t.Fatalf("expected user id %q, got %q", user.ID, got.ID)
	}

	if err := store.CreateUser(ctx, user); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("expected already exists, got %v", err)
	}
}
