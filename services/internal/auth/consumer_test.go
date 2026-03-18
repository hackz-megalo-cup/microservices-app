package auth

import (
	"context"
	"errors"
	"testing"
)

type stubPokemonRegistrar struct {
	userID    string
	pokemonID string
	err       error
	called    bool
}

func (s *stubPokemonRegistrar) RegisterPokemon(_ context.Context, userID, pokemonID string) error {
	s.called = true
	s.userID = userID
	s.pokemonID = pokemonID
	return s.err
}

func TestHandleCaughtPokemon_SuccessResultRegistersPokemon(t *testing.T) {
	repo := &stubPokemonRegistrar{}
	data := []byte(`{"data":{"user_id":"u1","pokemon_id":"p25","result":"success"}}`)

	err := handleCaughtPokemon(context.Background(), repo, data)
	if err != nil {
		t.Fatalf("handleCaughtPokemon returned error: %v", err)
	}
	if !repo.called {
		t.Fatal("expected RegisterPokemon to be called")
	}
	if repo.userID != "u1" || repo.pokemonID != "p25" {
		t.Fatalf("unexpected register args: user_id=%q pokemon_id=%q", repo.userID, repo.pokemonID)
	}
}

func TestHandleCaughtPokemon_FailedResultSkipsRegistration(t *testing.T) {
	repo := &stubPokemonRegistrar{}
	data := []byte(`{"data":{"user_id":"u1","pokemon_id":"p25","result":"failed"}}`)

	err := handleCaughtPokemon(context.Background(), repo, data)
	if err != nil {
		t.Fatalf("handleCaughtPokemon returned error: %v", err)
	}
	if repo.called {
		t.Fatal("expected RegisterPokemon not to be called for non-success result")
	}
}

func TestHandleCaughtPokemon_EmptyResultBackCompatRegisters(t *testing.T) {
	repo := &stubPokemonRegistrar{}
	data := []byte(`{"data":{"user_id":"u1","pokemon_id":"p25"}}`)

	err := handleCaughtPokemon(context.Background(), repo, data)
	if err != nil {
		t.Fatalf("handleCaughtPokemon returned error: %v", err)
	}
	if !repo.called {
		t.Fatal("expected RegisterPokemon to be called when result is omitted")
	}
}

func TestHandleCaughtPokemon_RepoErrorReturnsError(t *testing.T) {
	repo := &stubPokemonRegistrar{err: errors.New("boom")}
	data := []byte(`{"data":{"user_id":"u1","pokemon_id":"p25","result":"success"}}`)

	err := handleCaughtPokemon(context.Background(), repo, data)
	if err == nil {
		t.Fatal("expected error")
	}
}
