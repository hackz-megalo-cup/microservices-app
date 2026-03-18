import { Code, ConnectError } from "@connectrpc/connect";
import { QueryCache, QueryClient } from "@tanstack/react-query";

const NON_RETRYABLE_CODES = new Set([
  Code.InvalidArgument,
  Code.NotFound,
  Code.AlreadyExists,
  Code.PermissionDenied,
  Code.Unauthenticated,
  Code.FailedPrecondition,
  Code.Canceled,
]);

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 60_000,
      retry: (failureCount, error) => {
        if (error instanceof ConnectError && NON_RETRYABLE_CODES.has(error.code)) {
          return false;
        }
        return failureCount < 3;
      },
      retryDelay: (attempt) => Math.min(1000 * 2 ** attempt, 30_000),
    },
  },
  queryCache: new QueryCache({
    onError: (error) => {
      console.error("query failed", error);
      if (error instanceof ConnectError && error.code === Code.Unauthenticated) {
        localStorage.removeItem("demo_jwt");
        localStorage.removeItem("auth_user");
        window.location.href = "/login";
      }
    },
  }),
});
