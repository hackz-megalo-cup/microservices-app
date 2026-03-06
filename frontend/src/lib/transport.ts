import { createConnectTransport } from '@connectrpc/connect-web';
import { authInterceptor } from '../interceptors/auth';

export const apiBaseUrl =
  import.meta.env.VITE_API_BASE_URL ||
  (typeof window !== 'undefined' ? window.location.origin : 'http://localhost:30081');

export const transport = createConnectTransport({
  baseUrl: apiBaseUrl,
  defaultTimeoutMs: 10_000,
  useBinaryFormat: false,
  interceptors: [authInterceptor],
});
