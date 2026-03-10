import { useState } from "react";
import { loginUser, registerUser } from "../api/auth-api";
import type { AuthCredentials, AuthState } from "../types";

const TOKEN_KEY = "demo_jwt";

export function useAuth() {
  const [email, setEmail] = useState("demo@example.com");
  const [password, setPassword] = useState("password");
  const [state, setState] = useState<AuthState>({
    status: localStorage.getItem(TOKEN_KEY) ? "Token is set" : "No token",
    response: null,
  });

  const register = async () => {
    setState({ status: "Registering...", response: null });
    try {
      const credentials: AuthCredentials = { email, password };
      const data = await registerUser(credentials);
      setState({
        status: "Registered successfully (users table に INSERT 済み)",
        response: data as unknown as Record<string, unknown>,
      });
    } catch (err) {
      setState({
        status: err instanceof Error ? err.message : "register failed",
        response: null,
      });
    }
  };

  const login = async () => {
    setState({ status: "Logging in...", response: null });
    try {
      const credentials: AuthCredentials = { email, password };
      const data = await loginUser(credentials);
      localStorage.setItem(TOKEN_KEY, data.token);
      setState({
        status: "Logged in (DB から bcrypt 検証 → JWT 発行)",
        response: data as unknown as Record<string, unknown>,
      });
    } catch (err) {
      setState({
        status: err instanceof Error ? err.message : "login failed",
        response: null,
      });
    }
  };

  const clear = () => {
    localStorage.removeItem(TOKEN_KEY);
    setState({ status: "Token cleared", response: null });
  };

  return {
    email,
    setEmail,
    password,
    setPassword,
    status: state.status,
    response: state.response,
    register,
    login,
    clear,
  };
}
