import { AuthPanel, GatewayDemo } from "../../../lib/demo-components";

export function ApiTestPage() {
  return (
    <main
      style={{
        maxWidth: 760,
        margin: "0 auto",
        fontFamily: "sans-serif",
        padding: "24px 16px",
        minHeight: "100dvh",
        background: "#fff",
        color: "#000",
      }}
    >
      <h1>connect-query Frontend Sample</h1>
      <AuthPanel />
      <GatewayDemo />
    </main>
  );
}
