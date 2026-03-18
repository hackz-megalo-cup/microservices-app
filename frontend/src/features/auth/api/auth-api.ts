import { createClient } from "@connectrpc/connect";
import type { User as RpcUser } from "../../../gen/auth/v1/auth_pb";
import { AuthService } from "../../../gen/auth/v1/auth_pb";
import { transport } from "../../../lib/transport";
import type { AuthCredentials, LoginResponse, RegisterResponse } from "../types";

const client = createClient(AuthService, transport);

function toUserResponse(user: RpcUser | undefined): RegisterResponse {
  if (!user) {
    throw new Error("invalid response: user is missing");
  }

  return {
    id: user.id,
    email: user.email,
    role: user.role,
  };
}

export async function registerUser(credentials: AuthCredentials): Promise<RegisterResponse> {
  const response = await client.registerUser({
    email: credentials.email,
    password: credentials.password,
  });

  return toUserResponse(response.user);
}

export async function loginUser(credentials: AuthCredentials): Promise<LoginResponse> {
  const response = await client.loginUser({
    email: credentials.email,
    password: credentials.password,
  });

  if (!response.token) {
    throw new Error("invalid response: token is missing");
  }

  return {
    token: response.token,
    user: toUserResponse(response.user),
  };
}
