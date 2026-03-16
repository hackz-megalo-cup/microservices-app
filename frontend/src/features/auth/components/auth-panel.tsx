import { useAuth } from "../hooks/use-auth";

export function AuthPanel() {
  const { email, setEmail, password, setPassword, status, response, register, login, clear } =
    useAuth();

  return (
    <section style={{ border: "1px solid #ddd", padding: 16, borderRadius: 8, marginBottom: 16 }}>
      <h2>
        Auth Demo <span style={{ fontSize: 12, color: "#666" }}>→ auth_db.users</span>
      </h2>
      <p>Register: DB に bcrypt ハッシュで保存 / Login: DB 検索 → bcrypt 検証 → JWT 発行</p>
      <div style={{ display: "flex", gap: 8, alignItems: "center", flexWrap: "wrap" }}>
        <input value={email} onChange={(e) => setEmail(e.target.value)} placeholder="email" />
        <input
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          type="password"
          placeholder="password"
        />
        <button type="button" onClick={() => void register()}>
          Register
        </button>
        <button type="button" onClick={() => void login()}>
          Login
        </button>
        <button type="button" onClick={clear}>
          Clear Token
        </button>
      </div>
      <div style={{ marginTop: 8, fontSize: 14 }}>{status}</div>
      {response && (
        <pre style={{ marginTop: 12, background: "#f7f7f7", padding: 12 }}>
          {JSON.stringify(response, null, 2)}
        </pre>
      )}
    </section>
  );
}
