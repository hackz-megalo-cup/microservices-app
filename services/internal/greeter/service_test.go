package greeter

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"

	callerv1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/caller/v1"
	greeterv1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/greeter/v1"
)

// mockCallerClient is a hand-written mock of callerv1connect.CallerServiceClient.
type mockCallerClient struct {
	resp *connect.Response[callerv1.CallExternalResponse]
	err  error
}

func (m *mockCallerClient) CallExternal(_ context.Context, _ *connect.Request[callerv1.CallExternalRequest]) (*connect.Response[callerv1.CallExternalResponse], error) {
	return m.resp, m.err
}

func TestGreet_Normal(t *testing.T) {
	mock := &mockCallerClient{
		resp: connect.NewResponse(&callerv1.CallExternalResponse{
			StatusCode: 200,
			BodyLength: 42,
		}),
	}
	svc := NewService(mock, "http://example.com", 5*time.Second, nil)

	resp, err := svc.Greet(context.Background(), connect.NewRequest(&greeterv1.GreetRequest{Name: "Alice"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetMessage() != "Hello Alice from greeter-service!" {
		t.Errorf("got message %q, want %q", resp.Msg.GetMessage(), "Hello Alice from greeter-service!")
	}
	if resp.Msg.GetExternalStatus() != 200 {
		t.Errorf("got status %d, want 200", resp.Msg.GetExternalStatus())
	}
	if resp.Msg.GetExternalBodyLength() != 42 {
		t.Errorf("got body length %d, want 42", resp.Msg.GetExternalBodyLength())
	}
}

func TestGreet_EmptyNameFallback(t *testing.T) {
	mock := &mockCallerClient{
		resp: connect.NewResponse(&callerv1.CallExternalResponse{
			StatusCode: 200,
			BodyLength: 10,
		}),
	}
	svc := NewService(mock, "http://example.com", 5*time.Second, nil)

	resp, err := svc.Greet(context.Background(), connect.NewRequest(&greeterv1.GreetRequest{Name: ""}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetMessage() != "Hello World from greeter-service!" {
		t.Errorf("got message %q, want %q", resp.Msg.GetMessage(), "Hello World from greeter-service!")
	}
}

func TestGreet_CallerConnectError(t *testing.T) {
	callerErr := connect.NewError(connect.CodeInternal, errors.New("caller exploded"))
	mock := &mockCallerClient{err: callerErr}
	svc := NewService(mock, "http://example.com", 5*time.Second, nil)

	_, err := svc.Greet(context.Background(), connect.NewRequest(&greeterv1.GreetRequest{Name: "Bob"}))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		t.Fatalf("expected connect.Error, got %T", err)
	}
	if connectErr.Code() != connect.CodeInternal {
		t.Errorf("got code %v, want %v", connectErr.Code(), connect.CodeInternal)
	}
}

func TestGreet_CallerGenericError(t *testing.T) {
	mock := &mockCallerClient{err: errors.New("network failure")}
	svc := NewService(mock, "http://example.com", 5*time.Second, nil)

	_, err := svc.Greet(context.Background(), connect.NewRequest(&greeterv1.GreetRequest{Name: "Charlie"}))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		t.Fatalf("expected connect.Error, got %T", err)
	}
	if connectErr.Code() != connect.CodeUnavailable {
		t.Errorf("got code %v, want %v", connectErr.Code(), connect.CodeUnavailable)
	}
}
