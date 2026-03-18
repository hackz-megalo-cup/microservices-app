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

	tx, txErr := s.outbox.BeginTx(ctx)
	if txErr != nil {
		return nil, connect.NewError(connect.CodeInternal, txErr)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, execErr := tx.Exec(ctx,
		`INSERT INTO pokemon (id, name, type, hp, attack, speed, special_move_name, special_move_damage)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		id, req.Msg.Name, req.Msg.Type, req.Msg.Hp, req.Msg.Attack, req.Msg.Speed,
		req.Msg.SpecialMoveName, req.Msg.SpecialMoveDamage,
	)
	if execErr != nil {
		return nil, connect.NewError(connect.CodeInternal, execErr)
	}

	event := platform.NewEvent(EventCreated, "masterdata-service", map[string]any{"stream_id": id.String()})
	if outboxErr := s.outbox.InsertEvent(ctx, tx, platform.TopicMasterdataCreated, event); outboxErr != nil {
		return nil, connect.NewError(connect.CodeInternal, outboxErr)
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return nil, connect.NewError(connect.CodeInternal, commitErr)
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
	tx, txErr := s.outbox.BeginTx(ctx)
	if txErr != nil {
		return nil, connect.NewError(connect.CodeInternal, txErr)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, execErr := tx.Exec(ctx,
		`INSERT INTO type_matchup (attacking_type, defending_type, effectiveness)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (attacking_type, defending_type) DO UPDATE SET effectiveness = EXCLUDED.effectiveness`,
		req.Msg.AttackingType, req.Msg.DefendingType, req.Msg.Effectiveness,
	)
	if execErr != nil {
		return nil, connect.NewError(connect.CodeInternal, execErr)
	}

	event := platform.NewEvent(EventCreated, "masterdata-service", map[string]any{
		"attacking_type": req.Msg.AttackingType,
		"defending_type": req.Msg.DefendingType,
	})
	if outboxErr := s.outbox.InsertEvent(ctx, tx, platform.TopicMasterdataCreated, event); outboxErr != nil {
		return nil, connect.NewError(connect.CodeInternal, outboxErr)
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return nil, connect.NewError(connect.CodeInternal, commitErr)
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

	tx, txErr := s.outbox.BeginTx(ctx)
	if txErr != nil {
		return nil, connect.NewError(connect.CodeInternal, txErr)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, execErr := tx.Exec(ctx,
		`INSERT INTO item_master (id, name) VALUES ($1, $2)`,
		id, req.Msg.Name,
	); execErr != nil {
		return nil, connect.NewError(connect.CodeInternal, execErr)
	}

	for _, e := range req.Msg.Effects {
		effectID, err := uuid.NewV7()
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if _, execErr := tx.Exec(ctx,
			`INSERT INTO item_effect (id, item_id, effect_type, target_type, capture_rate_bonus, flavor_text)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			effectID, id, e.EffectType, nullableString(e.TargetType), e.CaptureRateBonus, nullableString(e.FlavorText),
		); execErr != nil {
			return nil, connect.NewError(connect.CodeInternal, execErr)
		}
	}

	event := platform.NewEvent(EventCreated, "masterdata-service", map[string]any{"stream_id": id.String()})
	if outboxErr := s.outbox.InsertEvent(ctx, tx, platform.TopicMasterdataCreated, event); outboxErr != nil {
		return nil, connect.NewError(connect.CodeInternal, outboxErr)
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return nil, connect.NewError(connect.CodeInternal, commitErr)
	}

	return connect.NewResponse(&pb.CreateItemResponse{Id: id.String()}), nil
}

