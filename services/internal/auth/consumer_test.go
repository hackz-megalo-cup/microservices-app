package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"
)

func TestHandleCaughtPokemon(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &mockUserRepository{}
		event := PokemonCaught{
			UserID:    "user-123",
			PokemonID: "pikachu",
			CaughtAt:  time.Now(),
		}
		data, _ := json.Marshal(event)

		err := handleCaughtPokemon(context.Background(), repo, data)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		repo := &mockUserRepository{}
		data := []byte(`{invalid json}`)

		err := handleCaughtPokemon(context.Background(), repo, data)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("missing user_id", func(t *testing.T) {
		repo := &mockUserRepository{}
		event := PokemonCaught{
			UserID:    "",
			PokemonID: "pikachu",
			CaughtAt:  time.Now(),
		}
		data, _ := json.Marshal(event)

		err := handleCaughtPokemon(context.Background(), repo, data)

		// Should return nil (warn and skip)
		if err != nil {
			t.Fatalf("expected nil, got error: %v", err)
		}
	})

	t.Run("missing pokemon_id", func(t *testing.T) {
		repo := &mockUserRepository{}
		event := PokemonCaught{
			UserID:    "user-123",
			PokemonID: "",
			CaughtAt:  time.Now(),
		}
		data, _ := json.Marshal(event)

		err := handleCaughtPokemon(context.Background(), repo, data)

		// Should return nil (warn and skip)
		if err != nil {
			t.Fatalf("expected nil, got error: %v", err)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repo := &mockUserRepository{
			registerPokemonErr: sql.ErrConnDone,
		}
		event := PokemonCaught{
			UserID:    "user-123",
			PokemonID: "pikachu",
			CaughtAt:  time.Now(),
		}
		data, _ := json.Marshal(event)

		err := handleCaughtPokemon(context.Background(), repo, data)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
