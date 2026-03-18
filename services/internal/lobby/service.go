package lobby

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	itemv1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/item/v1"
	"github.com/hackz-megalo-cup/microservices-app/services/gen/go/item/v1/itemv1connect"
	pb "github.com/hackz-megalo-cup/microservices-app/services/gen/go/lobby/v1"
	masterdatav1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/masterdata/v1"
	"github.com/hackz-megalo-cup/microservices-app/services/gen/go/masterdata/v1/masterdatav1connect"
	raid_lobbyv1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/raid_lobby/v1"
	"github.com/hackz-megalo-cup/microservices-app/services/gen/go/raid_lobby/v1/raid_lobbyv1connect"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

type Service struct {
	eventStore       *platform.EventStore
	outbox           *platform.OutboxStore
	lobbyDB          *pgxpool.Pool
	authDB           *pgxpool.Pool
	itemClient       itemv1connect.ItemServiceClient
	raidLobbyClient  raid_lobbyv1connect.RaidLobbyServiceClient
	masterdataClient masterdatav1connect.MasterdataServiceClient
}

func NewService(
	eventStore *platform.EventStore,
	outbox *platform.OutboxStore,
	lobbyDB *pgxpool.Pool,
	authDB *pgxpool.Pool,
	itemClient itemv1connect.ItemServiceClient,
	raidLobbyClient raid_lobbyv1connect.RaidLobbyServiceClient,
	masterdataClient masterdatav1connect.MasterdataServiceClient,
) *Service {
	return &Service{
		eventStore:       eventStore,
		outbox:           outbox,
		lobbyDB:          lobbyDB,
		authDB:           authDB,
		itemClient:       itemClient,
		raidLobbyClient:  raidLobbyClient,
		masterdataClient: masterdataClient,
	}
}

func (s *Service) SetActivePokemon(ctx context.Context, req *connect.Request[pb.SetActivePokemonRequest]) (*connect.Response[pb.SetActivePokemonResponse], error) {
	userID := req.Msg.GetUserId()
	pokemonID := req.Msg.GetPokemonId()
	if userID == "" || pokemonID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user_id and pokemon_id are required"))
	}

	// NOTE: If authDB is nil (AUTH_DATABASE_URL not set or connection failed at startup),
	// ownership check is skipped — any user can assign any pokemon_id.
	// Ensure AUTH_DATABASE_URL is always configured in production.
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

	// Persist via event sourcing.
	agg := NewAggregate(userID)
	_ = platform.LoadAggregate(ctx, s.eventStore, agg) // ignore "not found" for new users
	agg.SetActivePokemon(userID, pokemonID)
	if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, LobbyTopicMapper); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save aggregate: %w", err))
	}

	// UPSERT read model for fast lookups by GetActivePokemon.
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

	// NOTE: If lobbyDB is nil (LOBBY_DATABASE_URL not set or connection failed at startup),
	// this RPC returns an internal error rather than silently returning an empty pokemon_id.
	// Ensure LOBBY_DATABASE_URL is always configured in production.
	if s.lobbyDB == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("lobby database unavailable"))
	}

	var pokemonID string
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
	type caughtResult struct {
		ids map[string]bool
		err error
	}

	itemsCh := make(chan itemsResult, 1)
	raidsCh := make(chan raidsResult, 1)
	pokemonCh := make(chan pokemonResult, 1)
	caughtCh := make(chan caughtResult, 1)
	itemNamesCh := make(chan map[string]string, 1)

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
	go func() {
		ids, err := s.fetchCaughtPokemon(ctx, userID)
		caughtCh <- caughtResult{ids, err}
	}()
	go func() {
		itemNamesCh <- s.fetchItemNames(ctx)
	}()

	ir := <-itemsCh
	rr := <-raidsCh
	pr := <-pokemonCh
	cr := <-caughtCh
	itemNamesMap := <-itemNamesCh

	if ir.err != nil {
		slog.WarnContext(ctx, "failed to fetch user items", "error", ir.err)
	}
	if rr.err != nil {
		slog.WarnContext(ctx, "failed to fetch open raids", "error", rr.err)
	}
	if pr.err != nil {
		slog.WarnContext(ctx, "failed to fetch pokemon", "error", pr.err)
	}
	if cr.err != nil {
		slog.WarnContext(ctx, "failed to fetch caught pokemon", "error", cr.err)
	}

	pokemonMap := buildPokemonMap(pr.list)
	ownedItems := buildOwnedItems(ir.rows, itemNamesMap)
	raids := buildRaids(rr.rows, pokemonMap)
	pokedex := buildPokedex(pr.list, cr.ids)

	return connect.NewResponse(&pb.GetLobbyOverviewResponse{
		Items:   ownedItems,
		Raids:   raids,
		Pokedex: pokedex,
	}), nil
}

