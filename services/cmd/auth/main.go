package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
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
	dbPool := platform.InitDB(ctx, os.Getenv("DATABASE_URL"), migrationsFS, serviceName)
	if dbPool != nil {
		defer dbPool.Close()
	}

	var sqlDB *sql.DB
	if dbPool != nil {
		sqlDB = stdlib.OpenDBFromPool(dbPool)
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

	mux := http.NewServeMux()
	mux.Handle(path, handler)
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

	// Start Kafka consumer for capture.caught events
	kafkaConsumer, _ := platform.NewKafkaConsumer(
		ctx,
		brokers,
		"auth-service-consumer",
		[]string{platform.TopicCaptureCaught},
	)
	if kafkaConsumer != nil {
		go func() {
			if err := auth.RunConsumer(ctx, auth.ConsumerConfig{
				Client: kafkaConsumer,
				Repo:   repo,
			}); err != nil && !errors.Is(err, context.Canceled) {
				slog.Error("kafka consumer error", "error", err)
			}
		}()
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		BaseContext:  func(net.Listener) context.Context { return ctx },
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 30 * time.Second,
		Handler:      h2c.NewHandler(mux, &http2.Server{}),
	}

	srvErr := make(chan error, 1)
	go func() {
		logger.InfoContext(ctx, "starting auth service", "port", port)
		srvErr <- srv.ListenAndServe()
	}()

	select {
	case err = <-srvErr:
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
		return nil, connect.NewError(connect.CodeInternal, err)
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
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
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
		return nil, connect.NewError(connect.CodeNotFound, err)
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
