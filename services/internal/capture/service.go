package capture

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	pb "github.com/hackz-megalo-cup/microservices-app/services/gen/go/capture/v1"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

type Service struct {
	eventStore *platform.EventStore
	outbox     *platform.OutboxStore
}

func NewService(eventStore *platform.EventStore, outbox *platform.OutboxStore) *Service {
	return &Service{
		eventStore: eventStore,
		outbox:     outbox,
	}
}

func (s *Service) GetCaptureSession(_ context.Context, _ *connect.Request[pb.GetCaptureSessionRequest]) (*connect.Response[pb.GetCaptureSessionResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("GetCaptureSession is not implemented"))
}

func (s *Service) UseItem(_ context.Context, _ *connect.Request[pb.UseItemRequest]) (*connect.Response[pb.UseItemResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("UseItem is not implemented"))
}

func (s *Service) ThrowBall(_ context.Context, _ *connect.Request[pb.ThrowBallRequest]) (*connect.Response[pb.ThrowBallResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("ThrowBall is not implemented"))
}

func (s *Service) EndSession(_ context.Context, _ *connect.Request[pb.EndSessionRequest]) (*connect.Response[pb.EndSessionResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("EndSession is not implemented"))
}
