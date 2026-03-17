package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/protobuf/types/known/timestamppb"

	authv1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/auth/v1"
	"github.com/hackz-megalo-cup/microservices-app/services/gen/go/auth/v1/authv1connect"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/auth"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

const (
	serviceName    = "auth-service"
	serviceVersion = "0.1.0"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := platform.NewLogger()
	shutdownOTel, err := platform.SetupOTelSDK(ctx, serviceName, serviceVersion)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, shutdownOTel(context.Background()))
	}()

	migrationsFS, _ := fs.Sub(auth.MigrationsFS, "migrations")
	dbPool, sqlDB := initDatabases(ctx, migrationsFS)
	if dbPool != nil {
		defer dbPool.Close()
	}
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	brokers := platform.ParseKafkaBrokers(os.Getenv("KAFKA_BROKERS"))
	platform.TryEnsureTopics(ctx, brokers)

	publisher, _ := platform.NewEventPublisher(brokers)
	defer publisher.Close()

	outbox := platform.NewOutboxStore(dbPool, publisher)
	outbox.StartPoller(ctx, 500*time.Millisecond)

	eventStore := platform.NewEventStore(dbPool)

	// Load RSA keys for JWT signing
	privateKey, publicKey, kid, err := loadRSAKeys()
	if err != nil {
		return err
	}

	repo := auth.NewUserRepository(sqlDB)
	authSvc := auth.NewService(repo, eventStore, outbox, privateKey, publicKey, kid)

	otelInterceptor, err := otelconnect.NewInterceptor(otelconnect.WithTrustRemote())
	if err != nil {
		return err
	}

	verifier := platform.NewJWTVerifier(os.Getenv("JWKS_URL"))
	idempotencyStore := platform.NewIdempotencyStore(dbPool)
	platform.StartIdempotencyCleanup(ctx, idempotencyStore)

	connectOpts := connect.WithInterceptors(
		otelInterceptor,
		platform.NewAuthInterceptor(verifier),
		platform.NewIdempotencyInterceptor(idempotencyStore),
		platform.NewLoggingInterceptor(logger),
	)

	path, handler := authv1connect.NewAuthServiceHandler(
		&authHandler{svc: authSvc},
		connectOpts,
	)

	mux := newServerMux(path, handler, dbPool, verifier, publicKey, kid)
	startCaptureConsumer(ctx, brokers, repo)

	port := resolvePort()
	srv := newHTTPServer(ctx, mux, port)

	return runHTTPServer(ctx, logger, srv, port)
}

func initDatabases(ctx context.Context, migrationsFS fs.FS) (*pgxpool.Pool, *sql.DB) {
	dbPool := platform.InitDB(ctx, os.Getenv("DATABASE_URL"), migrationsFS, serviceName)
	if dbPool == nil {
		return nil, nil
	}
	return dbPool, stdlib.OpenDBFromPool(dbPool)
}

func newServerMux(path string, handler http.Handler, dbPool *pgxpool.Pool, verifier *platform.JWTVerifier, publicKey *rsa.PublicKey, kid string) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	registerHealthzHandler(mux, dbPool)
	registerVerifyHandler(mux, verifier)
	registerJWKSHandler(mux, publicKey, kid)
	return mux
}

func registerHealthzHandler(mux *http.ServeMux, dbPool *pgxpool.Pool) {
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if dbPool != nil {
			if err := dbPool.Ping(r.Context()); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte("db unhealthy\n"))
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
}

func registerVerifyHandler(mux *http.ServeMux, verifier *platform.JWTVerifier) {
	mux.HandleFunc("/verify", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		token := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}

		if verifier == nil {
			w.WriteHeader(http.StatusOK)
			return
		}

		claims, err := verifier.Verify(token)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if claims.Subject != "" {
			w.Header().Set("X-User-Id", claims.Subject)
		}
		if claims.Role != "" {
			w.Header().Set("X-User-Role", claims.Role)
		}

		w.WriteHeader(http.StatusOK)
	})
}

func registerJWKSHandler(mux *http.ServeMux, publicKey *rsa.PublicKey, kid string) {
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		jwkSet := map[string]interface{}{
			"keys": []map[string]interface{}{
				{
					"kty": "RSA",
					"use": "sig",
					"kid": kid,
					"n":   extractRSAModulus(publicKey),
					"e":   extractRSAExponent(publicKey),
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwkSet)
	})
}

func startCaptureConsumer(ctx context.Context, brokers []string, repo *auth.UserRepository) {
	kafkaConsumer, _ := platform.NewKafkaConsumer(
		ctx,
		brokers,
		"auth-service-consumer",
		[]string{platform.TopicCaptureCaught},
	)
	if kafkaConsumer == nil {
		return
	}

	go func() {
		if err := auth.RunConsumer(ctx, auth.ConsumerConfig{
			Client: kafkaConsumer,
			Repo:   repo,
		}); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("kafka consumer error", "error", err)
		}
	}()
}

func resolvePort() string {
	port := os.Getenv("PORT")
	if port == "" {
		return "8090"
	}
	return port
}

func newHTTPServer(ctx context.Context, mux *http.ServeMux, port string) *http.Server {
	return &http.Server{
		Addr:         ":" + port,
		BaseContext:  func(net.Listener) context.Context { return ctx },
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 30 * time.Second,
		Handler:      h2c.NewHandler(mux, &http2.Server{}),
	}
}

