package masterdata

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	pb "github.com/hackz-megalo-cup/microservices-app/services/gen/go/masterdata/v1"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

type Service struct {
	eventStore *platform.EventStore
	outbox     *platform.OutboxStore
	pool       *pgxpool.Pool
}

func NewService(eventStore *platform.EventStore, outbox *platform.OutboxStore, pool *pgxpool.Pool) *Service {
	return &Service{
		eventStore: eventStore,
		outbox:     outbox,
		pool:       pool,
	}
}

func (s *Service) CreatePokemon(ctx context.Context, req *connect.Request[pb.CreatePokemonRequest]) (*connect.Response[pb.CreatePokemonResponse], error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	_, err = s.pool.Exec(ctx,
		`INSERT INTO pokemon (id, name, type, hp, attack, speed, special_move_name, special_move_damage)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		id, req.Msg.Name, req.Msg.Type, req.Msg.Hp, req.Msg.Attack, req.Msg.Speed,
		req.Msg.SpecialMoveName, req.Msg.SpecialMoveDamage,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	agg := NewAggregate(id.String())
	agg.Create()
	if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, TopicMapper); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.CreatePokemonResponse{Id: id.String()}), nil
}

func (s *Service) GetPokemon(ctx context.Context, req *connect.Request[pb.GetPokemonRequest]) (*connect.Response[pb.GetPokemonResponse], error) {
	var p pb.Pokemon
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, type, hp, attack, speed, special_move_name, special_move_damage FROM pokemon WHERE id = $1`,
		req.Msg.Id,
	).Scan(&p.Id, &p.Name, &p.Type, &p.Hp, &p.Attack, &p.Speed, &p.SpecialMoveName, &p.SpecialMoveDamage)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.GetPokemonResponse{Pokemon: &p}), nil
}

func (s *Service) ListPokemon(ctx context.Context, _ *connect.Request[pb.ListPokemonRequest]) (*connect.Response[pb.ListPokemonResponse], error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, type, hp, attack, speed, special_move_name, special_move_damage FROM pokemon ORDER BY created_at`,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer rows.Close()

	var pokemons []*pb.Pokemon
	for rows.Next() {
		var p pb.Pokemon
		if err := rows.Scan(&p.Id, &p.Name, &p.Type, &p.Hp, &p.Attack, &p.Speed, &p.SpecialMoveName, &p.SpecialMoveDamage); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		pokemons = append(pokemons, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.ListPokemonResponse{Pokemon: pokemons}), nil
}

func (s *Service) CreateTypeMatchup(ctx context.Context, req *connect.Request[pb.CreateTypeMatchupRequest]) (*connect.Response[pb.CreateTypeMatchupResponse], error) {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO type_matchup (attacking_type, defending_type, effectiveness)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (attacking_type, defending_type) DO UPDATE SET effectiveness = EXCLUDED.effectiveness`,
		req.Msg.AttackingType, req.Msg.DefendingType, req.Msg.Effectiveness,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	streamID, err := uuid.NewV7()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	agg := NewAggregate(streamID.String())
	agg.Create()
	if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, TopicMapper); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.CreateTypeMatchupResponse{}), nil
}

func (s *Service) ListTypeMatchups(ctx context.Context, _ *connect.Request[pb.ListTypeMatchupsRequest]) (*connect.Response[pb.ListTypeMatchupsResponse], error) {
	rows, err := s.pool.Query(ctx,
		`SELECT attacking_type, defending_type, effectiveness FROM type_matchup ORDER BY attacking_type, defending_type`,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer rows.Close()

	var matchups []*pb.TypeMatchup
	for rows.Next() {
		var m pb.TypeMatchup
		var eff float32
		if err := rows.Scan(&m.AttackingType, &m.DefendingType, &eff); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		m.Effectiveness = float64(eff)
		matchups = append(matchups, &m)
	}
	if err := rows.Err(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.ListTypeMatchupsResponse{Matchups: matchups}), nil
}

func (s *Service) CreateItem(ctx context.Context, req *connect.Request[pb.CreateItemRequest]) (*connect.Response[pb.CreateItemResponse], error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	_, err = s.pool.Exec(ctx,
		`INSERT INTO item_master (id, name, effect_type, target_type, capture_rate_bonus)
		 VALUES ($1, $2, $3, $4, $5)`,
		id, req.Msg.Name, req.Msg.EffectType, nullableString(req.Msg.TargetType), req.Msg.CaptureRateBonus,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	agg := NewAggregate(id.String())
	agg.Create()
	if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, TopicMapper); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.CreateItemResponse{Id: id.String()}), nil
}

func (s *Service) GetItem(ctx context.Context, req *connect.Request[pb.GetItemRequest]) (*connect.Response[pb.GetItemResponse], error) {
	var item pb.Item
	var targetType *string
	var captureRateBonus float32
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, effect_type, target_type, capture_rate_bonus FROM item_master WHERE id = $1`,
		req.Msg.Id,
	).Scan(&item.Id, &item.Name, &item.EffectType, &targetType, &captureRateBonus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if targetType != nil {
		item.TargetType = *targetType
	}
	item.CaptureRateBonus = float64(captureRateBonus)

	return connect.NewResponse(&pb.GetItemResponse{Item: &item}), nil
}

func (s *Service) ListItems(ctx context.Context, _ *connect.Request[pb.ListItemsRequest]) (*connect.Response[pb.ListItemsResponse], error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, effect_type, target_type, capture_rate_bonus FROM item_master ORDER BY created_at`,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer rows.Close()

	var items []*pb.Item
	for rows.Next() {
		var item pb.Item
		var targetType *string
		var captureRateBonus float32
		if err := rows.Scan(&item.Id, &item.Name, &item.EffectType, &targetType, &captureRateBonus); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if targetType != nil {
			item.TargetType = *targetType
		}
		item.CaptureRateBonus = float64(captureRateBonus)
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.ListItemsResponse{Items: items}), nil
}

// nullableString converts empty string to nil for nullable DB columns.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
