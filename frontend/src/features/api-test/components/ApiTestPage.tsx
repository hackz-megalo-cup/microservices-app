import { AuthPanel } from "../../auth/components/AuthPanel";
import { GatewayDemo } from "../../gateway/components/GatewayDemo";
import { GreeterDemo } from "../../greeter/components/GreeterDemo";

export function ApiTestPage() {
  return (
    <main
      style={{ maxWidth: 760, margin: "24px auto", fontFamily: "sans-serif", padding: "0 16px" }}
    >
      <h1>connect-query Frontend Sample</h1>
      <AuthPanel />
      <GreeterDemo />
      <GatewayDemo />
    </main>
  );
}