func runHTTPServer(ctx context.Context, logger *slog.Logger, srv *http.Server, port string) error {
	srvErr := make(chan error, 1)
	go func() {
		logger.InfoContext(ctx, "starting auth service", "port", port)
		srvErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-srvErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// authHandler implements authv1connect.AuthServiceHandler
type authHandler struct {
	svc *auth.Service
}

func (h *authHandler) RegisterUser(ctx context.Context, req *connect.Request[authv1.RegisterUserRequest]) (*connect.Response[authv1.RegisterUserResponse], error) {
	resp, err := h.svc.RegisterUser(ctx, auth.RegisterUserRequest{
		Email:    req.Msg.Email,
		Password: req.Msg.Password,
	})
	if err != nil {
		code := mapServiceErrorCode(err)
		return nil, connect.NewError(code, err)
	}
	return connect.NewResponse(&authv1.RegisterUserResponse{
		User: userToProto(resp),
	}), nil
}

func (h *authHandler) LoginUser(ctx context.Context, req *connect.Request[authv1.LoginUserRequest]) (*connect.Response[authv1.LoginUserResponse], error) {
	resp, err := h.svc.LoginUser(ctx, auth.LoginUserRequest{
		Email:    req.Msg.Email,
		Password: req.Msg.Password,
	})
	if err != nil {
		code := mapServiceErrorCode(err)
		return nil, connect.NewError(code, err)
	}
	return connect.NewResponse(&authv1.LoginUserResponse{
		Token: resp.Token,
		User:  userToProto(resp.User),
	}), nil
}

func (h *authHandler) GetUserProfile(ctx context.Context, req *connect.Request[authv1.GetUserProfileRequest]) (*connect.Response[authv1.GetUserProfileResponse], error) {
	resp, err := h.svc.GetUserProfile(ctx, auth.GetUserProfileRequest{
		UserID: req.Msg.UserId,
	})
	if err != nil {
		code := mapServiceErrorCode(err)
		return nil, connect.NewError(code, err)
	}
	return connect.NewResponse(&authv1.GetUserProfileResponse{
		User: userToProto(resp),
	}), nil
}

// userToProto converts UserResponse to protobuf User message
func userToProto(user *auth.UserResponse) *authv1.User {
	proto := &authv1.User{
		Id:        user.ID,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: timestampFromTime(user.CreatedAt),
	}
	if user.LastLoginAt != nil {
		proto.LastLoginAt = timestampFromTime(*user.LastLoginAt)
	}
	return proto
}

// timestampFromTime converts time.Time to protobuf Timestamp
func timestampFromTime(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}

// loadRSAKeys loads RSA keys from environment or generates them dynamically
func loadRSAKeys() (*rsa.PrivateKey, *rsa.PublicKey, string, error) {
	privateKeyPEM := os.Getenv("RSA_PRIVATE_KEY")
	publicKeyPEM := os.Getenv("RSA_PUBLIC_KEY")

	var privateKey *rsa.PrivateKey
	var publicKey *rsa.PublicKey
	var err error

	if privateKeyPEM != "" && publicKeyPEM != "" {
		// Parse from environment
		privateKey, err = parsePrivateKey(privateKeyPEM)
		if err != nil {
			return nil, nil, "", fmt.Errorf("failed to parse private key: %w", err)
		}
		publicKey, err = parsePublicKey(publicKeyPEM)
		if err != nil {
			return nil, nil, "", fmt.Errorf("failed to parse public key: %w", err)
		}
		slog.Info("loaded RSA keys from environment")
	} else {
		// Generate dynamically (dev/test only)
		privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, nil, "", fmt.Errorf("failed to generate key pair: %w", err)
		}
		publicKey = &privateKey.PublicKey
		slog.Warn("generated RSA key pair dynamically (not for production)")
	}

	// Generate key ID from public key
	kid := generateKeyID(publicKey)
	return privateKey, publicKey, kid, nil
}

func parsePrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("not an RSA private key")
	}
	return rsaKey, nil
}

func parsePublicKey(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}
	return rsaKey, nil
}

func generateKeyID(publicKey *rsa.PublicKey) string {
	pubBytes, _ := x509.MarshalPKIXPublicKey(publicKey)
	hash := sha256.Sum256(pubBytes)
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// extractRSAModulus extracts the modulus (n) from RSA public key as base64url
func extractRSAModulus(publicKey *rsa.PublicKey) string {
	return base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes())
}

// extractRSAExponent extracts the exponent (e) from RSA public key as base64url
func extractRSAExponent(publicKey *rsa.PublicKey) string {
	eBytes := make([]byte, 3)
	big.NewInt(int64(publicKey.E)).FillBytes(eBytes)
	// Trim leading zeros
	i := 0
	for i < len(eBytes) && eBytes[i] == 0 {
		i++
	}
	if i == len(eBytes) {
		return base64.RawURLEncoding.EncodeToString([]byte{0})
	}
	return base64.RawURLEncoding.EncodeToString(eBytes[i:])
}

// mapServiceErrorCode converts service layer errors to gRPC error codes
func mapServiceErrorCode(err error) connect.Code {
	if err == nil {
		return connect.CodeInternal
	}

	errMsg := err.Error()

	// Validation errors
	if errMsg == "email and password are required" ||
		errMsg == "user_id is required" ||
		errMsg == "user_id and pokemon_id are required" {
		return connect.CodeInvalidArgument
	}

	// Duplicate key error
	if errMsg == "email already exists" {
		return connect.CodeAlreadyExists
	}

	// Authentication errors
	if errMsg == "invalid email or password" {
		return connect.CodeUnauthenticated
	}

	// Not found errors
	if errMsg == "user not found" {
		return connect.CodeNotFound
	}

	// Internal errors (default)
	return connect.CodeInternal
}
