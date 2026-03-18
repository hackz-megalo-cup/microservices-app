//go:build integration

// Package e2e は、ゲスト登録→レイド→バトル→捕獲→ポケモン所持確認の
// フルフロー統合テストを提供する。
package e2e

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io/fs"
	"net/url"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	authv1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/auth/v1"
	capturev1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/capture/v1"
	raidlobbyv1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/raid_lobby/v1"

	"github.com/hackz-megalo-cup/microservices-app/services/internal/auth"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/capture"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
	raidlobby "github.com/hackz-megalo-cup/microservices-app/services/internal/raid_lobby"
)

// TestGuestToCaptureFlow はゲストアカウント作成からポケモン捕獲まで
// の完全なE2Eフローをテストする。
//
//  1. ゲスト登録 + ログイン
//  2. レイド作成 + 参加
//  3. バトル開始 + 終了(勝利)
//  4. キャプチャセッション生成 + ボール投げ(捕獲成功)
//  5. ポケモン所持登録 + 確認
func TestGuestToCaptureFlow(t *testing.T) {
	ctx := context.Background()

	// ── Infrastructure ──────────────────────────────────────────
	adminPool, baseConnStr := startPostgres(t)

	authPool, authConnStr := createDB(t, adminPool, baseConnStr, "auth_e2e")
	raidPool, raidConnStr := createDB(t, adminPool, baseConnStr, "raid_e2e")
	capturePool, captureConnStr := createDB(t, adminPool, baseConnStr, "capture_e2e")

	runMigrations(t, authConnStr, auth.MigrationsFS)
	runMigrations(t, raidConnStr, raidlobby.MigrationsFS)
	runMigrations(t, captureConnStr, capture.MigrationsFS)

	// ── Services ────────────────────────────────────────────────
	privateKey, publicKey := generateRSAKeys(t)

	authSvc := auth.NewService(
		platform.NewEventStore(authPool),
		platform.NewOutboxStore(authPool, nil),
		authPool, privateKey, publicKey, "e2e-kid",
	)
	raidSvc := raidlobby.NewService(
		platform.NewEventStore(raidPool),
		platform.NewOutboxStore(raidPool, nil),
		raidPool, nil, nil,
	)
	captureSvc := capture.NewService(
		platform.NewEventStore(capturePool),
		platform.NewOutboxStore(capturePool, nil),
		capturePool, nil, nil,
	)
	captureES := platform.NewEventStore(capturePool)
	captureOB := platform.NewOutboxStore(capturePool, nil)

	// ── Step 1: ゲスト登録 ──────────────────────────────────────
	guestEmail := fmt.Sprintf("guest_%s@guest.local", uuid.NewString()[:8])
	guestPassword := uuid.NewString()

	registerResp, err := authSvc.RegisterUser(ctx, connect.NewRequest(&authv1.RegisterUserRequest{
		Email:    guestEmail,
		Password: guestPassword,
		Name:     "テストトレーナー",
	}))
	if err != nil {
		t.Fatalf("RegisterUser: %v", err)
	}
	userID := registerResp.Msg.User.Id
	if userID == "" {
		t.Fatal("expected non-empty user ID")
	}
	if registerResp.Msg.User.Name != "テストトレーナー" {
		t.Errorf("name: got %q, want %q", registerResp.Msg.User.Name, "テストトレーナー")
	}
	t.Logf("[1] guest registered: %s", userID)

	// ── Step 2: ログイン ────────────────────────────────────────
	loginResp, err := authSvc.LoginUser(ctx, connect.NewRequest(&authv1.LoginUserRequest{
		Email:    guestEmail,
		Password: guestPassword,
	}))
	if err != nil {
		t.Fatalf("LoginUser: %v", err)
	}
	if loginResp.Msg.Token == "" {
		t.Fatal("expected non-empty JWT token")
	}
	t.Logf("[2] guest logged in, JWT issued")

	// ── Step 3: レイド作成 ──────────────────────────────────────
	bossPokemonID := uuid.NewString()

	createRaidResp, err := raidSvc.CreateRaid(ctx, connect.NewRequest(&raidlobbyv1.CreateRaidRequest{
		BossPokemonId: bossPokemonID,
	}))
	if err != nil {
		t.Fatalf("CreateRaid: %v", err)
	}
	lobbyID := createRaidResp.Msg.LobbyId
	t.Logf("[3] raid created: lobby=%s boss=%s", lobbyID, bossPokemonID)

	// ── Step 4: レイド参加 ──────────────────────────────────────
	joinResp, err := raidSvc.JoinRaid(ctx, connect.NewRequest(&raidlobbyv1.JoinRaidRequest{
		LobbyId: lobbyID,
		UserId:  userID,
	}))
	if err != nil {
		t.Fatalf("JoinRaid: %v", err)
	}
	if joinResp.Msg.ParticipantId == "" {
		t.Fatal("expected non-empty participant ID")
	}
	t.Logf("[4] joined raid: participant=%s", joinResp.Msg.ParticipantId)

	// ── Step 5: バトル開始 ──────────────────────────────────────
	startResp, err := raidSvc.StartBattle(ctx, connect.NewRequest(&raidlobbyv1.StartBattleRequest{
		LobbyId: lobbyID,
	}))
	if err != nil {
		t.Fatalf("StartBattle: %v", err)
	}
	battleSessionID := startResp.Msg.BattleSessionId
	t.Logf("[5] battle started: session=%s", battleSessionID)

	// ── Step 6: バトル終了(勝利) ────────────────────────────────
	if err := raidSvc.HandleBattleFinished(ctx, lobbyID, battleSessionID, "win"); err != nil {
		t.Fatalf("HandleBattleFinished (raid): %v", err)
	}

	// raid_lobby status が finished になっていることを確認
	var raidStatus string
	if err := raidPool.QueryRow(ctx, `SELECT status FROM raid_lobby WHERE id = $1`, lobbyID).Scan(&raidStatus); err != nil {
		t.Fatalf("query raid_lobby status: %v", err)
	}
	if raidStatus != "finished" {
		t.Errorf("raid status: got %q, want %q", raidStatus, "finished")
	}
	t.Logf("[6] battle finished (win), raid status=%s", raidStatus)

	// ── Step 7: キャプチャセッション作成 ────────────────────────
	// バトル勝利後、Kafkaコンシューマが HandleBattleFinished を呼ぶ流れを
	// 模擬する。テストでは捕獲率を1.0にして確実に成功させる。
	captureSessionID := uuid.NewString()
	captureAgg := capture.NewCaptureAggregate(captureSessionID)
	captureAgg.Start(battleSessionID, userID, bossPokemonID, 1.0)
	if err := platform.SaveAggregate(ctx, captureES, captureOB, captureAgg, capture.CaptureTopicMapper); err != nil {
		t.Fatalf("save capture aggregate: %v", err)
	}
	if _, err := capturePool.Exec(ctx,
		`INSERT INTO capture_session (id, battle_session_id, user_id, pokemon_id, base_rate, current_rate, result, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, 'pending', $7)`,
		captureSessionID, battleSessionID, userID, bossPokemonID, 1.0, 1.0, time.Now().UTC(),
	); err != nil {
		t.Fatalf("insert capture_session: %v", err)
	}
	t.Logf("[7] capture session created: %s (rate=1.0)", captureSessionID)

	// ── Step 8: セッション取得で確認 ────────────────────────────
	getResp, err := captureSvc.GetCaptureSession(ctx, connect.NewRequest(&capturev1.GetCaptureSessionRequest{
		SessionId: captureSessionID,
	}))
	if err != nil {
		t.Fatalf("GetCaptureSession: %v", err)
	}
	if getResp.Msg.Result != "pending" {
		t.Errorf("capture session result: got %q, want %q", getResp.Msg.Result, "pending")
	}
	if getResp.Msg.UserId != userID {
		t.Errorf("capture session user_id: got %q, want %q", getResp.Msg.UserId, userID)
	}
	t.Logf("[8] capture session verified: status=%s rate=%.1f", getResp.Msg.Result, getResp.Msg.CurrentRate)

	// ── Step 9: ボールを投げる(捕獲) ────────────────────────────
	throwResp, err := captureSvc.ThrowBall(ctx, connect.NewRequest(&capturev1.ThrowBallRequest{
		SessionId: captureSessionID,
	}))
	if err != nil {
		t.Fatalf("ThrowBall: %v", err)
	}
	if throwResp.Msg.Result != "success" {
		t.Fatalf("ThrowBall result: got %q, want %q", throwResp.Msg.Result, "success")
	}
	t.Logf("[9] ball thrown: result=%s", throwResp.Msg.Result)

	// ── Step 10: セッション終了 ─────────────────────────────────
	endResp, err := captureSvc.EndSession(ctx, connect.NewRequest(&capturev1.EndSessionRequest{
		SessionId: captureSessionID,
	}))
	if err != nil {
		t.Fatalf("EndSession: %v", err)
	}
	if endResp.Msg.Result != "success" {
		t.Fatalf("EndSession result: got %q, want %q", endResp.Msg.Result, "success")
	}
	t.Logf("[10] capture session ended: result=%s", endResp.Msg.Result)

	// ── Step 11: ポケモン登録(Kafkaコンシューマの模擬) ──────────
	if err := authSvc.RegisterPokemon(ctx, userID, bossPokemonID); err != nil {
		t.Fatalf("RegisterPokemon: %v", err)
	}
	t.Logf("[11] pokemon registered to user")

	// ── Step 12: ポケモン所持確認 ───────────────────────────────
	pokemonResp, err := authSvc.GetUserPokemon(ctx, connect.NewRequest(&authv1.GetUserPokemonRequest{
		UserId: userID,
	}))
	if err != nil {
		t.Fatalf("GetUserPokemon: %v", err)
	}

	found := false
	for _, pid := range pokemonResp.Msg.PokemonIds {
		if pid == bossPokemonID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("pokemon %s not found in user's collection: %v", bossPokemonID, pokemonResp.Msg.PokemonIds)
	}
	t.Logf("[12] pokemon ownership verified!")

	// ── Step 13: DB 上のイベントストア検証 ──────────────────────
	verifyEventCount(t, authPool, userID, "user.registered", 1)
	verifyEventCount(t, authPool, userID, "user.logged_in", 1)
	verifyEventCount(t, raidPool, lobbyID, "raid_lobby.created", 1)
	verifyEventCount(t, raidPool, lobbyID, "raid.user_joined", 1)
	verifyEventCount(t, raidPool, lobbyID, "raid.battle_started", 1)
	verifyEventCount(t, raidPool, lobbyID, "raid_lobby.finished", 1)
	verifyEventCount(t, capturePool, captureSessionID, "capture.started", 1)
	verifyEventCount(t, capturePool, captureSessionID, "capture.ball_thrown", 1)
	verifyEventCount(t, capturePool, captureSessionID, "capture.completed", 1)
	t.Logf("[13] all event store entries verified!")
}

