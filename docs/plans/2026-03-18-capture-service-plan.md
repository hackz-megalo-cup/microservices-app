# Capture Service Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement the capture service (issue #16) — all sub-tasks 6-1 through 6-5 on `feat/capture-context` branch.

**Architecture:** Event-sourced Go service following the existing platform pattern. Subscribes to `battle.finished` Kafka events to create capture sessions, exposes 4 RPCs (GetCaptureSession, UseItem, ThrowBall, EndSession) via connect-rpc. Depends on masterdata and item services for item effects and pokemon data.

**Tech Stack:** Go, connect-rpc, PostgreSQL, Kafka (Redpanda), platform event sourcing framework

**Design doc:** `docs/plans/2026-03-18-capture-service-design.md`

---

## Task 1: Scaffold service with new-service.sh (Issue #34 — 6-1)

**Files:**
- Run: `scripts/new-service.sh go capture 8088`
- Auto-generated (DO NOT EDIT): `services/cmd/capture/main.go`, `services/internal/capture/embed.go`, migrations 001-005
- Auto-wired: `docker-compose.yml`, `scripts/init-db.sh`, `services/internal/platform/topics.go`, `deploy/` configs

**Step 1: Run scaffold script**

```bash
cd /Users/thirdlf03/src/github.com/hackz-megalo-cup/microservices-app
scripts/new-service.sh go capture 8088
```

**Step 2: Verify scaffold output**

```bash
ls services/cmd/capture/main.go
ls services/internal/capture/embed.go
ls services/internal/capture/events.go
ls services/internal/capture/aggregate.go
ls services/internal/capture/service.go
ls services/internal/capture/migrations/
ls deploy/docker/capture/Dockerfile
ls proto/capture/v1/capture.proto
```

**Step 3: Verify build**

```bash
cd services && go build ./cmd/capture
```

**Step 4: Commit scaffold**

```bash
git add -A
git commit -m "feat(capture): scaffold capture service with new-service.sh (#34)"
```

---

## Task 2: Define proto and generate code (Issue #34 — 6-1 continued)

**Files:**
- Modify: `proto/capture/v1/capture.proto`
- Generated: `services/gen/go/capture/v1/`, `frontend/src/gen/capture/`

**Step 1: Replace proto with capture-specific definition**

Replace the content of `proto/capture/v1/capture.proto` with:

```protobuf
syntax = "proto3";

package capture.v1;

option go_package = "github.com/hackz-megalo-cup/microservices-app/services/gen/go/capture/v1;capturev1";

service CaptureService {
  rpc GetCaptureSession(GetCaptureSessionRequest) returns (GetCaptureSessionResponse) {}
  rpc UseItem(UseItemRequest) returns (UseItemResponse) {}
  rpc ThrowBall(ThrowBallRequest) returns (ThrowBallResponse) {}
  rpc EndSession(EndSessionRequest) returns (EndSessionResponse) {}
}

message GetCaptureSessionRequest {
  string session_id = 1;
}

message CaptureAction {
  string id = 1;
  string action_type = 2;
  string item_id = 3;
  double rate_change = 4;
  string created_at = 5;
}

message GetCaptureSessionResponse {
  string session_id = 1;
  string battle_session_id = 2;
  string user_id = 3;
  string pokemon_id = 4;
  double base_rate = 5;
  double current_rate = 6;
  string result = 7;
  repeated CaptureAction actions = 8;
}

message UseItemRequest {
  string session_id = 1;
  string item_id = 2;
}

message UseItemResponse {
  double rate_before = 1;
  double rate_after = 2;
  bool escaped = 3;
}

message ThrowBallRequest {
  string session_id = 1;
}

message ThrowBallResponse {
  string result = 1;
}

message EndSessionRequest {
  string session_id = 1;
}

message EndSessionResponse {
  string result = 1;
}
```

**Step 2: Generate code**

```bash
buf lint && buf generate
find services/gen -name '*.go' -exec gofmt -w {} +
```

**Step 3: Verify build**

```bash
cd services && go build ./cmd/capture
```

**Step 4: Commit**

```bash
git add proto/capture/ services/gen/ frontend/src/gen/
git commit -m "feat(capture): define capture proto with 4 RPCs (#34)"
```

