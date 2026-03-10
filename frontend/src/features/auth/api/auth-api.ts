import { apiBaseUrl } from "../../../lib/transport";
import type { AuthCredentials, LoginResponse, RegisterResponse } from "../types";

export async function registerUser(credentials: AuthCredentials): Promise<RegisterResponse> {
  const res = await fetch(`${apiBaseUrl}/auth/register`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(credentials),
  });
  const json = await res.json();
  if (!res.ok) {
    throw new Error(json.error || `register failed: ${res.status}`);
  }
  return json as RegisterResponse;
}

export async function loginUser(credentials: AuthCredentials): Promise<LoginResponse> {
  const res = await fetch(`${apiBaseUrl}/auth/login`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(credentials),
  });
  const json = await res.json();
  if (!res.ok) {
    throw new Error(json.error || `login failed: ${res.status}`);
  }
  return json as LoginResponse;
}
