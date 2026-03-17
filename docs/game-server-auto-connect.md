# Game Server Auto-Connect: EKS対応

## 問題

現在の `raid-test-page.tsx` は `VITE_GAME_SERVER_URL` (デフォルト `https://localhost:7777`) に直接 `/cert-hash` を fetch しているが、EKS上では:
- GameServerのポートはAgones Passthroughで**動的割り当て**（7000-8000）
- GameServerは**allocateされるまでセッションを作らない**（`no active session` で拒否）
- フロントエンドから直接allocator APIは叩けない（クラスタ内部のみ）

## 解決策: Gateway に allocate エンドポイントを追加

```
[Frontend] --POST /api/raid/allocate--> [Gateway] --gRPC--> [Agones Allocator]
                                                                    |
                                                            GameServerAllocation
                                                                    |
[Frontend] <-- {host, port, certHash} -- [Gateway] <-- allocated GS info + /cert-hash fetch
```

## 実装箇所

### 1. Gateway: `/api/raid/allocate` エンドポイント追加

**ファイル**: `services/gateway/` (または新規 `services/gateway/internal/agones/`)

処理フロー:
1. Agones Allocator に `GameServerAllocation` を作成（Kubernetes API経由）
2. 割り当てられた GameServer の address + port を取得
3. その GameServer の `https://<address>:<port>/cert-hash` を fetch
4. フロントエンドに `{ host, port, certHash }` を返す

```go
// POST /api/raid/allocate
// Request: { "lobbyId": "uuid", "bossPokemonId": "uuid" }
// Response: { "host": "ec2-xxx.compute.amazonaws.com", "port": 7292, "certHash": "abcdef..." }

func (h *Handler) AllocateGameServer(w http.ResponseWriter, r *http.Request) {
    var req struct {
        LobbyID      string `json:"lobbyId"`
        BossPokemonID string `json:"bossPokemonId"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    // Kubernetes client で GameServerAllocation を作成
    allocation := &allocationv1.GameServerAllocation{
        ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
        Spec: allocationv1.GameServerAllocationSpec{
            Selectors: []allocationv1.GameServerSelector{{
                LabelSelector: metav1.LabelSelector{
                    MatchLabels: map[string]string{"game": "raid-battle"},
                },
            }},
            MetaPatch: allocationv1.MetaPatch{
                Annotations: map[string]string{
                    "raid.lobby-id":        req.LobbyID,
                    "raid.boss-pokemon-id": req.BossPokemonID,
                },
            },
        },
    }

    result, err := agonesClient.AllocationV1().GameServerAllocations("default").Create(ctx, allocation, metav1.CreateOptions{})
    // result.Status.Address, result.Status.Ports[0].Port が接続先

    // cert-hash を game server から取得
    certHash := fetchCertHash(result.Status.Address, result.Status.Ports[0].Port)

    json.NewEncoder(w).Encode(map[string]any{
        "host":     result.Status.Address,
        "port":     result.Status.Ports[0].Port,
        "certHash": certHash,
    })
}
```

**必要な依存**:
```
go get agones.dev/agones@v1.56.0
go get k8s.io/client-go
```

**必要なRBAC** (infra側で追加済みの `agones-sdk` ClusterRoleか、gateway用に新規):
- `gameserverallocations` の `create` 権限
- gateway の ServiceAccount に紐付け

### 2. Gateway の Kubernetes RBAC

**ファイル**: `deploy/k8s/gateway.nix` または infra 側の nixidy

Gateway pod が Agones Allocator API を叩けるように:
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: gateway-agones-allocator
rules:
  - apiGroups: ["allocation.agones.dev"]
    resources: ["gameserverallocations"]
    verbs: ["create"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: gateway-agones-allocator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: gateway-agones-allocator
subjects:
  - kind: ServiceAccount
    name: default
    namespace: microservices
```

### 3. Traefik IngressRoute 追加 (infra側)

**ファイル**: `nixidy/env/local/traefik.nix`

```nix
{
  apiVersion = "traefik.io/v1alpha1";
  kind = "IngressRoute";
  metadata = {
    name = "raid-allocate-route";
    namespace = "microservices";
  };
  spec = {
    entryPoints = [ "web" ];
    routes = [
      {
        match = "PathPrefix(`/api/raid`)";
        kind = "Rule";
        priority = 95;
        middlewares = [
          { name = "cors-middleware"; }
          { name = "rate-limit-middleware"; }
        ];
        services = [
          {
            name = "gateway";
            port = 8082;
          }
        ];
      }
    ];
  };
}
```

### 4. Frontend: auto-connect を Gateway 経由に変更

**ファイル**: `frontend/src/features/raid-test/components/raid-test-page.tsx`

```tsx
// 変更前
const gameServerUrl = import.meta.env.VITE_GAME_SERVER_URL || "https://localhost:7777";

// 変更後
const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || "http://localhost:30081";

// auto-connect 部分
const autoConnect = async () => {
  // 1. Gateway 経由で GameServer を allocate
  const allocRes = await fetch(`${apiBaseUrl}/api/raid/allocate`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      lobbyId: generateUuid(),
      bossPokemonId: generateUuid(),
    }),
    signal: abort.signal,
  });
  if (!allocRes.ok) return;
  const { host, port, certHash: hash } = await allocRes.json();

  // 2. 取得した接続情報で WebTransport 接続
  setHost(host);
  setPort(String(port));
  setCertHash(hash);

  const hashBytes = hexToUint8Array(hash);
  setConnectionState("connecting");

  const transport = new WebTransport(`https://${host}:${port}/wt`, {
    serverCertificateHashes: [
      { algorithm: "sha-256", value: hashBytes.buffer },
    ],
  });
  // ... 以下既存の接続処理
};
```

### 5. 環境変数

**ローカル開発** (`.env`):
```
VITE_API_BASE_URL=http://localhost:30081
VITE_GAME_SERVER_URL=https://localhost:7777  # ローカルdev用のフォールバック維持
```

**EKS** (Traefik経由なので `VITE_API_BASE_URL` はフロントエンドのオリジンと同じ):
- `VITE_API_BASE_URL` は不要（相対パス `/api/raid/allocate` で動く）
- または明示的に `VITE_API_BASE_URL=https://app.thirdlf03.com`

## 作業順序

1. **Gateway**: allocate エンドポイント実装 + Agones client 追加
2. **RBAC**: gateway ServiceAccount に allocator 権限付与
3. **Traefik**: `/api/raid` ルート追加 (infra側)
4. **Frontend**: auto-connect を Gateway 経由に変更
5. **テスト**: EKS上で `curl https://app.thirdlf03.com/api/raid/allocate` → allocate → 接続確認

## 注意点

- Agones allocator の gRPC API ではなく **Kubernetes API** (`GameServerAllocation` リソース作成) を使う方が簡単。gateway pod 内から `k8s.io/client-go` で in-cluster config を使えばOK
- cert-hash の fetch は gateway → game-server 間のクラスタ内通信（pod IP直接）で行う
- WebSocket フォールバック時も同じ allocate エンドポイントを使える
