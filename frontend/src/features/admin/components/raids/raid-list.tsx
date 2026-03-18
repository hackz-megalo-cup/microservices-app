import { useNavigate } from "react-router";
import { useAdminRaids } from "../../hooks/use-admin-raids";

const STATUS_FILTERS = [
  { label: "全件", value: "" },
  { label: "waiting", value: "waiting" },
  { label: "in_battle", value: "in_battle" },
] as const;

function formatDate(ms: number | null): string {
  if (ms === null) {
    return "-";
  }
  return new Date(ms).toLocaleString("ja-JP");
}

export function RaidList() {
  const navigate = useNavigate();
  const { raids, isLoading, error, statusFilter, setStatusFilter, startMutation } = useAdminRaids();

  function handleStartBattle(lobbyId: string) {
    if (!window.confirm("バトルを開始しますか？")) {
      return;
    }
    startMutation.mutate({ lobbyId });
  }

  if (isLoading) {
    return (
      <div className="p-8">
        <p className="text-text-secondary">読み込み中...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-8">
        <p className="text-text-secondary">{error.message}</p>
      </div>
    );
  }

  return (
    <div className="p-8">
      <header className="mb-8 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">レイド管理</h1>
          <p className="text-sm text-text-secondary mt-1">レイドロビーの管理</p>
        </div>
        <button
          type="button"
          onClick={() => void navigate("/admin/raids/new")}
          className="px-4 py-2 rounded-xl bg-accent text-bg-primary text-sm font-medium hover:opacity-90 transition-opacity"
        >
          新規作成
        </button>
      </header>

      <div className="mb-4 flex gap-2">
        {STATUS_FILTERS.map((filter) => (
          <button
            key={filter.value}
            type="button"
            onClick={() => setStatusFilter(filter.value)}
            className={`px-4 py-2 rounded-xl text-sm font-medium transition-colors ${
              statusFilter === filter.value
                ? "bg-accent text-bg-primary"
                : "bg-bg-card text-text-secondary border border-bg-hover hover:bg-bg-hover hover:text-text-primary"
            }`}
          >
            {filter.label}
          </button>
        ))}
      </div>

      <div className="bg-bg-card border border-bg-hover rounded-2xl overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="border-b border-bg-hover">
              <th className="px-6 py-4 text-left text-xs font-semibold text-text-secondary tracking-wider">
                ID
              </th>
              <th className="px-6 py-4 text-left text-xs font-semibold text-text-secondary tracking-wider">
                ボスポケモン
              </th>
              <th className="px-6 py-4 text-left text-xs font-semibold text-text-secondary tracking-wider">
                参加
              </th>
              <th className="px-6 py-4 text-left text-xs font-semibold text-text-secondary tracking-wider">
                ステータス
              </th>
              <th className="px-6 py-4 text-left text-xs font-semibold text-text-secondary tracking-wider">
                作成日時
              </th>
              <th className="px-6 py-4 text-right text-xs font-semibold text-text-secondary tracking-wider">
                操作
              </th>
            </tr>
          </thead>
          <tbody>
            {raids.length === 0 && (
              <tr>
                <td colSpan={6} className="px-6 py-8 text-center text-sm text-text-secondary">
                  レイドがありません
                </td>
              </tr>
            )}
            {raids.map((raid) => (
              <tr
                key={raid.id}
                className="border-b border-bg-hover last:border-0 hover:bg-bg-hover transition-colors"
              >
                <td className="px-6 py-4 text-sm text-text-primary font-mono">
                  {raid.id.slice(0, 8)}
                </td>
                <td className="px-6 py-4 text-sm text-text-primary font-medium">{raid.bossName}</td>
                <td className="px-6 py-4 text-sm text-text-secondary">
                  {raid.currentParticipants}/{raid.maxParticipants}
                </td>
                <td className="px-6 py-4 text-sm text-text-secondary">{raid.status}</td>
                <td className="px-6 py-4 text-sm text-text-secondary">
                  {formatDate(raid.createdAtMs)}
                </td>
                <td className="px-6 py-4 text-right">
                  <button
                    type="button"
                    onClick={() => handleStartBattle(raid.id)}
                    disabled={startMutation.isPending}
                    className="px-3 py-1.5 rounded-lg bg-bg-hover text-text-secondary text-xs font-medium hover:text-text-primary transition-colors"
                  >
                    バトル開始
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
