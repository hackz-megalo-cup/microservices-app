const configuredApiBaseUrl = import.meta.env.VITE_API_BASE_URL;
const browserOrigin = typeof window !== "undefined" ? window.location.origin : "";

function stripTrailingSlash(value: string): string {
  return value.endsWith("/") ? value.slice(0, -1) : value;
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

export const apiBaseUrl = configuredApiBaseUrl || browserOrigin;

export const gameServerUrl =
  import.meta.env.VITE_GAME_SERVER_URL || (import.meta.env.DEV ? "https://localhost:7777" : "");

export const parsedGameServerUrl = gameServerUrl ? new URL(gameServerUrl) : null;

export function resolveApiUrl(path: string): string {
  if (!configuredApiBaseUrl) {
    return path;
  }
  return new URL(path, `${stripTrailingSlash(configuredApiBaseUrl)}/`).toString();
}

export function buildApiCorsPattern(): RegExp[] {
  if (!apiBaseUrl) {
    return [];
  }
  return [new RegExp(escapeRegExp(stripTrailingSlash(apiBaseUrl)))];
}