func (s *Service) GetItem(ctx context.Context, req *connect.Request[pb.GetItemRequest]) (*connect.Response[pb.GetItemResponse], error) {
	var item pb.Item
	err := s.pool.QueryRow(ctx,
		`SELECT id, name FROM item_master WHERE id = $1`,
		req.Msg.Id,
	).Scan(&item.Id, &item.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	effects, err := s.loadEffects(ctx, item.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	item.Effects = effects

	return connect.NewResponse(&pb.GetItemResponse{Item: &item}), nil
}

func (s *Service) ListItems(ctx context.Context, _ *connect.Request[pb.ListItemsRequest]) (*connect.Response[pb.ListItemsResponse], error) {
	rows, err := s.pool.Query(ctx,
		`SELECT m.id, m.name,
		        COALESCE(e.effect_type, ''), COALESCE(e.target_type, ''),
		        COALESCE(e.capture_rate_bonus, 0), COALESCE(e.flavor_text, '')
		 FROM item_master m
		 LEFT JOIN item_effect e ON e.item_id = m.id
		 ORDER BY m.created_at, e.priority, e.created_at`,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer rows.Close()

	itemMap := make(map[string]*pb.Item)
	var order []string
	for rows.Next() {
		var itemID, itemName, effectType, targetType, flavorText string
		var bonus float32
		if err := rows.Scan(&itemID, &itemName, &effectType, &targetType, &bonus, &flavorText); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if _, ok := itemMap[itemID]; !ok {
			itemMap[itemID] = &pb.Item{Id: itemID, Name: itemName}
			order = append(order, itemID)
		}
		if effectType != "" {
			itemMap[itemID].Effects = append(itemMap[itemID].Effects, &pb.ItemEffect{
				EffectType:       effectType,
				TargetType:       targetType,
				CaptureRateBonus: float64(bonus),
				FlavorText:       flavorText,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	items := make([]*pb.Item, 0, len(order))
	for _, id := range order {
		items = append(items, itemMap[id])
	}

	return connect.NewResponse(&pb.ListItemsResponse{Items: items}), nil
}

// loadEffects fetches all effects for a given item id.
func (s *Service) loadEffects(ctx context.Context, itemID string) ([]*pb.ItemEffect, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT effect_type, COALESCE(target_type, ''), capture_rate_bonus, COALESCE(flavor_text, '')
		 FROM item_effect WHERE item_id = $1 ORDER BY priority, created_at`,
		itemID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var effects []*pb.ItemEffect
	for rows.Next() {
		var e pb.ItemEffect
		var bonus float32
		if err := rows.Scan(&e.EffectType, &e.TargetType, &bonus, &e.FlavorText); err != nil {
			return nil, err
		}
		e.CaptureRateBonus = float64(bonus)
		effects = append(effects, &e)
	}
	return effects, rows.Err()
}

func (s *Service) UpdatePokemon(ctx context.Context, req *connect.Request[pb.UpdatePokemonRequest]) (*connect.Response[pb.UpdatePokemonResponse], error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE pokemon SET name=$2, type=$3, hp=$4, attack=$5, speed=$6, special_move_name=$7, special_move_damage=$8 WHERE id=$1`,
		req.Msg.Id, req.Msg.Name, req.Msg.Type, req.Msg.Hp, req.Msg.Attack, req.Msg.Speed,
		req.Msg.SpecialMoveName, req.Msg.SpecialMoveDamage,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if tag.RowsAffected() == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("pokemon not found"))
	}
	p := &pb.Pokemon{
		Id: req.Msg.Id, Name: req.Msg.Name, Type: req.Msg.Type,
		Hp: req.Msg.Hp, Attack: req.Msg.Attack, Speed: req.Msg.Speed,
		SpecialMoveName: req.Msg.SpecialMoveName, SpecialMoveDamage: req.Msg.SpecialMoveDamage,
	}
	return connect.NewResponse(&pb.UpdatePokemonResponse{Pokemon: p}), nil
}

func (s *Service) DeletePokemon(ctx context.Context, req *connect.Request[pb.DeletePokemonRequest]) (*connect.Response[pb.DeletePokemonResponse], error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM pokemon WHERE id=$1`, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if tag.RowsAffected() == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("pokemon not found"))
	}
	return connect.NewResponse(&pb.DeletePokemonResponse{}), nil
}

func (s *Service) UpdateTypeMatchup(ctx context.Context, req *connect.Request[pb.UpdateTypeMatchupRequest]) (*connect.Response[pb.UpdateTypeMatchupResponse], error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE type_matchup SET effectiveness=$3 WHERE attacking_type=$1 AND defending_type=$2`,
		req.Msg.AttackingType, req.Msg.DefendingType, req.Msg.Effectiveness,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if tag.RowsAffected() == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("type matchup not found"))
	}
	m := &pb.TypeMatchup{
		AttackingType: req.Msg.AttackingType,
		DefendingType: req.Msg.DefendingType,
		Effectiveness: req.Msg.Effectiveness,
	}
	return connect.NewResponse(&pb.UpdateTypeMatchupResponse{Matchup: m}), nil
}

func (s *Service) DeleteTypeMatchup(ctx context.Context, req *connect.Request[pb.DeleteTypeMatchupRequest]) (*connect.Response[pb.DeleteTypeMatchupResponse], error) {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM type_matchup WHERE attacking_type=$1 AND defending_type=$2`,
		req.Msg.AttackingType, req.Msg.DefendingType,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if tag.RowsAffected() == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("type matchup not found"))
	}
	return connect.NewResponse(&pb.DeleteTypeMatchupResponse{}), nil
}

func (s *Service) UpdateItem(ctx context.Context, req *connect.Request[pb.UpdateItemRequest]) (*connect.Response[pb.UpdateItemResponse], error) {
	tx, txErr := s.outbox.BeginTx(ctx)
	if txErr != nil {
		return nil, connect.NewError(connect.CodeInternal, txErr)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `UPDATE item_master SET name=$2 WHERE id=$1`, req.Msg.Id, req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if tag.RowsAffected() == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("item not found"))
	}

	if _, err := tx.Exec(ctx, `DELETE FROM item_effect WHERE item_id=$1`, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	for _, e := range req.Msg.Effects {
		effectID, err := uuid.NewV7()
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if _, execErr := tx.Exec(ctx,
			`INSERT INTO item_effect (id, item_id, effect_type, target_type, capture_rate_bonus, flavor_text)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			effectID, req.Msg.Id, e.EffectType, nullableString(e.TargetType), e.CaptureRateBonus, nullableString(e.FlavorText),
		); execErr != nil {
			return nil, connect.NewError(connect.CodeInternal, execErr)
		}
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return nil, connect.NewError(connect.CodeInternal, commitErr)
	}

	effects, err := s.loadEffects(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&pb.UpdateItemResponse{Item: &pb.Item{Id: req.Msg.Id, Name: req.Msg.Name, Effects: effects}}), nil
}

func (s *Service) DeleteItem(ctx context.Context, req *connect.Request[pb.DeleteItemRequest]) (*connect.Response[pb.DeleteItemResponse], error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM item_master WHERE id=$1`, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if tag.RowsAffected() == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("item not found"))
	}
	return connect.NewResponse(&pb.DeleteItemResponse{}), nil
}

// nullableString converts empty string to nil for nullable DB columns.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
