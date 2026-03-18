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

// HandleBattleFinished is called by the battle.finished Kafka consumer.
// Creates a capture session for each participant when result=win.
func (s *Service) HandleBattleFinished(ctx context.Context, battleSessionID, bossPokemonID string, participantUserIDs []string) error {
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