---

## Task 3: Add capture Kafka topics to platform (Issue #34 — 6-1 continued)

**Files:**
- Modify: `services/internal/platform/topics.go`

The scaffold auto-adds `capture.created`, `capture.failed`, `capture.compensated`. We need additional capture-specific topics.

**Step 1: Add topic constants**

Add before the `// Dead Letter Queue topics.` comment:

```go
TopicCaptureStarted   = "capture.started"
TopicCaptureItemUsed  = "capture.item_used"
TopicCaptureBallThrown = "capture.ball_thrown"
TopicCaptureCompleted = "capture.completed"
```

Note: `TopicCaptureCreated`, `TopicCaptureFailed`, `TopicCaptureCompensated` are already added by the scaffold. Rename `TopicCaptureCreated` to `TopicCaptureStarted` if the scaffold created it, or add `TopicCaptureStarted` separately.

**Step 2: Add DLQ topic**

Add `TopicCaptureCompletedDLQ = "capture.completed.dlq"` in the DLQ section.

**Step 3: Add to DLQTopic mapping**

```go
TopicCaptureCompleted: TopicCaptureCompletedDLQ,
```

**Step 4: Add to DefaultTopics**

```go
TopicCaptureStarted:      3,
TopicCaptureItemUsed:     3,
TopicCaptureBallThrown:   3,
TopicCaptureCompleted:    3,
TopicCaptureCompletedDLQ: 1,
```

**Step 5: Verify build**

```bash
cd services && go build ./cmd/capture
```

**Step 6: Commit**

```bash
git add services/internal/platform/topics.go
git commit -m "feat(capture): add capture Kafka topics to platform (#34)"
```

---

## Task 4: Implement events.go (Issue #35 — 6-2)

**Files:**
- Modify: `services/internal/capture/events.go`

**Step 1: Replace scaffold events with capture domain events**

```go
package capture

import "github.com/hackz-megalo-cup/microservices-app/services/internal/platform"

const (
	EventCaptureStarted   = "capture.started"
	EventCaptureItemUsed  = "capture.item_used"
	EventCaptureBallThrown = "capture.ball_thrown"
	EventCaptureCompleted = "capture.completed"

	EventCaptureFailed      = "capture.failed"      // main.go が参照 — 削除禁止
	EventCaptureCompensated = "capture.compensated" // main.go が参照 — 削除禁止
)

type StartedData struct {
	SessionID       string `json:"session_id"`
	BattleSessionID string `json:"battle_session_id"`
	UserID          string `json:"user_id"`
	PokemonID       string `json:"pokemon_id"`
	BaseRate        float64 `json:"base_rate"`
}

type ItemUsedData struct {
	SessionID  string  `json:"session_id"`
	ItemID     string  `json:"item_id"`
	RateBefore float64 `json:"rate_before"`
	RateAfter  float64 `json:"rate_after"`
}

type BallThrownData struct {
	SessionID string `json:"session_id"`
	Result    string `json:"result"`
}

type CompletedData struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	PokemonID string `json:"pokemon_id"`
	Result    string `json:"result"`
}

type FailedData struct {
	Input string `json:"input"`
	Error string `json:"error"`
}

type CompensatedData struct {
	Reason string `json:"reason"`
}

func CaptureTopicMapper(eventType string) string {
	switch eventType {
	case EventCaptureStarted:
		return platform.TopicCaptureStarted
	case EventCaptureItemUsed:
		return platform.TopicCaptureItemUsed
	case EventCaptureBallThrown:
		return platform.TopicCaptureBallThrown
	case EventCaptureCompleted:
		return platform.TopicCaptureCompleted
	case EventCaptureFailed:
		return platform.TopicCaptureFailed
	case EventCaptureCompensated:
		return platform.TopicCaptureCompensated
	default:
		return ""
	}
}
```

**Step 2: Verify build**

```bash
cd services && go build ./cmd/capture
```

**Step 3: Commit**

```bash
git add services/internal/capture/events.go
git commit -m "feat(capture): define capture domain events (#35)"
```

---

## Task 5: Implement aggregate.go (Issue #35 — 6-2)

