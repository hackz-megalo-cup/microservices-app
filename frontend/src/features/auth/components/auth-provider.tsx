import { useMutation } from "@connectrpc/connect-query";
import type { ReactNode } from "react";
import { useCallback, useEffect, useState } from "react";
import { loginUser, registerUser } from "../../../gen/auth/v1/auth-AuthService_connectquery";
import type { AuthContextValue, AuthUser } from "../types";
import { AuthContext } from "./auth-context-internal";

const TOKEN_KEY = "demo_jwt";
const USER_KEY = "auth_user";

export function AuthProvider({ children }: { children: ReactNode }) {
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

  const registerMutation = useMutation(registerUser);
  const loginMutation = useMutation(loginUser);

  const loginAsGuest = useCallback(
    async (name: string) => {
      const guestId = crypto.randomUUID().slice(0, 8);
      const email = `guest_${guestId}@guest.local`;
      const password = crypto.randomUUID();

      await registerMutation.mutateAsync({ email, password, name });
      const data = await loginMutation.mutateAsync({ email, password });

      if (!data.user) {
        throw new Error("server response error: missing user data");
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
    [registerMutation, loginMutation],
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
    loginAsGuest,
    logout,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
