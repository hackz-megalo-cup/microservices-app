package capture

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	pb "github.com/hackz-megalo-cup/microservices-app/services/gen/go/capture/v1"
	itempb "github.com/hackz-megalo-cup/microservices-app/services/gen/go/item/v1"
	"github.com/hackz-megalo-cup/microservices-app/services/gen/go/item/v1/itemv1connect"
	masterdatapb "github.com/hackz-megalo-cup/microservices-app/services/gen/go/masterdata/v1"
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
		 FROM capture_session WHERE battle_session_id = $1`, sessionID,
	).Scan(&resp.SessionId, &resp.BattleSessionId, &resp.UserId, &resp.PokemonId,
		&resp.BaseRate, &resp.CurrentRate, &resp.Result)
	if err != nil {
		// Fallback: sessionID might be a lobbyId — find the latest pending session for this user
		userID := req.Header().Get("X-User-Id")
		if userID != "" {
			err = s.db.QueryRow(ctx,
				`SELECT id, battle_session_id, user_id, pokemon_id, base_rate, current_rate, result
				 FROM capture_session WHERE user_id = $1 AND result = 'pending'
				 ORDER BY created_at DESC LIMIT 1`, userID,
			).Scan(&resp.SessionId, &resp.BattleSessionId, &resp.UserId, &resp.PokemonId,
				&resp.BaseRate, &resp.CurrentRate, &resp.Result)
		}
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("session not found: %s", sessionID))
		}
	}
	// Use the real capture session ID for subsequent action queries
	sessionID = resp.SessionId

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

// HandleBattleFinished is called by the battle.finished Kafka consumer.
// Creates a capture session for each participant when result=win.
func (s *Service) HandleBattleFinished(ctx context.Context, battleSessionID, bossPokemonID string, participantUserIDs []string) error {
	for _, userID := range participantUserIDs {
		sessionID := uuid.NewString()

		agg := NewCaptureAggregate(sessionID)
		agg.Start(battleSessionID, userID, bossPokemonID, defaultBaseRate)

		if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, CaptureTopicMapper); err != nil {
			slog.Error("failed to save capture aggregate", "session_id", sessionID, "user_id", userID, "error", err)
			return fmt.Errorf("failed to save capture session for user %s: %w", userID, err)
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

	// Get item metadata from masterdata (before consuming item to avoid inventory loss on failure)
	var itemEffects []*masterdatapb.ItemEffect
	if s.masterdataClient != nil {
		itemResp, err := s.masterdataClient.GetItem(ctx, connect.NewRequest(&masterdatapb.GetItemRequest{Id: itemID}))
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get item metadata: %w", err))
		}
		itemEffects = itemResp.Msg.GetItem().GetEffects()
	}

	// Get boss pokemon type
	var bossType string
	if s.masterdataClient != nil {
		pokemonResp, err := s.masterdataClient.GetPokemon(ctx, connect.NewRequest(&masterdatapb.GetPokemonRequest{Id: agg.PokemonID}))
		if err != nil {
			slog.Warn("failed to get pokemon metadata", "pokemon_id", agg.PokemonID, "error", err)
		} else {
			bossType = pokemonResp.Msg.GetPokemon().GetType()
		}
	}

	// Consume item from inventory (after metadata lookup so no loss on metadata failure)
	if s.itemClient != nil {
		_, err := s.itemClient.UseItem(ctx, connect.NewRequest(&itempb.UseItemRequest{
			UserId: agg.UserID, ItemId: itemID, Quantity: 1,
		}))
		if err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("failed to use item: %w", err))
		}
	}

	rateBefore := agg.CurrentRate
	var escaped bool
	var flavorText string

	escaped, flavorText = applyItemEffect(agg, itemID, rateBefore, bossType, itemEffects)

	if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, CaptureTopicMapper); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save: %w", err))
	}

	// Update read model (best-effort; event store is source of truth)
	if s.db != nil {
		if _, err := s.db.Exec(ctx,
			`UPDATE capture_session SET current_rate = $1, result = $2 WHERE id = $3`,
			agg.CurrentRate, agg.Result, sessionID); err != nil {
			slog.Warn("failed to update capture_session read model", "session_id", sessionID, "error", err)
		}
		if _, err := s.db.Exec(ctx,
			`INSERT INTO capture_action (id, session_id, action_type, item_id, rate_change, created_at)
			 VALUES ($1, $2, 'use_item', $3, $4, $5)`,
			uuid.NewString(), sessionID, itemID, agg.CurrentRate-rateBefore, time.Now().UTC()); err != nil {
			slog.Warn("failed to insert capture_action read model", "session_id", sessionID, "error", err)
		}
	}

	return connect.NewResponse(&pb.UseItemResponse{
		RateBefore: rateBefore,
		RateAfter:  agg.CurrentRate,
		Escaped:    escaped,
		FlavorText: flavorText,
	}), nil
}

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

	var result string
	if rand.Float64() < agg.CurrentRate {
		result = "success"
	} else {
		result = "fail"
	}

	agg.ThrowBall(result)
	if result == "success" {
		agg.Complete("success")
	}

	if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, CaptureTopicMapper); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save: %w", err))
	}

	// Update read model (best-effort; event store is source of truth)
	if s.db != nil {
		if _, err := s.db.Exec(ctx,
			`UPDATE capture_session SET result = $1 WHERE id = $2`,
			agg.Result, sessionID); err != nil {
			slog.Warn("failed to update capture_session read model", "session_id", sessionID, "error", err)
		}
		if _, err := s.db.Exec(ctx,
			`INSERT INTO capture_action (id, session_id, action_type, rate_change, created_at)
			 VALUES ($1, $2, 'throw_ball', 0, $3)`,
			uuid.NewString(), sessionID, time.Now().UTC()); err != nil {
			slog.Warn("failed to insert capture_action read model", "session_id", sessionID, "error", err)
		}
	}

	return connect.NewResponse(&pb.ThrowBallResponse{
		Result: result,
	}), nil
}

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
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("session is still pending"))
	}

	// Emit capture.completed for fail case if not already emitted
	if agg.Result == "fail" && !agg.Completed {
		agg.Complete(agg.Result)
		if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, CaptureTopicMapper); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save: %w", err))
		}
	}

	return connect.NewResponse(&pb.EndSessionResponse{
		Result: agg.Result,
	}), nil
}

// findMatchedEffect returns the effect that matches bossType,
// preferring specific target matches over wildcard (empty target_type).
func findMatchedEffect(effects []*masterdatapb.ItemEffect, bossType string) *masterdatapb.ItemEffect {
	var wildcard *masterdatapb.ItemEffect
	for _, e := range effects {
		if e.GetTargetType() == bossType {
			return e
		}
		if e.GetTargetType() == "" && wildcard == nil {
			wildcard = e
		}
	}
	return wildcard
}

// applyItemEffect applies the matched item effect to the aggregate and returns escaped flag and flavor text.
func applyItemEffect(agg *CaptureAggregate, itemID string, rateBefore float64, bossType string, effects []*masterdatapb.ItemEffect) (escaped bool, flavorText string) {
	matched := findMatchedEffect(effects, bossType)
	if matched == nil {
		agg.UseItem(itemID, rateBefore, rateBefore)
		return false, ""
	}

	switch matched.GetEffectType() {
	case "escape":
		agg.UseItem(itemID, rateBefore, rateBefore)
		agg.Escape()
		return true, matched.GetFlavorText()
	default:
		rateAfter := min(max(rateBefore+matched.GetCaptureRateBonus(), 0.0), 1.0)
		agg.UseItem(itemID, rateBefore, rateAfter)
		return false, matched.GetFlavorText()
	}
}