**Files:**
- Modify: `services/internal/capture/aggregate.go`

**Step 1: Replace scaffold aggregate with capture domain aggregate**

```go
package capture

import (
	"encoding/json"
	"log/slog"

	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

type CaptureAggregate struct {
	platform.AggregateBase
	BattleSessionID string
	UserID          string
	PokemonID       string
	BaseRate        float64
	CurrentRate     float64
	Result          string // pending, success, fail, escaped
}

func NewCaptureAggregate(id string) *CaptureAggregate {
	return &CaptureAggregate{
		AggregateBase: platform.NewAggregateBase(id),
		Result:        "pending",
	}
}

func (a *CaptureAggregate) StreamType() string { return "capture" }

func (a *CaptureAggregate) ApplyEvent(eventType string, data json.RawMessage) {
	switch eventType {
	case EventCaptureStarted:
		var d StartedData
		if err := json.Unmarshal(data, &d); err != nil {
			slog.Warn("failed to unmarshal started data", "error", err)
		}
		a.BattleSessionID = d.BattleSessionID
		a.UserID = d.UserID
		a.PokemonID = d.PokemonID
		a.BaseRate = d.BaseRate
		a.CurrentRate = d.BaseRate
		a.Result = "pending"
	case EventCaptureItemUsed:
		var d ItemUsedData
		if err := json.Unmarshal(data, &d); err != nil {
			slog.Warn("failed to unmarshal item_used data", "error", err)
		}
		a.CurrentRate = d.RateAfter
	case EventCaptureBallThrown:
		var d BallThrownData
		if err := json.Unmarshal(data, &d); err != nil {
			slog.Warn("failed to unmarshal ball_thrown data", "error", err)
		}
		a.Result = d.Result
	case EventCaptureCompleted:
		var d CompletedData
		if err := json.Unmarshal(data, &d); err != nil {
			slog.Warn("failed to unmarshal completed data", "error", err)
		}
		a.Result = d.Result
	case EventCaptureFailed:
		a.Result = "failed"
	case EventCaptureCompensated:
		a.Result = "compensated"
	}
}

func (a *CaptureAggregate) Start(battleSessionID, userID, pokemonID string, baseRate float64) {
	a.Raise(EventCaptureStarted, StartedData{
		SessionID:       a.AggregateID(),
		BattleSessionID: battleSessionID,
		UserID:          userID,
		PokemonID:       pokemonID,
		BaseRate:        baseRate,
	})
	a.BattleSessionID = battleSessionID
	a.UserID = userID
	a.PokemonID = pokemonID
	a.BaseRate = baseRate
	a.CurrentRate = baseRate
	a.Result = "pending"
}

func (a *CaptureAggregate) UseItem(itemID string, rateBefore, rateAfter float64) {
	a.Raise(EventCaptureItemUsed, ItemUsedData{
		SessionID:  a.AggregateID(),
		ItemID:     itemID,
		RateBefore: rateBefore,
		RateAfter:  rateAfter,
	})
	a.CurrentRate = rateAfter
}

func (a *CaptureAggregate) ThrowBall(result string) {
	a.Raise(EventCaptureBallThrown, BallThrownData{
		SessionID: a.AggregateID(),
		Result:    result,
	})
	a.Result = result
}

func (a *CaptureAggregate) Complete(result string) {
	a.Raise(EventCaptureCompleted, CompletedData{
		SessionID: a.AggregateID(),
		UserID:    a.UserID,
		PokemonID: a.PokemonID,
		Result:    result,
	})
	a.Result = result
}

func (a *CaptureAggregate) Escape() {
	a.CurrentRate = 0
	a.Result = "escaped"
	a.Raise(EventCaptureCompleted, CompletedData{
		SessionID: a.AggregateID(),
		UserID:    a.UserID,
		PokemonID: a.PokemonID,
		Result:    "escaped",
	})
}

func (a *CaptureAggregate) Fail(input string, reason string) {
	a.Raise(EventCaptureFailed, FailedData{
		Input: input,
		Error: reason,
	})
	a.Result = "failed"
}

func (a *CaptureAggregate) Compensate(reason string) {
	if a.Result == "compensated" {
		return
	}
	a.Raise(EventCaptureCompensated, CompensatedData{
		Reason: reason,
	})
	a.Result = "compensated"
}
```

