import { createClient } from "@connectrpc/connect";
import { useTransport } from "@connectrpc/connect-query";
import type { ReactNode } from "react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { AuthService } from "../../../gen/auth/v1/auth_pb";
import type { AuthContextValue, AuthUser } from "../types";
import { AuthContext } from "./auth-context-internal";

const TOKEN_KEY = "demo_jwt";
const USER_KEY = "auth_user";

export function AuthProvider({ children }: { children: ReactNode }) {
  const transport = useTransport();
  const client = useMemo(() => createClient(AuthService, transport), [transport]);
  const [user, setUser] = useState<AuthUser | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const token = localStorage.getItem(TOKEN_KEY);
    const savedUser = localStorage.getItem(USER_KEY);
    if (token && savedUser) {
      try {
        setUser(JSON.parse(savedUser) as AuthUser);
      } catch {
        localStorage.removeItem(TOKEN_KEY);
        localStorage.removeItem(USER_KEY);
      }
    }
    setIsLoading(false);
  }, []);

  const loginAsGuest = useCallback(
    async (name: string) => {
      const guestId = crypto.randomUUID().slice(0, 8);
      const email = `guest_${guestId}@guest.local`;
      const password = crypto.randomUUID();

      await client.registerUser(
        { email, password, name },
        { headers: new Headers({ "idempotency-key": crypto.randomUUID() }) },
      );
      const data = await client.loginUser(
        { email, password },
        { headers: new Headers({ "idempotency-key": crypto.randomUUID() }) },
      );

      if (!data.user) {
        throw new Error("invalid response: user is missing");
      }

      const authUser: AuthUser = {
        id: data.user.id,
        email: data.user.email,
        name: data.user.name,
        role: data.user.role,
      };

      localStorage.setItem(TOKEN_KEY, data.token);
      localStorage.setItem(USER_KEY, JSON.stringify(authUser));
      setUser(authUser);
    },
    [client],
  );

  const login = useCallback(
    async (email: string, password: string) => {
      const data = await client.loginUser(
        { email, password },
        { headers: new Headers({ "idempotency-key": crypto.randomUUID() }) },
      );

      if (!data.user) {
        throw new Error("invalid response: user is missing");
      }

      const authUser: AuthUser = {
        id: data.user.id,
        email: data.user.email,
        name: data.user.name,
        role: data.user.role,
      };

      localStorage.setItem(TOKEN_KEY, data.token);
      localStorage.setItem(USER_KEY, JSON.stringify(authUser));
      setUser(authUser);
    },
    [client],
  );

  const register = useCallback(
    async (email: string, password: string, name: string) => {
      await client.registerUser(
        { email, password, name },
        { headers: new Headers({ "idempotency-key": crypto.randomUUID() }) },
      );
      await login(email, password);
    },
    [client, login],
  );

  const logout = useCallback(() => {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(USER_KEY);
    setUser(null);
  }, []);

  const value: AuthContextValue = {
    user,
    isAuthenticated: !!user,
    isLoading,
    login,
    register,
    loginAsGuest,
    logout,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
