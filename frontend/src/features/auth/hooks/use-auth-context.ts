import { useContext } from "react";
import { AuthContext } from "../components/auth-context-internal";
import type { AuthContextValue } from "../types";

export function useAuthContext(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuthContext must be used within AuthProvider");
  }
  return ctx;
}
