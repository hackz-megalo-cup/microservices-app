package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"

	"connectrpc.com/connect"

	authv1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/auth/v1"
)

func TestRegisterUser_Validation(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey
	svc := NewService(nil, nil, nil, privateKey, publicKey, "test-kid")

	t.Run("empty email", func(t *testing.T) {
		req := connect.NewRequest(&authv1.RegisterUserRequest{
			Email:    "",
			Password: "password123",
		})
		_, err := svc.RegisterUser(context.Background(), req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var connectErr *connect.Error
		if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeInvalidArgument {
			t.Errorf("expected InvalidArgument error, got %v", err)
		}
	})

	t.Run("empty password", func(t *testing.T) {
		req := connect.NewRequest(&authv1.RegisterUserRequest{
			Email:    "test@example.com",
			Password: "",
		})
		_, err := svc.RegisterUser(context.Background(), req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var connectErr *connect.Error
		if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeInvalidArgument {
			t.Errorf("expected InvalidArgument error, got %v", err)
		}
	})
}

func TestLoginUser_Validation(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey
	svc := NewService(nil, nil, nil, privateKey, publicKey, "test-kid")

	t.Run("empty email", func(t *testing.T) {
		req := connect.NewRequest(&authv1.LoginUserRequest{
			Email:    "",
			Password: "password123",
		})
		_, err := svc.LoginUser(context.Background(), req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var connectErr *connect.Error
		if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeInvalidArgument {
			t.Errorf("expected InvalidArgument error, got %v", err)
		}
	})

	t.Run("empty password", func(t *testing.T) {
		req := connect.NewRequest(&authv1.LoginUserRequest{
			Email:    "test@example.com",
			Password: "",
		})
		_, err := svc.LoginUser(context.Background(), req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var connectErr *connect.Error
		if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeInvalidArgument {
			t.Errorf("expected InvalidArgument error, got %v", err)
		}
	})
}

func TestGetUserProfile_Validation(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey
	svc := NewService(nil, nil, nil, privateKey, publicKey, "test-kid")

	t.Run("empty user_id", func(t *testing.T) {
		req := connect.NewRequest(&authv1.GetUserProfileRequest{
			UserId: "",
		})
		_, err := svc.GetUserProfile(context.Background(), req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var connectErr *connect.Error
		if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeInvalidArgument {
			t.Errorf("expected InvalidArgument error, got %v", err)
		}
	})
}

func TestIssueJWT(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey
	svc := NewService(nil, nil, nil, privateKey, publicKey, "test-kid")

	token, err := svc.issueJWT("user-123", "test@example.com", "user", 3600)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}

	// Verify token can be parsed (basic validation)
	if len(token) < 50 {
		t.Errorf("token seems too short: %d chars", len(token))
	}
}

func TestRegisterPokemon(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey
	svc := NewService(nil, nil, nil, privateKey, publicKey, "test-kid")

	t.Run("empty user_id", func(t *testing.T) {
		err := svc.RegisterPokemon(context.Background(), "", "pikachu")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("empty pokemon_id", func(t *testing.T) {
		err := svc.RegisterPokemon(context.Background(), "user-123", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
