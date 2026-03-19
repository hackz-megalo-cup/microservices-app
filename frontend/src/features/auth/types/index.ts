export interface AuthCredentials {
  email: string;
  password: string;
}

export interface AuthUser {
  id: string;
  email: string;
  name: string;
  role: string;
}

export interface LoginResponse {
  token: string;
  user: AuthUser;
}

export interface RegisterResponse {
  id: string;
  email: string;
  name: string;
  role: string;
}

export interface UserProfileData {
  id: string;
  email: string;
  name: string;
  role: string;
  createdAt: string | null;
}

export interface UserProfileResult {
  profile: UserProfileData | null;
  isLoading: boolean;
  error: Error | null;
}

export interface AuthContextValue {
  user: AuthUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, name: string) => Promise<void>;
  loginAsGuest: (name: string) => Promise<void>;
  logout: () => void;
}
