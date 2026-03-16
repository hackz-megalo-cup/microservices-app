package agones

import (
	"context"
	"fmt"
	"log"
	"time"

	agonessdk "agones.dev/agones/pkg/sdk"
	sdk "agones.dev/agones/sdks/go"
)

type Lifecycle struct {
	sdk    *sdk.SDK
	cancel context.CancelFunc
}

func NewLifecycle() (*Lifecycle, error) {
	s, err := sdk.NewSDK()
	if err != nil {
		return nil, fmt.Errorf("agones SDK init: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	lc := &Lifecycle{sdk: s, cancel: cancel}

	// Start health ping immediately
	go lc.healthLoop(ctx)

	return lc, nil
}

func (lc *Lifecycle) healthLoop(ctx context.Context) {
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		if err := lc.sdk.Health(); err != nil {
			log.Printf("health ping failed: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

func (lc *Lifecycle) Port() (int32, error) {
	gs, err := lc.sdk.GameServer()
	if err != nil {
		return 0, fmt.Errorf("get game server: %w", err)
	}
	if len(gs.Status.Ports) == 0 {
		return 0, fmt.Errorf("no ports allocated")
	}
	return gs.Status.Ports[0].Port, nil
}

func (lc *Lifecycle) SetCertHash(hash string) error {
	return lc.sdk.SetAnnotation("cert-hash", hash)
}

func (lc *Lifecycle) WatchAllocated(callback func(annotations map[string]string)) error {
	return lc.sdk.WatchGameServer(func(gs *agonessdk.GameServer) {
		if gs.Status.State == "Allocated" {
			callback(gs.ObjectMeta.Annotations)
		}
	})
}

func (lc *Lifecycle) Ready() error {
	return lc.sdk.Ready()
}

func (lc *Lifecycle) Shutdown() error {
	lc.cancel()
	return lc.sdk.Shutdown()
}
