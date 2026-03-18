import { gameServerUrl, parsedGameServerUrl, resolveApiUrl } from "../../../lib/runtime-config";

export interface ConnectionInfo {
  host: string;
  port: string;
  certHash: string;
  lobbyId: string | undefined;
}

const allocateApiUrl = resolveApiUrl("/api/raid/allocate");
const activeApiUrl = resolveApiUrl("/api/raid/active");

export async function findActiveRaid(signal: AbortSignal): Promise<ConnectionInfo | null> {
  try {
    const res = await fetch(activeApiUrl, { signal });
    if (!res.ok) {
      return null;
    }
    const data = await res.json();
    if (!data.host || !data.certHash || !data.port) {
      return null;
    }
    return {
      host: String(data.host),
      port: String(data.port),
      certHash: String(data.certHash).trim(),
      lobbyId: data.lobbyId ? String(data.lobbyId) : undefined,
    };
  } catch {
    return null;
  }
}

export async function allocateRaid(
  signal: AbortSignal,
  lobbyId?: string,
): Promise<ConnectionInfo | null> {
  try {
    const resolvedLobbyId = lobbyId ?? crypto.randomUUID();
    const res = await fetch(allocateApiUrl, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        lobbyId: resolvedLobbyId,
        bossPokemonId: crypto.randomUUID(),
      }),
      signal,
    });
    if (!res.ok) {
      return null;
    }
    const data = await res.json();
    if (!data.host || !data.certHash || !data.port) {
      return null;
    }
    return {
      host: String(data.host),
      port: String(data.port),
      certHash: String(data.certHash),
      lobbyId: resolvedLobbyId,
    };
  } catch {
    return null;
  }
}

export async function fetchDirectConnection(signal: AbortSignal): Promise<ConnectionInfo | null> {
  if (!gameServerUrl || !parsedGameServerUrl) {
    return null;
  }
  try {
    const res = await fetch(`${gameServerUrl}/cert-hash`, { signal });
    if (!res.ok) {
      return null;
    }
    const hash = (await res.text()).trim();
    return {
      host: parsedGameServerUrl.hostname,
      port: parsedGameServerUrl.port,
      certHash: hash,
      lobbyId: undefined,
    };
  } catch {
    return null;
  }
}
