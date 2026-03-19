import { createClient } from "@connectrpc/connect";
import { useQuery, useTransport } from "@connectrpc/connect-query";
import { useMutation } from "@tanstack/react-query";
import { useMemo } from "react";
import { CaptureService } from "../../../gen/capture/v1/capture_pb";
import { getCaptureSession } from "../../../gen/capture/v1/capture-CaptureService_connectquery";

export function useCaptureSession(sessionId: string) {
  const transport = useTransport();
  const client = useMemo(() => createClient(CaptureService, transport), [transport]);
  const invokeCaptureUseItem = useMemo(() => client.useItem.bind(client), [client]);

  const sessionQuery = useQuery(getCaptureSession, { sessionId });

  // Use the real capture UUID from the backend response (not the battleSessionId from the URL)
  const captureSessionId = sessionQuery.data?.sessionId ?? sessionId;

  // All mutations must be declared at top level
  const itemMutation = useMutation({
    mutationFn: async ({ itemId }: { itemId: string }) => {
      const headers = new Headers({ "idempotency-key": crypto.randomUUID() });
      return invokeCaptureUseItem({ sessionId: captureSessionId, itemId }, { headers });
    },
  });

  const ballMutation = useMutation({
    mutationFn: async () => {
      const headers = new Headers({ "idempotency-key": crypto.randomUUID() });
      return client.throwBall({ sessionId: captureSessionId }, { headers });
    },
  });

  const sessionEndMutation = useMutation({
    mutationFn: async () => {
      const headers = new Headers({ "idempotency-key": crypto.randomUUID() });
      return client.endSession({ sessionId: captureSessionId }, { headers });
    },
  });

  const error =
    sessionQuery.error instanceof Error
      ? sessionQuery.error
      : sessionQuery.error
        ? new Error("セッション取得失敗")
        : null;

  return {
    session: sessionQuery.data,
    isLoading: sessionQuery.isPending,
    error,
    refetch: () => void sessionQuery.refetch(),
    itemMutation,
    ballMutation,
    sessionEndMutation,
  };
}
