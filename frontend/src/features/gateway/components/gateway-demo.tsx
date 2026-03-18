import { createClient } from "@connectrpc/connect";
import { useMutation } from "@tanstack/react-query";
import { useState } from "react";
import { GatewayService } from "../../../gen/gateway/v1/gateway_pb";
import { transport } from "../../../lib/transport";

export function GatewayDemo() {
  const [name, setName] = useState("World");
  const client = createClient(GatewayService, transport);

  const mutation = useMutation({
    mutationFn: async (nameInput: string) => {
      return client.invokeCustom(
        { name: nameInput },
        {
          headers: new Headers({
            "idempotency-key": crypto.randomUUID(),
          }),
        },
      );
    },
  });

  const handleSubmit = () => {
    mutation.mutate(name);
  };

  return (
    <section style={{ border: "1px solid #ddd", padding: 16, borderRadius: 8, marginTop: 16 }}>
      <h2>
        Gateway Mutation Demo{" "}
        <span style={{ fontSize: 12, color: "#666" }}>
          → gateway_db.invocations
        </span>
      </h2>
      <p>
        Gateway が invocations テーブルへ INSERT します。
      </p>

      <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="name"
          style={{ padding: 8 }}
        />
        <button type="button" onClick={handleSubmit} disabled={mutation.isPending}>
          {mutation.isPending ? "Sending..." : "Invoke Gateway"}
        </button>
      </div>

      {mutation.data && (
        <pre style={{ marginTop: 12, background: "#f7f7f7", padding: 12 }}>
          {JSON.stringify(mutation.data, null, 2)}
        </pre>
      )}

      {mutation.error && (
        <pre style={{ marginTop: 12, color: "#b00020" }}>{mutation.error.message}</pre>
      )}
    </section>
  );
}
