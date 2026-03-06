package gateway

import (
	"net/http"
	"testing"

	"connectrpc.com/connect"
)

func TestMapHTTPStatusToConnectCode(t *testing.T) {
	testCases := []struct {
		name   string
		status int
		want   connect.Code
	}{
		{name: "400", status: http.StatusBadRequest, want: connect.CodeInvalidArgument},
		{name: "401", status: http.StatusUnauthorized, want: connect.CodeUnauthenticated},
		{name: "403", status: http.StatusForbidden, want: connect.CodePermissionDenied},
		{name: "404", status: http.StatusNotFound, want: connect.CodeNotFound},
		{name: "409", status: http.StatusConflict, want: connect.CodeAlreadyExists},
		{name: "429", status: http.StatusTooManyRequests, want: connect.CodeResourceExhausted},
		{name: "502", status: http.StatusBadGateway, want: connect.CodeUnavailable},
		{name: "503", status: http.StatusServiceUnavailable, want: connect.CodeUnavailable},
		{name: "504", status: http.StatusGatewayTimeout, want: connect.CodeDeadlineExceeded},
		{name: "500", status: http.StatusInternalServerError, want: connect.CodeInternal},
		{name: "418", status: http.StatusTeapot, want: connect.CodeUnknown},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := MapHTTPStatusToConnectCode(tc.status)
			if got != tc.want {
				t.Fatalf("status %d: got %s want %s", tc.status, got.String(), tc.want.String())
			}
		})
	}
}