**Step 2: Verify build**

```bash
cd services && go build ./cmd/capture
```

**Step 3: Commit**

```bash
git add services/internal/capture/aggregate.go
git commit -m "feat(capture): implement capture aggregate with state machine (#35)"
```

---

## Task 6: Add domain migration + service.go with GetCaptureSession + HandleBattleFinished (Issue #35 — 6-2)

**Files:**
- Create: `services/internal/capture/migrations/006_capture_tables.up.sql`
- Create: `services/internal/capture/migrations/006_capture_tables.down.sql`
- Modify: `services/internal/capture/service.go`

**Step 1: Create domain migration (up)**

```sql
CREATE TABLE IF NOT EXISTS capture_session (
    id                UUID PRIMARY KEY,
    battle_session_id UUID NOT NULL,
    user_id           UUID NOT NULL,
    pokemon_id        UUID NOT NULL,
    base_rate         REAL NOT NULL,
    current_rate      REAL NOT NULL,
    result            TEXT NOT NULL DEFAULT 'pending',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS capture_action (
    id          UUID PRIMARY KEY,
    session_id  UUID NOT NULL REFERENCES capture_session(id),
    action_type TEXT NOT NULL,
    item_id     UUID,
    rate_change REAL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_capture_action_session ON capture_action (session_id);
CREATE INDEX IF NOT EXISTS idx_capture_session_user ON capture_session (user_id);
CREATE INDEX IF NOT EXISTS idx_capture_session_battle ON capture_session (battle_session_id);
```

**Step 2: Create domain migration (down)**

```sql
DROP TABLE IF EXISTS capture_action;
DROP TABLE IF EXISTS capture_session;
```

**Step 3: Implement service.go**

Replace scaffold `service.go` with the full implementation including:
- `Service` struct with `eventStore`, `outbox`, `db`, `masterdataClient`, `itemClient`
- `NewService` constructor
- `GetCaptureSession` — DB read from `capture_session` + `capture_action`
- `HandleBattleFinished` — Kafka consumer handler: create session per participant (base_rate=0.3)
- Stub `UseItem`, `ThrowBall`, `EndSession` (return unimplemented) to make build pass

