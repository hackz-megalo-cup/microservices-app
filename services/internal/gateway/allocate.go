package gateway

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	agonesclient "agones.dev/agones/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

type AllocateHandler struct {
	agonesClient agonesclient.Interface
	namespace    string
	httpClient   *http.Client
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

func NewAllocateHandler(namespace string) (*AllocateHandler, error) {
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

	address := result.Status.Address
	port := result.Status.Ports[0].Port

	certHash, err := h.fetchCertHash(r.Context(), address, port)
	if err != nil {
		slog.Error("failed to fetch cert hash", "error", err, "address", address, "port", port)
		http.Error(w, "failed to fetch cert hash from game server", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(AllocateResponse{
		Host:     address,
		Port:     port,
		CertHash: certHash,
	}); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
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
