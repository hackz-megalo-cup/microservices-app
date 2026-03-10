import type { Interceptor } from "@connectrpc/connect";

export const authInterceptor: Interceptor = (next) => async (req) => {
  const token = localStorage.getItem("demo_jwt");
  if (token) {
    req.header.set("authorization", `Bearer ${token}`);
  }
  return next(req);
};