```go
package capture

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	pb "github.com/hackz-megalo-cup/microservices-app/services/gen/go/capture/v1"
	"github.com/hackz-megalo-cup/microservices-app/services/gen/go/item/v1/itemv1connect"
	"github.com/hackz-megalo-cup/microservices-app/services/gen/go/masterdata/v1/masterdatav1connect"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

const defaultBaseRate = 0.3

type Service struct {
	eventStore       *platform.EventStore
	outbox           *platform.OutboxStore
	db               *pgxpool.Pool
	masterdataClient masterdatav1connect.MasterdataServiceClient
	itemClient       itemv1connect.ItemServiceClient
}

func NewService(
	eventStore *platform.EventStore,
	outbox *platform.OutboxStore,
	db *pgxpool.Pool,
	masterdataClient masterdatav1connect.MasterdataServiceClient,
	itemClient itemv1connect.ItemServiceClient,
) *Service {
	return &Service{
		eventStore:       eventStore,
		outbox:           outbox,
		db:               db,
		masterdataClient: masterdataClient,
		itemClient:       itemClient,
	}
}

func (s *Service) GetCaptureSession(ctx context.Context, req *connect.Request[pb.GetCaptureSessionRequest]) (*connect.Response[pb.GetCaptureSessionResponse], error) {
	sessionID := req.Msg.GetSessionId()
	if sessionID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("session_id is required"))
	}

	if s.db == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database not available"))
	}

	var resp pb.GetCaptureSessionResponse
	err := s.db.QueryRow(ctx,
		`SELECT id, battle_session_id, user_id, pokemon_id, base_rate, current_rate, result
		 FROM capture_session WHERE id = $1`, sessionID,
	).Scan(&resp.SessionId, &resp.BattleSessionId, &resp.UserId, &resp.PokemonId,
		&resp.BaseRate, &resp.CurrentRate, &resp.Result)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("session not found: %s", sessionID))
	}

	rows, err := s.db.Query(ctx,
		`SELECT id, action_type, COALESCE(item_id::text, ''), COALESCE(rate_change, 0), created_at
		 FROM capture_action WHERE session_id = $1 ORDER BY created_at ASC`, sessionID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var a pb.CaptureAction
			var createdAt time.Time
			if err := rows.Scan(&a.Id, &a.ActionType, &a.ItemId, &a.RateChange, &createdAt); err != nil {
				continue
			}
			a.CreatedAt = createdAt.Format(time.RFC3339)
			resp.Actions = append(resp.Actions, &a)
		}
	}

	return connect.NewResponse(&resp), nil
}

func (s *Service) HandleBattleFinished(ctx context.Context, battleSessionID, lobbyID, bossPokemonID string, participantUserIDs []string) error {
	for _, userID := range participantUserIDs {
		sessionID := uuid.NewString()

		agg := NewCaptureAggregate(sessionID)
		agg.Start(battleSessionID, userID, bossPokemonID, defaultBaseRate)

		if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, CaptureTopicMapper); err != nil {
			slog.Error("failed to save capture aggregate", "session_id", sessionID, "user_id", userID, "error", err)
			continue
		}

		if s.db != nil {
			if _, err := s.db.Exec(ctx,
				`INSERT INTO capture_session (id, battle_session_id, user_id, pokemon_id, base_rate, current_rate, result, created_at)
				 VALUES ($1, $2, $3, $4, $5, $6, 'pending', $7)`,
				sessionID, battleSessionID, userID, bossPokemonID, defaultBaseRate, defaultBaseRate, time.Now().UTC(),
			); err != nil {
				slog.Error("failed to insert capture_session", "session_id", sessionID, "error", err)
			}
		}

		slog.Info("capture session created", "session_id", sessionID, "user_id", userID, "pokemon_id", bossPokemonID)
	}
	return nil
}

func (s *Service) UseItem(ctx context.Context, req *connect.Request[pb.UseItemRequest]) (*connect.Response[pb.UseItemResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("not implemented"))
}

func (s *Service) ThrowBall(ctx context.Context, req *connect.Request[pb.ThrowBallRequest]) (*connect.Response[pb.ThrowBallResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("not implemented"))
}

func (s *Service) EndSession(ctx context.Context, req *connect.Request[pb.EndSessionRequest]) (*connect.Response[pb.EndSessionResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("not implemented"))
}
```

**Step 4: Update main.go**

Modify `services/cmd/capture/main.go` to:
- Import `masterdatav1connect`, `itemv1connect`
- Create masterdata and item clients from env vars
- Pass `db`, `masterdataClient`, `itemClient` to `NewService`
- Add `runBattleFinishedConsumer` goroutine (same pattern as raid-lobby)
- Update compensation handler to use `NewCaptureAggregate` and `CaptureTopicMapper`

The battle.finished consumer extracts: `session_id`, `lobby_id`, `boss_pokemon_id`, `result`, `participant_user_ids` from event data. Only processes `result=win`.

**Step 5: Verify build**

```bash
cd services && go build ./cmd/capture
```

**Step 6: Commit**

```bash
git add services/internal/capture/ services/cmd/capture/
git commit -m "feat(capture): implement battle.finished consumer and GetCaptureSession (#35)"
```

---

## Task 7: Implement UseItem RPC (Issue #36 — 6-3)

**Files:**
- Modify: `services/internal/capture/service.go`

**Step 1: Replace UseItem stub with full implementation**

Logic:
1. Load aggregate, verify `result == "pending"`
2. Call `itemClient.UseItem(user_id, item_id, 1)` to consume inventory
3. Call `masterdataClient.GetItem(item_id)` for `capture_rate_bonus`, `target_type`, `name`
4. Call `masterdataClient.GetPokemon(pokemon_id)` for boss `type`
5. Special: item name `"ざつくん"` + boss type `"python"` → escape (rate=0)
6. Normal: if `target_type` matches boss type → apply bonus; otherwise half bonus
7. Clamp `current_rate` to [0.0, 1.0]
8. Save aggregate + update read model (`capture_session.current_rate`, insert `capture_action`)

