package lobby

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	pb "github.com/hackz-megalo-cup/microservices-app/services/gen/go/lobby/v1"
	masterdatav1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/masterdata/v1"
	"github.com/hackz-megalo-cup/microservices-app/services/gen/go/masterdata/v1/masterdatav1connect"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

type Service struct {
	eventStore       *platform.EventStore
	outbox           *platform.OutboxStore
	lobbyDB          *pgxpool.Pool
	authDB           *pgxpool.Pool
	itemDB           *pgxpool.Pool
	raidLobbyDB      *pgxpool.Pool
	masterdataClient masterdatav1connect.MasterdataServiceClient
}

func NewService(
	eventStore *platform.EventStore,
	outbox *platform.OutboxStore,
	lobbyDB *pgxpool.Pool,
	authDB *pgxpool.Pool,
	itemDB *pgxpool.Pool,
	raidLobbyDB *pgxpool.Pool,
	masterdataClient masterdatav1connect.MasterdataServiceClient,
) *Service {
	return &Service{
		eventStore:       eventStore,
		outbox:           outbox,
		lobbyDB:          lobbyDB,
		authDB:           authDB,
		itemDB:           itemDB,
		raidLobbyDB:      raidLobbyDB,
		masterdataClient: masterdataClient,
	}
}

func (s *Service) SetActivePokemon(ctx context.Context, req *connect.Request[pb.SetActivePokemonRequest]) (*connect.Response[pb.SetActivePokemonResponse], error) {
	userID := req.Msg.GetUserId()
	pokemonID := req.Msg.GetPokemonId()
	if userID == "" || pokemonID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user_id and pokemon_id are required"))
	}

	// 所有チェック: auth_db.user_pokemon を参照
	if s.authDB != nil {
		var exists bool
		if err := s.authDB.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM user_pokemon WHERE user_id = $1 AND pokemon_id = $2)`,
			userID, pokemonID,
		).Scan(&exists); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("ownership check failed: %w", err))
		}
		if !exists {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("pokemon not owned by user"))
		}
	}

	// UPSERT: user_active_pokemon
	if s.lobbyDB != nil {
		if _, err := s.lobbyDB.Exec(ctx, `
			INSERT INTO user_active_pokemon (user_id, pokemon_id, updated_at)
			VALUES ($1, $2, now())
			ON CONFLICT (user_id) DO UPDATE SET pokemon_id = $2, updated_at = now()
		`, userID, pokemonID); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to set active pokemon: %w", err))
		}
	}

	return connect.NewResponse(&pb.SetActivePokemonResponse{Success: true}), nil
}

func (s *Service) GetActivePokemon(ctx context.Context, req *connect.Request[pb.GetActivePokemonRequest]) (*connect.Response[pb.GetActivePokemonResponse], error) {
	userID := req.Msg.GetUserId()
	if userID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user_id is required"))
	}

	var pokemonID string
	if s.lobbyDB != nil {
		err := s.lobbyDB.QueryRow(ctx,
			`SELECT pokemon_id FROM user_active_pokemon WHERE user_id = $1`,
			userID,
		).Scan(&pokemonID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no active pokemon set"))
			}
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get active pokemon: %w", err))
		}
	}

	return connect.NewResponse(&pb.GetActivePokemonResponse{PokemonId: pokemonID}), nil
}

func (s *Service) GetLobbyOverview(ctx context.Context, req *connect.Request[pb.GetLobbyOverviewRequest]) (*connect.Response[pb.GetLobbyOverviewResponse], error) {
	userID := req.Msg.GetUserId()
	if userID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user_id is required"))
	}

	type itemsResult struct {
		rows []userItemRow
		err  error
	}
	type raidsResult struct {
		rows []openRaidRow
		err  error
	}
	type pokemonResult struct {
		list []*masterdatav1.Pokemon
		err  error
	}

	itemsCh := make(chan itemsResult, 1)
	raidsCh := make(chan raidsResult, 1)
	pokemonCh := make(chan pokemonResult, 1)

	go func() {
		rows, err := s.fetchUserItems(ctx, userID)
		itemsCh <- itemsResult{rows, err}
	}()
	go func() {
		rows, err := s.fetchOpenRaids(ctx)
		raidsCh <- raidsResult{rows, err}
	}()
	go func() {
		list, err := s.fetchAllPokemon(ctx)
		pokemonCh <- pokemonResult{list, err}
	}()

	ir := <-itemsCh
	rr := <-raidsCh
	pr := <-pokemonCh

	if ir.err != nil {
		slog.WarnContext(ctx, "failed to fetch user items", "error", ir.err)
	}
	if rr.err != nil {
		slog.WarnContext(ctx, "failed to fetch open raids", "error", rr.err)
	}
	if pr.err != nil {
		slog.WarnContext(ctx, "failed to fetch pokemon", "error", pr.err)
	}

	pokemonMap := buildPokemonMap(pr.list)
	ownedItems := s.buildOwnedItems(ctx, ir.rows)
	raids := buildRaids(rr.rows, pokemonMap)
	pokedex := buildPokedex(pr.list)

	return connect.NewResponse(&pb.GetLobbyOverviewResponse{
		Items:   ownedItems,
		Raids:   raids,
		Pokedex: pokedex,
	}), nil
}

// userItemRow holds a user's item and its computed quantity from the event store.
type userItemRow struct {
	itemID   string
	quantity int32
}

// openRaidRow holds a raid lobby entry in waiting status.
type openRaidRow struct {
	id            string
	bossPokemonID string
}

// fetchUserItems queries item_db's event_store to compute per-item quantities for the user.
func (s *Service) fetchUserItems(ctx context.Context, userID string) ([]userItemRow, error) {
	if s.itemDB == nil {
		return nil, nil
	}
	rows, err := s.itemDB.Query(ctx, `
		SELECT
			SPLIT_PART(stream_id, ':', 2) AS item_id,
			SUM(
				CASE
					WHEN event_type = 'item.granted' THEN (data->>'quantity')::int
					WHEN event_type = 'item.used'    THEN -(data->>'quantity')::int
					ELSE 0
				END
			) AS quantity
		FROM event_store
		WHERE stream_type = 'item'
		  AND stream_id LIKE $1
		GROUP BY SPLIT_PART(stream_id, ':', 2)
		HAVING SUM(
			CASE
				WHEN event_type = 'item.granted' THEN (data->>'quantity')::int
				WHEN event_type = 'item.used'    THEN -(data->>'quantity')::int
				ELSE 0
			END
		) > 0
	`, userID+":%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []userItemRow
	for rows.Next() {
		var r userItemRow
		if err := rows.Scan(&r.itemID, &r.quantity); err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

// fetchOpenRaids queries raid_lobby_db for raids with status 'waiting'.
func (s *Service) fetchOpenRaids(ctx context.Context) ([]openRaidRow, error) {
	if s.raidLobbyDB == nil {
		return nil, nil
	}
	rows, err := s.raidLobbyDB.Query(ctx,
		`SELECT id, boss_pokemon_id FROM raid_lobby WHERE status = 'waiting'`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var raids []openRaidRow
	for rows.Next() {
		var r openRaidRow
		if err := rows.Scan(&r.id, &r.bossPokemonID); err != nil {
			return nil, err
		}
		raids = append(raids, r)
	}
	return raids, rows.Err()
}

// fetchAllPokemon calls masterdata ListPokemon for the full pokedex.
func (s *Service) fetchAllPokemon(ctx context.Context) ([]*masterdatav1.Pokemon, error) {
	if s.masterdataClient == nil {
		return nil, nil
	}
	resp, err := s.masterdataClient.ListPokemon(ctx, connect.NewRequest(&masterdatav1.ListPokemonRequest{}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetPokemon(), nil
}

// buildOwnedItems enriches userItemRows with names from masterdata.
func (s *Service) buildOwnedItems(ctx context.Context, userItems []userItemRow) []*pb.OwnedItem {
	if len(userItems) == 0 {
		return nil
	}

	itemNameMap := s.fetchItemNames(ctx)

	owned := make([]*pb.OwnedItem, 0, len(userItems))
	for _, ui := range userItems {
		owned = append(owned, &pb.OwnedItem{
			ItemId:   ui.itemID,
			Name:     itemNameMap[ui.itemID],
			Quantity: ui.quantity,
		})
	}
	return owned
}

// fetchItemNames calls masterdata ListItems and returns a map of id→name.
func (s *Service) fetchItemNames(ctx context.Context) map[string]string {
	m := make(map[string]string)
	if s.masterdataClient == nil {
		return m
	}
	resp, err := s.masterdataClient.ListItems(ctx, connect.NewRequest(&masterdatav1.ListItemsRequest{}))
	if err != nil {
		slog.WarnContext(ctx, "failed to list items from masterdata", "error", err)
		return m
	}
	for _, item := range resp.Msg.GetItems() {
		m[item.GetId()] = item.GetName()
	}
	return m
}

func buildPokemonMap(pokemon []*masterdatav1.Pokemon) map[string]string {
	m := make(map[string]string, len(pokemon))
	for _, p := range pokemon {
		m[p.GetId()] = p.GetName()
	}
	return m
}

func buildRaids(rows []openRaidRow, pokemonMap map[string]string) []*pb.Raid {
	if len(rows) == 0 {
		return nil
	}
	raids := make([]*pb.Raid, 0, len(rows))
	for _, r := range rows {
		raids = append(raids, &pb.Raid{
			Id:          r.id,
			PokemonId:   r.bossPokemonID,
			PokemonName: pokemonMap[r.bossPokemonID],
			Status:      pb.RaidStatus_RAID_STATUS_OPEN,
		})
	}
	return raids
}

func buildPokedex(pokemon []*masterdatav1.Pokemon) []*pb.PokedexEntry {
	if len(pokemon) == 0 {
		return nil
	}
	entries := make([]*pb.PokedexEntry, 0, len(pokemon))
	for _, p := range pokemon {
		entries = append(entries, &pb.PokedexEntry{
			PokemonId:   p.GetId(),
			PokemonName: p.GetName(),
			Caught:      false,
		})
	}
	return entries
}
