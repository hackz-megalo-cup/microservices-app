import { useCallback, useEffect, useRef, useState } from "react";
import { allocateRaid, fetchDirectConnection, findActiveRaid } from "../api/connection-info";
import type { ConnectionState, ServerMessage } from "../types";

// ---------------------------------------------------------------------------
// WebTransport type declarations (not yet in lib.dom.d.ts)
// ---------------------------------------------------------------------------
interface WebTransportHash {
  algorithm: string;
  value: ArrayBuffer;
}

interface WebTransportOptions {
  serverCertificateHashes?: WebTransportHash[];
}

interface WebTransportDatagramDuplexStream {
  readable: ReadableStream<Uint8Array>;
  writable: WritableStream<Uint8Array>;
}

interface WebTransport {
  ready: Promise<void>;
  closed: Promise<void>;
  datagrams: WebTransportDatagramDuplexStream;
  incomingUnidirectionalStreams: ReadableStream<ReadableStream<Uint8Array>>;
  createBidirectionalStream(): Promise<{
    readable: ReadableStream<Uint8Array>;
    writable: WritableStream<Uint8Array>;
  }>;
  close(): void;
}

declare const WebTransport: {
  prototype: WebTransport;
  new (url: string, options?: WebTransportOptions): WebTransport;
};

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
function hexToUint8Array(hex: string): Uint8Array {
  const clean = hex.replace(/\s+/g, "");
  const bytes = new Uint8Array(clean.length / 2);
  for (let i = 0; i < clean.length; i += 2) {
    bytes[i / 2] = Number.parseInt(clean.substring(i, i + 2), 16);
  }
  return bytes;
}

function isIosDevice(): boolean {
  if (typeof navigator === "undefined") {
    return false;
  }
  const ua = navigator.userAgent;
  return /iP(hone|ad|od)/.test(ua) || (ua.includes("Macintosh") && navigator.maxTouchPoints > 1);
}

function supportsWebTransport(): boolean {
  return typeof WebTransport !== "undefined";
}

function toWebSocketUrl(url: string): string {
  const wsUrl = new URL(url, typeof window !== "undefined" ? window.location.href : undefined);
  wsUrl.protocol = wsUrl.protocol === "https:" ? "wss:" : "ws:";
  return wsUrl.toString();
}

// ---------------------------------------------------------------------------
// Hook options & return type
// ---------------------------------------------------------------------------
export interface UseGameConnectionOptions {
  userId: string;
  lobbyId?: string;
  onMessage: (msg: ServerMessage) => void;
  autoConnect?: boolean;
}

export interface UseGameConnectionReturn {
  status: ConnectionState;
  sendTap: () => void;
  sendSpecial: () => void;
  disconnect: () => void;
}

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------
import { resolveApiUrl } from "../../../lib/runtime-config";

