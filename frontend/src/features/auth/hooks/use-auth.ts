import { useMutation } from "@connectrpc/connect-query";
import { useState } from "react";
import { loginUser, registerUser } from "../../../gen/auth/v1/auth-AuthService_connectquery";

const TOKEN_KEY = "demo_jwt";

export function useAuth() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [name, setName] = useState("");

  const registerMutation = useMutation(registerUser);
  const loginMutation = useMutation(loginUser);

  const register = () => {
    registerMutation.mutate({ email, password, name });
  };

  const login = () => {
    loginMutation.mutate(
      { email, password },
      {
        onSuccess: (data) => {
          localStorage.setItem(TOKEN_KEY, data.token);
        },
      },
    );
  };

  const clear = () => {
    localStorage.removeItem(TOKEN_KEY);
    registerMutation.reset();
    loginMutation.reset();
  };

  const activeMutation =
    loginMutation.submittedAt > registerMutation.submittedAt ? loginMutation : registerMutation;

  const status = activeMutation.isPending
    ? "Processing..."
    : activeMutation.isError
      ? activeMutation.error instanceof Error
        ? activeMutation.error.message
        : "failed"
      : activeMutation.isSuccess
        ? loginMutation.isSuccess
          ? "Logged in (JWT issued)"
          : "Registered successfully"
        : localStorage.getItem(TOKEN_KEY)
          ? "Token is set"
          : "No token";

  const response = activeMutation.data
    ? (activeMutation.data as unknown as Record<string, unknown>)
    : null;

  return {
    email,
    setEmail,
    password,
    setPassword,
    name,
    setName,
    status,
    response,
    register,
    login,
    clear,
  };
}