```go
func (s *Service) UseItem(ctx context.Context, req *connect.Request[pb.UseItemRequest]) (*connect.Response[pb.UseItemResponse], error) {
	sessionID := req.Msg.GetSessionId()
	itemID := req.Msg.GetItemId()
	if sessionID == "" || itemID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("session_id and item_id are required"))
	}

	agg := NewCaptureAggregate(sessionID)
	if err := platform.LoadAggregate(ctx, s.eventStore, agg); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("session not found"))
	}
	if agg.Result != "pending" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("session is not pending (result: %s)", agg.Result))
	}

	// Consume item from inventory
	userID := agg.UserID
	if s.itemClient != nil {
		_, err := s.itemClient.UseItem(ctx, connect.NewRequest(&itemv1pb.UseItemRequest{
			UserId: userID, ItemId: itemID, Quantity: 1,
		}))
		if err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("failed to use item: %w", err))
		}
	}

	// Get item metadata from masterdata
	var itemName, targetType string
	var captureRateBonus float64
	if s.masterdataClient != nil {
		itemResp, err := s.masterdataClient.GetItem(ctx, connect.NewRequest(&masterdatav1pb.GetItemRequest{Id: itemID}))
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get item metadata: %w", err))
		}
		item := itemResp.Msg.GetItem()
		itemName = item.GetName()
		targetType = item.GetTargetType()
		captureRateBonus = item.GetCaptureRateBonus()
	}

	// Get boss pokemon type
	var bossType string
	if s.masterdataClient != nil {
		pokemonResp, err := s.masterdataClient.GetPokemon(ctx, connect.NewRequest(&masterdatav1pb.GetPokemonRequest{Id: agg.PokemonID}))
		if err != nil {
			slog.Warn("failed to get pokemon metadata", "pokemon_id", agg.PokemonID, "error", err)
		} else {
			bossType = pokemonResp.Msg.GetPokemon().GetType()
		}
	}

	rateBefore := agg.CurrentRate
	var escaped bool

	// Special: ざつくん + python boss → escape
	if itemName == "ざつくん" && bossType == "python" {
		agg.UseItem(itemID, rateBefore, 0)
		agg.Escape()
		escaped = true
	} else {
		bonus := captureRateBonus
		if targetType != "" && targetType == bossType {
			// Type match: full bonus
		} else {
			bonus = bonus * 0.5
		}
		rateAfter := rateBefore + bonus
		if rateAfter > 1.0 {
			rateAfter = 1.0
		}
		if rateAfter < 0.0 {
			rateAfter = 0.0
		}
		agg.UseItem(itemID, rateBefore, rateAfter)
	}

	if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, CaptureTopicMapper); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save: %w", err))
	}

	// Update read model
	if s.db != nil {
		_, _ = s.db.Exec(ctx,
			`UPDATE capture_session SET current_rate = $1, result = $2 WHERE id = $3`,
			agg.CurrentRate, agg.Result, sessionID)
		_, _ = s.db.Exec(ctx,
			`INSERT INTO capture_action (id, session_id, action_type, item_id, rate_change, created_at)
			 VALUES ($1, $2, 'use_item', $3, $4, $5)`,
			uuid.NewString(), sessionID, itemID, agg.CurrentRate-rateBefore, time.Now().UTC())
	}

	return connect.NewResponse(&pb.UseItemResponse{
		RateBefore: rateBefore,
		RateAfter:  agg.CurrentRate,
		Escaped:    escaped,
	}), nil
}
```

Note: Import aliases needed — `itemv1pb` for item proto, `masterdatav1pb` for masterdata proto.

**Step 2: Verify build**

```bash
cd services && go build ./cmd/capture
```

**Step 3: Commit**

```bash
git add services/internal/capture/service.go
git commit -m "feat(capture): implement UseItem RPC with item effects and escape logic (#36)"
```

---

## Task 8: Implement ThrowBall RPC (Issue #37 — 6-4)

**Files:**
- Modify: `services/internal/capture/service.go`

**Step 1: Replace ThrowBall stub**

