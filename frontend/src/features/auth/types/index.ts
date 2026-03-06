export interface AuthCredentials {
  email: string;
  password: string;
}

export interface AuthUser {
  id: string;
  email: string;
  role: string;
}

export interface LoginResponse {
  token: string;
  user: AuthUser;
}

export interface RegisterResponse {
  id: string;
  email: string;
  role: string;
}

export interface AuthState {
  status: string;
  response: Record<string, unknown> | null;
}