// ─── Helpers ────────────────────────────────────────────────────

func startPostgres(t *testing.T) (*pgxpool.Pool, string) {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("e2e_default"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to connect to postgres: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	return pool, connStr
}

func createDB(t *testing.T, adminPool *pgxpool.Pool, baseConnStr, dbName string) (*pgxpool.Pool, string) {
	t.Helper()
	ctx := context.Background()

	if _, err := adminPool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName)); err != nil {
		t.Fatalf("CREATE DATABASE %s: %v", dbName, err)
	}

	connStr := replaceDBInConnStr(baseConnStr, dbName)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to connect to %s: %v", dbName, err)
	}
	t.Cleanup(func() { pool.Close() })

	return pool, connStr
}

func replaceDBInConnStr(connStr, newDB string) string {
	u, err := url.Parse(connStr)
	if err != nil {
		panic(fmt.Sprintf("invalid connection string: %v", err))
	}
	u.Path = "/" + newDB
	return u.String()
}

func runMigrations(t *testing.T, connStr string, migrationsFS fs.ReadFileFS) {
	t.Helper()
	subFS, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		t.Fatalf("fs.Sub migrations: %v", err)
	}
	if err := platform.RunMigrations(connStr, subFS); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
}

func generateRSAKeys(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	return privateKey, &privateKey.PublicKey
}

func verifyEventCount(t *testing.T, pool *pgxpool.Pool, streamID, eventType string, want int) {
	t.Helper()
	var count int
	err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM event_store WHERE stream_id = $1 AND event_type = $2`,
		streamID, eventType,
	).Scan(&count)
	if err != nil {
		t.Fatalf("query event_store (%s/%s): %v", streamID, eventType, err)
	}
	if count != want {
		t.Errorf("event_store %s: got %d, want %d", eventType, count, want)
	}
}