export function useGameConnection({
  userId,
  lobbyId,
  onMessage,
  autoConnect = true,
}: UseGameConnectionOptions): UseGameConnectionReturn {
  const [status, setStatus] = useState<ConnectionState>("disconnected");

  const transportRef = useRef<WebTransport | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const dgWriterRef = useRef<WritableStreamDefaultWriter | null>(null);
  const protocolRef = useRef<"wt" | "ws">("wt");
  const autoConnectedRef = useRef(false);
  const onMessageRef = useRef(onMessage);
  onMessageRef.current = onMessage;

  const userIdRef = useRef(userId);
  userIdRef.current = userId;

  // --- Parse incoming JSON ---
  const handleRaw = useCallback((raw: string) => {
    try {
      const msg = JSON.parse(raw) as ServerMessage;
      if (typeof msg === "object" && msg !== null && "t" in msg) {
        onMessageRef.current(msg);
      }
    } catch {
      // non-JSON, ignore
    }
  }, []);

  // --- WT stream reader ---
  const readStream = useCallback(
    async (reader: ReadableStreamDefaultReader<Uint8Array>) => {
      const decoder = new TextDecoder();
      let buffer = "";
      try {
        for (;;) {
          const { value, done } = await reader.read();
          if (done) {
            break;
          }
          buffer += decoder.decode(value, { stream: true });
          const lines = buffer.split("\n");
          buffer = lines.pop() ?? "";
          for (const line of lines) {
            if (line.trim()) {
              handleRaw(line);
            }
          }
        }
      } catch {
        // stream closed
      }
    },
    [handleRaw],
  );

  const readDatagrams = useCallback(
    async (transport: WebTransport) => {
      const decoder = new TextDecoder();
      const reader = transport.datagrams.readable.getReader();
      try {
        for (;;) {
          const { value, done } = await reader.read();
          if (done) {
            break;
          }
          handleRaw(decoder.decode(value));
        }
      } catch {
        // closed
      }
    },
    [handleRaw],
  );

  const readIncomingUniStreams = useCallback(
    async (transport: WebTransport) => {
      const reader = transport.incomingUnidirectionalStreams.getReader();
      try {
        for (;;) {
          const { value: stream, done } = await reader.read();
          if (done) {
            break;
          }
          readStream(stream.getReader());
        }
      } catch {
        // closed
      }
    },
    [readStream],
  );

  // --- Close ---
  const closeAll = useCallback(() => {
    if (transportRef.current) {
      transportRef.current.close();
      transportRef.current = null;
      dgWriterRef.current = null;
    }
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
  }, []);

  // --- Connect via WebTransport ---
  const connectWt = useCallback(
    async (host: string, port: string, hash: string) => {
      closeAll();
      setStatus("connecting");
      protocolRef.current = "wt";

      const hashBytes = hexToUint8Array(hash);
      const transport = new WebTransport(`https://${host}:${port}/wt`, {
        serverCertificateHashes: [{ algorithm: "sha-256", value: hashBytes.buffer as ArrayBuffer }],
      });
      await transport.ready;
      transportRef.current = transport;

      dgWriterRef.current = transport.datagrams.writable.getWriter();
      readDatagrams(transport);
      readIncomingUniStreams(transport);
      setStatus("connected");

      transport.closed.then(() => setStatus("disconnected")).catch(() => setStatus("disconnected"));

      // Auto-join
      const joinPayload = JSON.stringify({ t: "join", userId: userIdRef.current });
      const stream = await transport.createBidirectionalStream();
      const writer = stream.writable.getWriter();
      await writer.write(new TextEncoder().encode(joinPayload));
      await writer.close();
      const reader = stream.readable.getReader();
      try {
        while (!(await reader.read()).done) {
          /* drain */
        }
      } catch {
        // stream closed
      }
    },
    [closeAll, readDatagrams, readIncomingUniStreams],
  );

  // --- Connect via WebSocket ---
  const connectWs = useCallback(
    async (host: string, port: string, connLobbyId?: string) => {
      closeAll();
      setStatus("connecting");
      protocolRef.current = "ws";

      const wsUrl = connLobbyId
        ? (() => {
            const base = new URL(resolveApiUrl("/api/raid/ws"));
            base.searchParams.set("lobbyId", connLobbyId);
            return toWebSocketUrl(base.toString());
          })()
        : `wss://${host}:${port}/ws`;

      const ws = await new Promise<WebSocket>((resolve, reject) => {
        const socket = new WebSocket(wsUrl);
        let settled = false;
        socket.onopen = () => {
          settled = true;
          resolve(socket);
        };
        socket.onerror = () => {
          if (!settled) {
            settled = true;
            reject(new Error(`WebSocket connection failed: ${wsUrl}`));
          }
        };
        socket.onclose = (event) => {
          if (!settled) {
            settled = true;
            reject(new Error(`WebSocket closed during connect: ${event.code}`));
          }
        };
      });

      wsRef.current = ws;
      ws.onmessage = (event) => handleRaw(String(event.data));
      ws.onclose = () => setStatus("disconnected");
      ws.onerror = () => setStatus("error");
      setStatus("connected");

      // Auto-join
      ws.send(JSON.stringify({ t: "join", userId: userIdRef.current }));
    },
    [closeAll, handleRaw],
  );

  // --- Auto-connect on mount ---
  useEffect(() => {
    if (!autoConnect || autoConnectedRef.current) {
      return;
    }
    autoConnectedRef.current = true;

    const abort = new AbortController();

    const run = async () => {
      try {
        // Direct game server first (local dev), then K8s gateway APIs
        const conn =
          (await fetchDirectConnection(abort.signal)) ??
          (await findActiveRaid(abort.signal)) ??
          (await allocateRaid(abort.signal, lobbyId));
        if (!conn) {
          return;
        }

        const shouldUseWs = isIosDevice() || !supportsWebTransport();

        if (shouldUseWs) {
          await connectWs(conn.host, conn.port, conn.lobbyId);
          return;
        }

        try {
          await connectWt(conn.host, conn.port, conn.certHash);
        } catch {
          // WT failed, fallback to WS
          await connectWs(conn.host, conn.port, conn.lobbyId);
        }
      } catch (err) {
        if (!abort.signal.aborted) {
          console.warn("Auto-connect failed:", err);
          setStatus("disconnected");
        }
      }
    };

    run();

    return () => {
      abort.abort();
      closeAll();
    };
  }, [autoConnect, lobbyId, connectWt, connectWs, closeAll]);

  // --- Send helpers (I3: try/catch wrapped) ---
  const sendTap = useCallback(() => {
    try {
      const payload = JSON.stringify({ t: "tap" });
      if (protocolRef.current === "wt" && dgWriterRef.current) {
        dgWriterRef.current.write(new TextEncoder().encode(payload));
      } else if (wsRef.current) {
        wsRef.current.send(payload);
      }
    } catch {
      // send failed, ignore
    }
  }, []);

  const sendSpecial = useCallback(() => {
    try {
      const payload = JSON.stringify({ t: "special", userId: userIdRef.current });
      if (protocolRef.current === "wt" && transportRef.current) {
        transportRef.current.createBidirectionalStream().then(async (stream) => {
          const writer = stream.writable.getWriter();
          await writer.write(new TextEncoder().encode(payload));
          await writer.close();
          const reader = stream.readable.getReader();
          try {
            while (!(await reader.read()).done) {
              /* drain */
            }
          } catch {
            // closed
          }
        });
      } else if (wsRef.current) {
        wsRef.current.send(payload);
      }
    } catch {
      // send failed, ignore
    }
  }, []);

  const disconnect = useCallback(() => {
    closeAll();
    setStatus("disconnected");
  }, [closeAll]);

  return { status, sendTap, sendSpecial, disconnect };
}