// userItemRow holds a user's item and its computed quantity.
type userItemRow struct {
	itemID   string
	quantity int32
}

// openRaidRow holds a raid lobby entry in waiting status.
type openRaidRow struct {
	id            string
	bossPokemonID string
}

// fetchUserItems calls item-service GetUserItems RPC.
func (s *Service) fetchUserItems(ctx context.Context, userID string) ([]userItemRow, error) {
	if s.itemClient == nil {
		return nil, nil
	}
	resp, err := s.itemClient.GetUserItems(ctx, connect.NewRequest(&itemv1.GetUserItemsRequest{UserId: userID}))
	if err != nil {
		return nil, err
	}
	rows := make([]userItemRow, 0, len(resp.Msg.GetItems()))
	for _, it := range resp.Msg.GetItems() {
		rows = append(rows, userItemRow{itemID: it.GetItemId(), quantity: it.GetQuantity()})
	}
	return rows, nil
}

// fetchOpenRaids calls raid-lobby-service ListOpenRaids RPC.
func (s *Service) fetchOpenRaids(ctx context.Context) ([]openRaidRow, error) {
	if s.raidLobbyClient == nil {
		return nil, nil
	}
	resp, err := s.raidLobbyClient.ListOpenRaids(ctx, connect.NewRequest(&raid_lobbyv1.ListOpenRaidsRequest{}))
	if err != nil {
		return nil, err
	}
	rows := make([]openRaidRow, 0, len(resp.Msg.GetRaids()))
	for _, r := range resp.Msg.GetRaids() {
		rows = append(rows, openRaidRow{id: r.GetId(), bossPokemonID: r.GetBossPokemonId()})
	}
	return rows, nil
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

// fetchCaughtPokemon queries auth DB for all pokemon the user has caught.
// NOTE: If authDB is nil (AUTH_DATABASE_URL not set or connection failed at startup),
// this returns (nil, nil) — buildPokedex will treat every entry as caught=false,
// so the entire pokedex will appear uncaught. This matches the behaviour of
// SetActivePokemon skipping ownership checks when authDB is nil.
func (s *Service) fetchCaughtPokemon(ctx context.Context, userID string) (map[string]bool, error) {
	if s.authDB == nil {
		return nil, nil
	}
	rows, err := s.authDB.Query(ctx, `SELECT pokemon_id FROM user_pokemon WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make(map[string]bool)
	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			return nil, err
		}
		ids[pid] = true
	}
	return ids, rows.Err()
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

func buildOwnedItems(userItems []userItemRow, itemNameMap map[string]string) []*pb.OwnedItem {
	if len(userItems) == 0 {
		return nil
	}
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

func buildPokedex(pokemon []*masterdatav1.Pokemon, caught map[string]bool) []*pb.PokedexEntry {
	if len(pokemon) == 0 {
		return nil
	}
	entries := make([]*pb.PokedexEntry, 0, len(pokemon))
	for _, p := range pokemon {
		entries = append(entries, &pb.PokedexEntry{
			PokemonId:   p.GetId(),
			PokemonName: p.GetName(),
			Caught:      caught[p.GetId()],
		})
	}
	return entries
}
