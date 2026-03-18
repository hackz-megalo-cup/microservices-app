//go:build integration

// Package e2e は、ゲスト登録→スターター選択→レイド→バトル→捕獲→ポケモン所持確認の
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
	itemv1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/item/v1"
	lobbyv1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/lobby/v1"
	raidlobbyv1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/raid_lobby/v1"

	"github.com/hackz-megalo-cup/microservices-app/services/internal/auth"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/capture"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/item"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/lobby"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
	raidlobby "github.com/hackz-megalo-cup/microservices-app/services/internal/raid_lobby"
)

// スターターポケモンID (Go)
const starterPokemonID = "00000000-0000-0000-0000-000000000001"

// スターターアイテムID一覧
var starterItemIDs = []string{
	"018f4e1a-0001-7000-8000-000000000001", // どりーさん
	"018f4e1a-0002-7000-8000-000000000002", // ざつくん
	"018f4e1a-0003-7000-8000-000000000003", // レッドブル
	"018f4e1a-0004-7000-8000-000000000004", // モンスター
	"018f4e1a-0005-7000-8000-000000000005", // こんにゃく
	"018f4e1a-0006-7000-8000-000000000006", // クッション
	"018f4e1a-0007-7000-8000-000000000007", // ひよこ
}

// TestGuestToCaptureFlow はゲストアカウント作成からポケモン捕獲までの
// 完全なE2Eフローをテストする。
//
//  1. ゲスト登録 + ログイン
//  2. スターターポケモン選択 + アクティブ設定 + アイテム付与
//  3. レイド作成 + 参加 + バトル
//  4. キャプチャセッション + 捕獲
//  5. 最終検証(2匹所持)
func TestGuestToCaptureFlow(t *testing.T) {
	ctx := context.Background()

	// ── Infrastructure ──────────────────────────────────────────
	adminPool, baseConnStr := startPostgres(t)

	authPool, authConnStr := createDB(t, adminPool, baseConnStr, "auth_e2e")
	raidPool, raidConnStr := createDB(t, adminPool, baseConnStr, "raid_e2e")
	capturePool, captureConnStr := createDB(t, adminPool, baseConnStr, "capture_e2e")
	itemPool, itemConnStr := createDB(t, adminPool, baseConnStr, "item_e2e")
	lobbyPool, lobbyConnStr := createDB(t, adminPool, baseConnStr, "lobby_e2e")

	runMigrations(t, authConnStr, auth.MigrationsFS)
	runMigrations(t, raidConnStr, raidlobby.MigrationsFS)
	runMigrations(t, captureConnStr, capture.MigrationsFS)
	runMigrations(t, itemConnStr, item.MigrationsFS)
	runMigrations(t, lobbyConnStr, lobby.MigrationsFS)

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
	itemSvc := item.NewService(
		platform.NewEventStore(itemPool),
		platform.NewOutboxStore(itemPool, nil),
		itemPool,
	)
	// lobby service with nil authClient — skips ownership check
	lobbySvc := lobby.NewService(
		platform.NewEventStore(lobbyPool),
		platform.NewOutboxStore(lobbyPool, nil),
		lobbyPool, nil, nil, nil, nil,
	)

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

	// ── Step 3: スターターポケモン選択 ──────────────────────────
	_, err = authSvc.ChooseStarter(ctx, connect.NewRequest(&authv1.ChooseStarterRequest{
		UserId:    userID,
		PokemonId: starterPokemonID,
	}))
	if err != nil {
		t.Fatalf("ChooseStarter: %v", err)
	}
	t.Logf("[3] starter chosen: %s", starterPokemonID)

	// ── Step 4: アクティブポケモン設定 ──────────────────────────
	setActiveResp, err := lobbySvc.SetActivePokemon(ctx, connect.NewRequest(&lobbyv1.SetActivePokemonRequest{
		UserId:    userID,
		PokemonId: starterPokemonID,
	}))
	if err != nil {
		t.Fatalf("SetActivePokemon: %v", err)
	}
	if !setActiveResp.Msg.Success {
		t.Fatal("expected SetActivePokemon success=true")
	}
	t.Logf("[4] active pokemon set: %s", starterPokemonID)

	// ── Step 5: スターターアイテム付与 ──────────────────────────
	for _, itemID := range starterItemIDs {
		_, err := itemSvc.GrantItem(ctx, connect.NewRequest(&itemv1.GrantItemRequest{
			UserId:   userID,
			ItemId:   itemID,
			Quantity: 1,
			Reason:   "starter_bonus",
		}))
		if err != nil {
			t.Fatalf("GrantItem(%s): %v", itemID, err)
		}
	}
	t.Logf("[5] %d starter items granted", len(starterItemIDs))

	// ── Step 6: スターター所持確認 ──────────────────────────────
	pokemonResp, err := authSvc.GetUserPokemon(ctx, connect.NewRequest(&authv1.GetUserPokemonRequest{
		UserId: userID,
	}))
	if err != nil {
		t.Fatalf("GetUserPokemon: %v", err)
	}
	if len(pokemonResp.Msg.PokemonIds) != 1 {
		t.Fatalf("expected 1 pokemon, got %d: %v", len(pokemonResp.Msg.PokemonIds), pokemonResp.Msg.PokemonIds)
	}
	if pokemonResp.Msg.PokemonIds[0] != starterPokemonID {
		t.Fatalf("expected starter %s, got %s", starterPokemonID, pokemonResp.Msg.PokemonIds[0])
	}
	t.Logf("[6] starter ownership verified")

	// ── Step 7: レイド作成 ──────────────────────────────────────
	bossPokemonID := uuid.NewString()

	createRaidResp, err := raidSvc.CreateRaid(ctx, connect.NewRequest(&raidlobbyv1.CreateRaidRequest{
		BossPokemonId: bossPokemonID,
	}))
	if err != nil {
		t.Fatalf("CreateRaid: %v", err)
	}
	lobbyID := createRaidResp.Msg.LobbyId
	t.Logf("[7] raid created: lobby=%s boss=%s", lobbyID, bossPokemonID)

	// ── Step 8: レイド参加 ──────────────────────────────────────
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
	t.Logf("[8] joined raid: participant=%s", joinResp.Msg.ParticipantId)

	// ── Step 9: バトル開始 ──────────────────────────────────────
	startResp, err := raidSvc.StartBattle(ctx, connect.NewRequest(&raidlobbyv1.StartBattleRequest{
		LobbyId: lobbyID,
	}))
	if err != nil {
		t.Fatalf("StartBattle: %v", err)
	}
	battleSessionID := startResp.Msg.BattleSessionId
	t.Logf("[9] battle started: session=%s", battleSessionID)

	// ── Step 10: バトル終了(勝利) ───────────────────────────────
	if err := raidSvc.HandleBattleFinished(ctx, lobbyID, battleSessionID, "win"); err != nil {
		t.Fatalf("HandleBattleFinished (raid): %v", err)
	}
	var raidStatus string
	if err := raidPool.QueryRow(ctx, `SELECT status FROM raid_lobby WHERE id = $1`, lobbyID).Scan(&raidStatus); err != nil {
		t.Fatalf("query raid_lobby status: %v", err)
	}
	if raidStatus != "finished" {
		t.Errorf("raid status: got %q, want %q", raidStatus, "finished")
	}
	t.Logf("[10] battle finished (win), raid status=%s", raidStatus)

	// ── Step 11: キャプチャセッション作成 ────────────────────────
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
	t.Logf("[11] capture session created: %s (rate=1.0)", captureSessionID)

	// ── Step 12: セッション取得で確認 ────────────────────────────
	getResp, err := captureSvc.GetCaptureSession(ctx, connect.NewRequest(&capturev1.GetCaptureSessionRequest{
		SessionId: captureSessionID,
	}))
	if err != nil {
		t.Fatalf("GetCaptureSession: %v", err)
	}
	if getResp.Msg.Result != "pending" {
		t.Errorf("capture session result: got %q, want %q", getResp.Msg.Result, "pending")
	}
	t.Logf("[12] capture session verified: status=%s rate=%.1f", getResp.Msg.Result, getResp.Msg.CurrentRate)

	// ── Step 13: ボールを投げる(捕獲) ────────────────────────────
	throwResp, err := captureSvc.ThrowBall(ctx, connect.NewRequest(&capturev1.ThrowBallRequest{
		SessionId: captureSessionID,
	}))
	if err != nil {
		t.Fatalf("ThrowBall: %v", err)
	}
	if throwResp.Msg.Result != "success" {
		t.Fatalf("ThrowBall result: got %q, want %q", throwResp.Msg.Result, "success")
	}
	t.Logf("[13] ball thrown: result=%s", throwResp.Msg.Result)

	// ── Step 14: セッション終了 ─────────────────────────────────
	endResp, err := captureSvc.EndSession(ctx, connect.NewRequest(&capturev1.EndSessionRequest{
		SessionId: captureSessionID,
	}))
	if err != nil {
		t.Fatalf("EndSession: %v", err)
	}
	if endResp.Msg.Result != "success" {
		t.Fatalf("EndSession result: got %q, want %q", endResp.Msg.Result, "success")
	}
	t.Logf("[14] capture session ended: result=%s", endResp.Msg.Result)

	// ── Step 15: 捕獲ポケモン登録(Kafkaコンシューマの模擬) ──────
	if err := authSvc.RegisterPokemon(ctx, userID, bossPokemonID); err != nil {
		t.Fatalf("RegisterPokemon: %v", err)
	}
	t.Logf("[15] captured pokemon registered to user")

	// ── Step 16: 最終検証 — 2匹所持確認 ─────────────────────────
	finalResp, err := authSvc.GetUserPokemon(ctx, connect.NewRequest(&authv1.GetUserPokemonRequest{
		UserId: userID,
	}))
	if err != nil {
		t.Fatalf("GetUserPokemon (final): %v", err)
	}
	if len(finalResp.Msg.PokemonIds) != 2 {
		t.Fatalf("expected 2 pokemon, got %d: %v", len(finalResp.Msg.PokemonIds), finalResp.Msg.PokemonIds)
	}
	foundStarter, foundCaptured := false, false
	for _, pid := range finalResp.Msg.PokemonIds {
		if pid == starterPokemonID {
			foundStarter = true
		}
		if pid == bossPokemonID {
			foundCaptured = true
		}
	}
	if !foundStarter {
		t.Errorf("starter pokemon %s not found in collection", starterPokemonID)
	}
	if !foundCaptured {
		t.Errorf("captured pokemon %s not found in collection", bossPokemonID)
	}
	t.Logf("[16] final verification: 2 pokemon owned (starter + captured)")

	// ── Step 17: イベントストア整合性検証 ────────────────────────
	verifyEventCount(t, authPool, userID, "user.registered", 1)
	verifyEventCount(t, authPool, userID, "user.logged_in", 1)
	verifyEventCount(t, raidPool, lobbyID, "raid_lobby.created", 1)
	verifyEventCount(t, raidPool, lobbyID, "raid.user_joined", 1)
	verifyEventCount(t, raidPool, lobbyID, "raid.battle_started", 1)
	verifyEventCount(t, raidPool, lobbyID, "raid_lobby.finished", 1)
	verifyEventCount(t, capturePool, captureSessionID, "capture.started", 1)
	verifyEventCount(t, capturePool, captureSessionID, "capture.ball_thrown", 1)
	verifyEventCount(t, capturePool, captureSessionID, "capture.completed", 1)
	t.Logf("[17] all event store entries verified!")
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
