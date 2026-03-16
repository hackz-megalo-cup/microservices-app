import { useCallback, useRef, useState } from "react";

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
// Types
// ---------------------------------------------------------------------------
type ConnectionState = "disconnected" | "connecting" | "connected" | "error";

interface LogEntry {
  id: number;
  time: string;
  direction: "\u2192" | "\u2190";
  data: string;
}

interface RttRecord {
  protocol: "wt" | "ws";
  rtt: number;
}

let logSeqCounter = 0;

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

function generateUuid(): string {
  return crypto.randomUUID();
}

function timestamp(): string {
  return new Date().toLocaleTimeString("ja-JP", { hour12: false });
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------
export function RaidTestPage() {
  // --- Form state ---
  const [host, setHost] = useState("localhost");
  const [port, setPort] = useState("");
  const [certHash, setCertHash] = useState("");
  const [protocol, setProtocol] = useState<"wt" | "ws">("wt");
  const [userId] = useState(generateUuid);

  // --- Connection state ---
  const [connectionState, setConnectionState] = useState<ConnectionState>("disconnected");

  // --- Battle state ---
  const [bossHp, setBossHp] = useState(0);
  const [bossMaxHp, setBossMaxHp] = useState(0);
  const [tapCount, setTapCount] = useState(0);
  const [requiredForSpecial] = useState(10);
  const [result, setResult] = useState<string | null>(null);

  // --- Log ---
  const [messages, setMessages] = useState<LogEntry[]>([]);

  // --- RTT measurement ---
  const [lastRtt, setLastRtt] = useState<number | null>(null);
  const [rttHistory, setRttHistory] = useState<RttRecord[]>([]);
  const tapSentAtRef = useRef<number>(0);

  // --- Refs for mutable connection handles ---
  const transportRef = useRef<WebTransport | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const dgWriterRef = useRef<WritableStreamDefaultWriter | null>(null);

  // --- Logging helper ---
  const addLog = useCallback((direction: "\u2192" | "\u2190", data: string) => {
    const entry: LogEntry = { id: ++logSeqCounter, time: timestamp(), direction, data };
    setMessages((prev) => [entry, ...prev]);
  }, []);

  // --- Message handler ---
  const handleMessage = useCallback(
    (raw: string) => {
      addLog("\u2190", raw);
      try {
        const msg = JSON.parse(raw);
        switch (msg.t) {
          case "joined":
            if (msg.bossHp != null) {
              setBossHp(msg.bossHp);
            }
            if (msg.bossMaxHp != null) {
              setBossMaxHp(msg.bossMaxHp);
            }
            break;
          case "hp":
            if (msg.hp != null) {
              setBossHp(msg.hp);
            }
            if (tapSentAtRef.current > 0) {
              const rtt = performance.now() - tapSentAtRef.current;
              tapSentAtRef.current = 0;
              setLastRtt(rtt);
              setRttHistory((prev) => {
                const record: RttRecord = { protocol, rtt };
                return [...prev, record].slice(-100);
              });
            }
            break;
          case "special_used":
            if (msg.bossHp != null) {
              setBossHp(msg.bossHp);
            }
            break;
          case "finished":
            setResult(msg.result ?? "finished");
            break;
        }
      } catch {
        // not JSON, just log
      }
    },
    [addLog, protocol],
  );

  // --- Send helpers ---
  const sendReliable = useCallback(
    async (payload: string) => {
      addLog("\u2192", payload);
      if (protocol === "wt" && transportRef.current) {
        const stream = await transportRef.current.createBidirectionalStream();
        const writer = stream.writable.getWriter();
        const encoder = new TextEncoder();
        await writer.write(encoder.encode(payload));
        await writer.close();
        // Drain and close the readable side
        const reader = stream.readable.getReader();
        try {
          while (!(await reader.read()).done) {
            // discard server response on this stream
          }
        } catch {
          // stream closed
        }
      } else if (protocol === "ws" && wsRef.current) {
        wsRef.current.send(payload);
      }
    },
    [protocol, addLog],
  );

  const sendUnreliable = useCallback(
    async (payload: string) => {
      addLog("\u2192", payload);
      if (protocol === "wt" && dgWriterRef.current) {
        const encoded = new TextEncoder().encode(payload);
        await dgWriterRef.current.write(encoded);
      } else if (protocol === "ws" && wsRef.current) {
        wsRef.current.send(payload);
      }
    },
    [protocol, addLog],
  );

  // --- Read loops for WebTransport ---
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
              handleMessage(line);
            }
          }
        }
      } catch {
        // stream closed
      }
    },
    [handleMessage],
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
          handleMessage(decoder.decode(value));
        }
      } catch {
        // closed
      }
    },
    [handleMessage],
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
          const streamReader = stream.getReader();
          readStream(streamReader);
        }
      } catch {
        // closed
      }
    },
    [readStream],
  );

  // --- Connect ---
  const connect = useCallback(async () => {
    setConnectionState("connecting");
    setResult(null);
    setTapCount(0);

    try {
      if (protocol === "wt") {
        const hashBytes = hexToUint8Array(certHash);
        const transport = new WebTransport(`https://${host}:${port}/wt`, {
          serverCertificateHashes: [
            {
              algorithm: "sha-256",
              value: hashBytes.buffer as ArrayBuffer,
            },
          ],
        });
        await transport.ready;
        transportRef.current = transport;

        const dgWriter = transport.datagrams.writable.getWriter();
        dgWriterRef.current = dgWriter;

        readDatagrams(transport);
        readIncomingUniStreams(transport);

        setConnectionState("connected");

        transport.closed
          .then(() => {
            setConnectionState("disconnected");
          })
          .catch(() => {
            setConnectionState("disconnected");
          });
      } else {
        const ws = new WebSocket(`wss://${host}:${port}/ws`);
        wsRef.current = ws;
        ws.onopen = () => setConnectionState("connected");
        ws.onmessage = (event) => handleMessage(String(event.data));
        ws.onclose = () => setConnectionState("disconnected");
        ws.onerror = () => setConnectionState("error");
      }
    } catch (err) {
      setConnectionState("error");
      addLog("\u2190", `ERROR: ${err instanceof Error ? err.message : String(err)}`);
    }
  }, [
    protocol,
    host,
    port,
    certHash,
    handleMessage,
    addLog,
    readDatagrams,
    readIncomingUniStreams,
  ]);

  // --- Disconnect ---
  const disconnect = useCallback(() => {
    if (transportRef.current) {
      transportRef.current.close();
      transportRef.current = null;
      dgWriterRef.current = null;
    }
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    setConnectionState("disconnected");
  }, []);

  // --- Actions ---
  const handleJoin = () => {
    sendReliable(JSON.stringify({ t: "join", userId }));
  };

  const handleTap = () => {
    tapSentAtRef.current = performance.now();
    sendUnreliable(JSON.stringify({ t: "tap" }));
    setTapCount((c) => c + 1);
  };

  const handleSpecial = () => {
    sendReliable(JSON.stringify({ t: "special", userId }));
    setTapCount(0);
  };

  // --- Derived ---
  const isConnected = connectionState === "connected";
  const hpPercent = bossMaxHp > 0 ? (bossHp / bossMaxHp) * 100 : 0;
  const statusIndicator =
    connectionState === "connected"
      ? "\uD83D\uDFE2"
      : connectionState === "connecting"
        ? "\uD83D\uDFE1"
        : connectionState === "error"
          ? "\uD83D\uDD34"
          : "\u26AA";

  const calcStats = (proto: "wt" | "ws") => {
    const records = rttHistory.filter((r) => r.protocol === proto);
    if (records.length === 0) {
      return null;
    }
    const rtts = records.map((r) => r.rtt);
    return {
      count: rtts.length,
      avg: rtts.reduce((a, b) => a + b, 0) / rtts.length,
      min: Math.min(...rtts),
      max: Math.max(...rtts),
    };
  };
  const wtStats = calcStats("wt");
  const wsStats = calcStats("ws");

  return (
    <main className="min-h-dvh bg-bg-primary text-text-primary font-sans p-6">
      <div className="mx-auto max-w-2xl space-y-6">
        {/* Header */}
        <h1 className="text-2xl font-bold tracking-tight text-accent">RAID BATTLE DEBUG</h1>

        {/* Connection Form */}
        <section className="rounded-xl bg-bg-card p-5 shadow-card space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <label className="space-y-1">
              <span className="text-sm text-text-secondary">Host</span>
              <input
                type="text"
                value={host}
                onChange={(e) => setHost(e.target.value)}
                className="w-full rounded-lg bg-bg-primary px-3 py-2 text-sm text-text-primary border border-bg-hover focus:border-accent focus:outline-none"
              />
            </label>
            <label className="space-y-1">
              <span className="text-sm text-text-secondary">Port</span>
              <input
                type="text"
                value={port}
                onChange={(e) => setPort(e.target.value)}
                placeholder="7003"
                className="w-full rounded-lg bg-bg-primary px-3 py-2 text-sm text-text-primary border border-bg-hover focus:border-accent focus:outline-none"
              />
            </label>
          </div>

          <label className="block space-y-1">
            <span className="text-sm text-text-secondary">Cert Hash (hex, WebTransport only)</span>
            <input
              type="text"
              value={certHash}
              onChange={(e) => setCertHash(e.target.value)}
              placeholder="e.g. a1b2c3d4..."
              className="w-full rounded-lg bg-bg-primary px-3 py-2 text-sm font-mono text-text-primary border border-bg-hover focus:border-accent focus:outline-none"
            />
          </label>

          <fieldset className="flex items-center gap-6">
            <legend className="text-sm text-text-secondary mb-1">Protocol</legend>
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="radio"
                name="protocol"
                value="wt"
                checked={protocol === "wt"}
                onChange={() => setProtocol("wt")}
                className="accent-accent"
              />
              <span className="text-sm">WebTransport</span>
            </label>
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="radio"
                name="protocol"
                value="ws"
                checked={protocol === "ws"}
                onChange={() => setProtocol("ws")}
                className="accent-accent"
              />
              <span className="text-sm">WebSocket</span>
            </label>
          </fieldset>

          <div className="space-y-1">
            <span className="text-sm text-text-secondary">User ID</span>
            <p className="rounded-lg bg-bg-primary px-3 py-2 text-sm font-mono text-text-secondary select-all">
              {userId}
            </p>
          </div>

          <div className="flex gap-3">
            <button
              type="button"
              onClick={connect}
              disabled={connectionState === "connecting" || isConnected}
              className="rounded-lg bg-accent px-5 py-2 text-sm font-semibold text-bg-primary hover:bg-accent-dark disabled:opacity-40 disabled:cursor-not-allowed transition"
            >
              CONNECT
            </button>
            <button
              type="button"
              onClick={disconnect}
              disabled={!isConnected}
              className="rounded-lg border border-bg-hover px-5 py-2 text-sm font-semibold text-text-primary hover:bg-bg-hover disabled:opacity-40 disabled:cursor-not-allowed transition"
            >
              DISCONNECT
            </button>
            {isConnected && (
              <button
                type="button"
                onClick={handleJoin}
                className="rounded-lg bg-green px-5 py-2 text-sm font-semibold text-bg-primary hover:opacity-80 transition"
              >
                JOIN
              </button>
            )}
          </div>
        </section>

        {/* Status */}
        <section className="rounded-xl bg-bg-card p-4 shadow-card flex items-center gap-3">
          <span className="text-lg">{statusIndicator}</span>
          <span className="text-sm font-medium capitalize">{connectionState}</span>
          {result && (
            <span className="ml-auto rounded-full bg-accent/20 px-3 py-1 text-xs font-semibold text-accent">
              {result}
            </span>
          )}
        </section>

        {/* Boss HP */}
        <section className="rounded-xl bg-bg-card p-5 shadow-card space-y-2">
          <h2 className="text-sm font-semibold text-text-secondary">Boss HP</h2>
          <div className="h-6 w-full overflow-hidden rounded-full bg-bg-primary">
            <div
              className="h-full rounded-full bg-accent transition-all duration-300"
              style={{ width: `${hpPercent}%` }}
            />
          </div>
          <p className="text-right text-sm font-mono text-text-secondary">
            {bossHp.toLocaleString()} / {bossMaxHp.toLocaleString()}
          </p>
        </section>

        {/* RTT Stats */}
        {(wtStats || wsStats || lastRtt !== null) && (
          <section className="rounded-xl bg-bg-card p-5 shadow-card space-y-3">
            <h2 className="text-sm font-semibold text-text-secondary">Tap RTT (ms)</h2>

            {lastRtt !== null && (
              <p className="text-center text-2xl font-bold font-mono text-accent">
                {lastRtt.toFixed(1)} ms
                <span className="text-sm font-normal text-text-secondary ml-2">
                  ({protocol === "wt" ? "WebTransport" : "WebSocket"})
                </span>
              </p>
            )}

            {(wtStats || wsStats) && (
              <div className="grid grid-cols-2 gap-4 mt-3">
                {/* WT column */}
                <div
                  className={`rounded-lg p-3 ${wtStats ? "bg-bg-primary" : "bg-bg-primary/50 opacity-50"}`}
                >
                  <h3 className="text-xs font-semibold text-accent mb-2">WebTransport (UDP)</h3>
                  {wtStats ? (
                    <div className="space-y-1 font-mono text-sm">
                      <div className="flex justify-between">
                        <span className="text-text-secondary">Avg</span>
                        <span className="text-text-primary font-bold">
                          {wtStats.avg.toFixed(1)}
                        </span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-text-secondary">Min</span>
                        <span className="text-green">{wtStats.min.toFixed(1)}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-text-secondary">Max</span>
                        <span className="text-text-primary">{wtStats.max.toFixed(1)}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-text-secondary">Count</span>
                        <span className="text-text-primary">{wtStats.count}</span>
                      </div>
                    </div>
                  ) : (
                    <p className="text-xs text-text-secondary">No data</p>
                  )}
                </div>

                {/* WS column */}
                <div
                  className={`rounded-lg p-3 ${wsStats ? "bg-bg-primary" : "bg-bg-primary/50 opacity-50"}`}
                >
                  <h3 className="text-xs font-semibold text-accent mb-2">WebSocket (TCP)</h3>
                  {wsStats ? (
                    <div className="space-y-1 font-mono text-sm">
                      <div className="flex justify-between">
                        <span className="text-text-secondary">Avg</span>
                        <span className="text-text-primary font-bold">
                          {wsStats.avg.toFixed(1)}
                        </span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-text-secondary">Min</span>
                        <span className="text-green">{wsStats.min.toFixed(1)}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-text-secondary">Max</span>
                        <span className="text-text-primary">{wsStats.max.toFixed(1)}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-text-secondary">Count</span>
                        <span className="text-text-primary">{wsStats.count}</span>
                      </div>
                    </div>
                  ) : (
                    <p className="text-xs text-text-secondary">No data</p>
                  )}
                </div>
              </div>
            )}
          </section>
        )}

        {/* Attack buttons */}
        <section className="rounded-xl bg-bg-card p-5 shadow-card space-y-3">
          <div className="flex gap-3">
            <button
              type="button"
              onClick={handleTap}
              disabled={!isConnected}
              className="flex-1 rounded-lg bg-accent py-3 text-base font-bold text-bg-primary hover:bg-accent-dark disabled:opacity-40 disabled:cursor-not-allowed transition active:scale-95"
            >
              TAP ATTACK
            </button>
            <button
              type="button"
              onClick={handleSpecial}
              disabled={!isConnected || tapCount < requiredForSpecial}
              className="flex-1 rounded-lg bg-green py-3 text-base font-bold text-bg-primary hover:opacity-80 disabled:opacity-40 disabled:cursor-not-allowed transition active:scale-95"
            >
              SPECIAL
            </button>
          </div>
          <p className="text-center text-sm text-text-secondary">
            Taps: <span className="font-mono font-semibold text-text-primary">{tapCount}</span> /
            Required:{" "}
            <span className="font-mono font-semibold text-text-primary">{requiredForSpecial}</span>
          </p>
        </section>

        {/* Message Log */}
        <section className="rounded-xl bg-bg-card p-5 shadow-card space-y-3">
          <h2 className="text-sm font-semibold text-text-secondary text-center">
            Message Log (newest first)
          </h2>
          <div className="max-h-80 overflow-y-auto rounded-lg bg-bg-primary p-3 font-mono text-xs leading-relaxed">
            {messages.length === 0 ? (
              <p className="text-text-secondary text-center">No messages yet</p>
            ) : (
              messages.map((msg) => (
                <div
                  key={msg.id}
                  className={msg.direction === "\u2192" ? "text-accent" : "text-text-primary"}
                >
                  <span className="text-text-secondary">{msg.time}</span> {msg.direction} {msg.data}
                </div>
              ))
            )}
          </div>
        </section>
      </div>
    </main>
  );
}