```go
func (s *Service) ThrowBall(ctx context.Context, req *connect.Request[pb.ThrowBallRequest]) (*connect.Response[pb.ThrowBallResponse], error) {
	sessionID := req.Msg.GetSessionId()
	if sessionID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("session_id is required"))
	}

	agg := NewCaptureAggregate(sessionID)
	if err := platform.LoadAggregate(ctx, s.eventStore, agg); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("session not found"))
	}
	if agg.Result != "pending" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("session is not pending (result: %s)", agg.Result))
	}

	// Random capture check
	var result string
	if rand.Float64() < agg.CurrentRate {
		result = "success"
	} else {
		result = "fail"
	}

	agg.ThrowBall(result)

	// On success, also emit capture.completed
	if result == "success" {
		agg.Complete("success")
	}

	if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, CaptureTopicMapper); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save: %w", err))
	}

	// Update read model
	if s.db != nil {
		_, _ = s.db.Exec(ctx,
			`UPDATE capture_session SET result = $1 WHERE id = $2`,
			agg.Result, sessionID)
		_, _ = s.db.Exec(ctx,
			`INSERT INTO capture_action (id, session_id, action_type, rate_change, created_at)
			 VALUES ($1, $2, 'throw_ball', 0, $3)`,
			uuid.NewString(), sessionID, time.Now().UTC())
	}

	return connect.NewResponse(&pb.ThrowBallResponse{
		Result: result,
	}), nil
}
```

Add `"math/rand"` to imports (or `"math/rand/v2"` for Go 1.22+).

**Step 2: Verify build**

```bash
cd services && go build ./cmd/capture
```

**Step 3: Commit**

```bash
git add services/internal/capture/service.go
git commit -m "feat(capture): implement ThrowBall RPC with random capture check (#37)"
```

---

## Task 9: Implement EndSession RPC (Issue #38 — 6-5)

**Files:**
- Modify: `services/internal/capture/service.go`

**Step 1: Replace EndSession stub**

```go
func (s *Service) EndSession(ctx context.Context, req *connect.Request[pb.EndSessionRequest]) (*connect.Response[pb.EndSessionResponse], error) {
	sessionID := req.Msg.GetSessionId()
	if sessionID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("session_id is required"))
	}

	agg := NewCaptureAggregate(sessionID)
	if err := platform.LoadAggregate(ctx, s.eventStore, agg); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("session not found"))
	}
	if agg.Result == "pending" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("session is still pending — throw a ball first"))
	}

	// Emit capture.completed if not already emitted (fail/escaped cases)
	if agg.Result == "fail" || agg.Result == "escaped" {
		agg.Complete(agg.Result)
		if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, CaptureTopicMapper); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save: %w", err))
		}
	}

	return connect.NewResponse(&pb.EndSessionResponse{
		Result: agg.Result,
	}), nil
}
```

**Step 2: Verify build**

```bash
cd services && go build ./cmd/capture
```

**Step 3: Commit**

```bash
git add services/internal/capture/service.go
git commit -m "feat(capture): implement EndSession RPC (#38)"
```

---

## Task 10: Update docker-compose and verify full build

**Files:**
- Modify: `docker-compose.yml` (add `MASTERDATA_URL`, `ITEM_URL` env vars, add `auth` middleware to Traefik labels)

**Step 1: Add env vars to capture service in docker-compose.yml**

Add to the capture service environment section:
```yaml
MASTERDATA_URL: "http://masterdata:8084"
ITEM_URL: "http://item:8085"
```

Add `auth@file` to the Traefik middlewares label:
```yaml
- "traefik.http.routers.capture.middlewares=cors@file,auth@file,rate-limit@file,retry@file"
```

**Step 2: Verify full Go build**

```bash
cd services && go build ./...
```

**Step 3: Verify docker build**

```bash
docker compose build capture
```

**Step 4: Commit**

```bash
git add docker-compose.yml
git commit -m "feat(capture): add masterdata/item URLs and auth middleware to docker-compose (#16)"
```

---

## Task 11: Final verification

**Step 1: Full lint**

```bash
cd services && golangci-lint run ./...
```

**Step 2: Full test**

```bash
cd services && go test ./...
```

**Step 3: Buf checks**

```bash
buf lint
```

**Step 4: If all green, final commit if any fixes needed**

```bash
git add -A
git commit -m "fix(capture): address lint and test issues"
```
