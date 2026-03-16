package cert_test

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"testing"
	"time"

	"github.com/hackz-megalo-cup/microservices-app/game-server/internal/cert"
)

func TestGenerateEphemeralCert(t *testing.T) {
	tlsCert, hashHex, err := cert.GenerateEphemeral()
	if err != nil {
		t.Fatalf("GenerateEphemeral() error: %v", err)
	}

	// Can parse certificate
	leaf, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		t.Fatalf("ParseCertificate error: %v", err)
	}

	// Is ECDSA P-256
	if _, ok := leaf.PublicKey.(*ecdsa.PublicKey); !ok {
		t.Fatal("expected ECDSA public key")
	}

	// Validity <= 14 days
	duration := leaf.NotAfter.Sub(leaf.NotBefore)
	if duration > 14*24*time.Hour {
		t.Fatalf("cert validity %v exceeds 14 days", duration)
	}

	// SHA-256 hash is correct
	h := sha256.Sum256(tlsCert.Certificate[0])
	expectedHex := fmt.Sprintf("%x", h[:])
	if hashHex != expectedHex {
		t.Fatalf("hash mismatch: got %s, want %s", hashHex, expectedHex)
	}

	// Hash is 64 chars hex
	if len(hashHex) != 64 {
		t.Fatalf("expected 64 char hex hash, got %d", len(hashHex))
	}
}
