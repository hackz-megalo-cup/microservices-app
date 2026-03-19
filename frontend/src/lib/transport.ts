import { createConnectTransport } from "@connectrpc/connect-web";
import { authInterceptor } from "../interceptors/auth";
import { apiBaseUrl } from "./runtime-config";

export { apiBaseUrl } from "./runtime-config";

export const transport = createConnectTransport({
  baseUrl: apiBaseUrl,
  defaultTimeoutMs: 10_000,
  useBinaryFormat: false,
  interceptors: [authInterceptor],
});

export const streamTransport = createConnectTransport({
  baseUrl: apiBaseUrl,
  useBinaryFormat: false,
  interceptors: [authInterceptor],
});
