package gateway

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	agonesclient "agones.dev/agones/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// AllocationStore holds lobbyId → server info mappings so that late-joiners
// can look up an already-allocated game server.
type AllocationStore struct {
	mu      sync.RWMutex
	entries map[string]AllocateResponse
}

func NewAllocationStore() *AllocationStore {
	return &AllocationStore{entries: make(map[string]AllocateResponse)}
}

func (s *AllocationStore) Put(lobbyID string, resp AllocateResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[lobbyID] = resp
}

func (s *AllocationStore) Get(lobbyID string) (AllocateResponse, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	resp, ok := s.entries[lobbyID]
	return resp, ok
}

// ActiveEntry is a lobby-to-server mapping returned by First / List.
type ActiveEntry struct {
	LobbyID  string `json:"lobbyId"`
	Host     string `json:"host"`
	Port     int32  `json:"port"`
	CertHash string `json:"certHash"`
}

// First returns an arbitrary active allocation, or false if none exist.
func (s *AllocationStore) First() (ActiveEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for lobbyID, resp := range s.entries {
		return ActiveEntry{LobbyID: lobbyID, Host: resp.Host, Port: resp.Port, CertHash: resp.CertHash}, true
	}
	return ActiveEntry{}, false
}

func (s *AllocationStore) Delete(lobbyID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, lobbyID)
}

type AllocateHandler struct {
	agonesClient agonesclient.Interface
	namespace    string
	httpClient   *http.Client
	store        *AllocationStore
}

type AllocateRequest struct {
	LobbyID       string `json:"lobbyId"`
	BossPokemonID string `json:"bossPokemonId"`
}

type AllocateResponse struct {
	Host     string `json:"host"`
	Port     int32  `json:"port"`
	CertHash string `json:"certHash"`
}

func NewAllocateHandler(namespace string, store *AllocationStore) (*AllocateHandler, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	agonesClient, err := agonesclient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create agones client: %w", err)
	}

	return &AllocateHandler{
		agonesClient: agonesClient,
		namespace:    namespace,
		store:        store,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}, nil
}

func (h *AllocateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AllocateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	allocation := &allocationv1.GameServerAllocation{
		ObjectMeta: metav1.ObjectMeta{Namespace: h.namespace},
		Spec: allocationv1.GameServerAllocationSpec{
			Selectors: []allocationv1.GameServerSelector{{
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"game": "raid-battle"},
				},
			}},
			MetaPatch: allocationv1.MetaPatch{
				Annotations: map[string]string{
					"raid.lobby-id":        req.LobbyID,
					"raid.boss-pokemon-id": req.BossPokemonID,
				},
			},
		},
	}

	result, err := h.agonesClient.AllocationV1().GameServerAllocations(h.namespace).Create(
		r.Context(), allocation, metav1.CreateOptions{},
	)
	if err != nil {
		slog.Error("failed to allocate game server", "error", err)
		http.Error(w, "failed to allocate game server", http.StatusServiceUnavailable)
		return
	}

	if result.Status.State != allocationv1.GameServerAllocationAllocated {
		slog.Error("allocation not successful", "state", result.Status.State)
		http.Error(w, "no available game servers", http.StatusServiceUnavailable)
		return
	}

	if len(result.Status.Ports) == 0 {
		slog.Error("allocated game server has no ports", "address", result.Status.Address)
		http.Error(w, "allocated game server has no ports", http.StatusInternalServerError)
		return
	}

	address := result.Status.Address
	port := result.Status.Ports[0].Port

	certHash, err := h.resolveCertHash(r.Context(), result)
	if err != nil {
		slog.Error("failed to fetch cert hash", "error", err, "address", address, "port", port)
		http.Error(w, "failed to fetch cert hash from game server", http.StatusInternalServerError)
		return
	}

	resp := AllocateResponse{
		Host:     address,
		Port:     port,
		CertHash: certHash,
	}

	if req.LobbyID != "" {
		h.store.Put(req.LobbyID, resp)
		slog.Info("stored allocation for lobby", "lobbyId", req.LobbyID, "host", address, "port", port)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func (h *AllocateHandler) resolveCertHash(ctx context.Context, result *allocationv1.GameServerAllocation) (string, error) {
	if metadata := result.Status.Metadata; metadata != nil {
		if certHash := firstCertHash(metadata.Annotations); certHash != "" {
			return certHash, nil
		}
	}

	if name := result.Status.GameServerName; name != "" {
		gs, err := h.agonesClient.AgonesV1().GameServers(h.namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("get game server %q: %w", name, err)
		}
		if certHash := firstCertHash(gs.Annotations); certHash != "" {
			return certHash, nil
		}
	}

	address := result.Status.Address
	if len(result.Status.Ports) == 0 {
		return "", fmt.Errorf("allocated game server has no ports")
	}
	port := result.Status.Ports[0].Port
	certHash, err := h.fetchCertHash(ctx, address, port)
	if err != nil {
		return "", err
	}
	if certHash = strings.TrimSpace(certHash); certHash == "" {
		return "", fmt.Errorf("cert hash is empty")
	}
	return certHash, nil
}

func firstCertHash(annotations map[string]string) string {
	for _, key := range []string{"cert-hash", "agones.dev/sdk-cert-hash"} {
		if certHash := strings.TrimSpace(annotations[key]); certHash != "" {
			return certHash
		}
	}
	return ""
}

func (h *AllocateHandler) fetchCertHash(ctx context.Context, address string, port int32) (string, error) {
	url := fmt.Sprintf("https://%s:%d/cert-hash", address, port)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("cert-hash request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cert-hash returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

type raidLocator struct {
	agonesClient agonesclient.Interface
	namespace    string
	httpClient   *http.Client
}

func newRaidLocator(namespace string) (*raidLocator, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	agonesClient, err := agonesclient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create agones client: %w", err)
	}

	return &raidLocator{
		agonesClient: agonesClient,
		namespace:    namespace,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}, nil
}

func (l *raidLocator) activeEntryFromGameServer(ctx context.Context, gs agonesv1.GameServer) (ActiveEntry, error) {
	if len(gs.Status.Ports) == 0 {
		return ActiveEntry{}, fmt.Errorf("game server %q has no ports", gs.Name)
	}

	certHash := firstCertHash(gs.Annotations)
	if certHash == "" {
		hash, err := l.fetchCertHash(ctx, gs.Status.Address, gs.Status.Ports[0].Port)
		if err != nil {
			return ActiveEntry{}, err
		}
		certHash = strings.TrimSpace(hash)
	}
	if certHash == "" {
		return ActiveEntry{}, fmt.Errorf("game server %q has empty cert hash", gs.Name)
	}

	return ActiveEntry{
		LobbyID:  strings.TrimSpace(gs.Annotations["raid.lobby-id"]),
		Host:     gs.Status.Address,
		Port:     gs.Status.Ports[0].Port,
		CertHash: certHash,
	}, nil
}

func (l *raidLocator) fetchCertHash(ctx context.Context, address string, port int32) (string, error) {
	url := fmt.Sprintf("https://%s:%d/cert-hash", address, port)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("cert-hash request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cert-hash returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (l *raidLocator) listAllocatedRaidServers(ctx context.Context) ([]agonesv1.GameServer, error) {
	list, err := l.agonesClient.AgonesV1().GameServers(l.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "game=raid-battle",
	})
	if err != nil {
		return nil, err
	}

	servers := make([]agonesv1.GameServer, 0, len(list.Items))
	for _, gs := range list.Items {
		if gs.Status.State != agonesv1.GameServerStateAllocated {
			continue
		}
		if strings.TrimSpace(gs.Annotations["raid.lobby-id"]) == "" {
			continue
		}
		servers = append(servers, gs)
	}

	sort.Slice(servers, func(i, j int) bool {
		left := servers[i].CreationTimestamp.Time
		right := servers[j].CreationTimestamp.Time
		if left.Equal(right) {
			return servers[i].Name < servers[j].Name
		}
		return left.Before(right)
	})

	return servers, nil
}

func (l *raidLocator) findFirstActive(ctx context.Context) (ActiveEntry, bool, error) {
	servers, err := l.listAllocatedRaidServers(ctx)
	if err != nil {
		return ActiveEntry{}, false, err
	}
	for _, gs := range servers {
		entry, err := l.activeEntryFromGameServer(ctx, gs)
		if err != nil {
			slog.Warn("failed to resolve active raid entry", "gameServer", gs.Name, "error", err)
			continue
		}
		return entry, true, nil
	}
	return ActiveEntry{}, false, nil
}

func (l *raidLocator) findByLobbyID(ctx context.Context, lobbyID string) (AllocateResponse, bool, error) {
	servers, err := l.listAllocatedRaidServers(ctx)
	if err != nil {
		return AllocateResponse{}, false, err
	}
	for _, gs := range servers {
		if strings.TrimSpace(gs.Annotations["raid.lobby-id"]) != lobbyID {
			continue
		}
		entry, err := l.activeEntryFromGameServer(ctx, gs)
		if err != nil {
			return AllocateResponse{}, false, err
		}
		return AllocateResponse{
			Host:     entry.Host,
			Port:     entry.Port,
			CertHash: entry.CertHash,
		}, true, nil
	}
	return AllocateResponse{}, false, nil
}

// ActiveHandler returns the first active allocation so new visitors can
// automatically join an in-progress raid.
type ActiveHandler struct {
	locator *raidLocator
	store   *AllocationStore
}

func NewActiveHandler(namespace string, store *AllocationStore) *ActiveHandler {
	locator, err := newRaidLocator(namespace)
	if err != nil {
		slog.Warn("agones lookup not available for active raid", "error", err)
	}
	return &ActiveHandler{locator: locator, store: store}
}

func (h *ActiveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.locator != nil {
		entry, ok, err := h.locator.findFirstActive(r.Context())
		if err != nil {
			slog.Error("failed to look up active raid from agones", "error", err)
		} else if ok {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(entry); err != nil {
				slog.Error("failed to encode active response", "error", err)
			}
			return
		}
	}

	entry, ok := h.store.First()
	if !ok {
		http.Error(w, "no active raids", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(entry); err != nil {
		slog.Error("failed to encode active response", "error", err)
	}
}

// JoinHandler looks up an existing allocation by lobbyId so late-joiners
// can connect to an in-progress game server.
type JoinHandler struct {
	locator *raidLocator
	store   *AllocationStore
}

func NewJoinHandler(namespace string, store *AllocationStore) *JoinHandler {
	locator, err := newRaidLocator(namespace)
	if err != nil {
		slog.Warn("agones lookup not available for join raid", "error", err)
	}
	return &JoinHandler{locator: locator, store: store}
}

func (h *JoinHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	lobbyID := r.URL.Query().Get("lobbyId")
	if lobbyID == "" {
		http.Error(w, "lobbyId query parameter is required", http.StatusBadRequest)
		return
	}

	if h.locator != nil {
		resp, ok, err := h.locator.findByLobbyID(r.Context(), lobbyID)
		if err != nil {
			slog.Error("failed to look up raid by lobby id from agones", "lobbyId", lobbyID, "error", err)
		} else if ok {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				slog.Error("failed to encode join response", "error", err)
			}
			return
		}
	}

	resp, ok := h.store.Get(lobbyID)
	if !ok {
		http.Error(w, "no allocation found for this lobby", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to encode join response", "error", err)
	}
}
