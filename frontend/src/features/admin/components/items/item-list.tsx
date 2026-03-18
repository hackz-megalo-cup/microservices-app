import { useNavigate } from "react-router";
import { useAdminItems } from "../../hooks/use-admin-items";

export function ItemList() {
  const navigate = useNavigate();
  const { items, isLoading, error, deleteMutation } = useAdminItems();

  function handleDelete(id: string, name: string) {
    if (!window.confirm(`「${name}」を削除しますか？`)) {
      return;
    }
    deleteMutation.mutate({ id });
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
        <p className="text-text-secondary">エラーが発生しました</p>
      </div>
    );
  }

  return (
    <div className="p-8">
      <header className="mb-8 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">アイテム管理</h1>
          <p className="text-sm text-text-secondary mt-1">アイテムのマスターデータを管理</p>
        </div>
        <button
          type="button"
          onClick={() => void navigate("/admin/items/new")}
          className="px-4 py-2 rounded-xl bg-accent text-bg-primary text-sm font-medium hover:opacity-90 transition-opacity"
        >
          新規作成
        </button>
      </header>

      <div className="bg-bg-card border border-bg-hover rounded-2xl overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="border-b border-bg-hover">
              <th className="px-6 py-4 text-left text-xs font-semibold text-text-secondary tracking-wider">
                名前
              </th>
              <th className="px-6 py-4 text-left text-xs font-semibold text-text-secondary tracking-wider">
                エフェクト数
              </th>
              <th className="px-6 py-4 text-right text-xs font-semibold text-text-secondary tracking-wider">
                操作
              </th>
            </tr>
          </thead>
          <tbody>
            {items.length === 0 && (
              <tr>
                <td colSpan={3} className="px-6 py-8 text-center text-sm text-text-secondary">
                  アイテムがありません
                </td>
              </tr>
            )}
            {items.map((item) => (
              <tr
                key={item.id}
                className="border-b border-bg-hover last:border-0 hover:bg-bg-hover transition-colors"
              >
                <td className="px-6 py-4 text-sm text-text-primary font-medium">{item.name}</td>
                <td className="px-6 py-4 text-sm text-text-secondary">{item.effects.length}</td>
                <td className="px-6 py-4 text-right">
                  <div className="flex items-center justify-end gap-2">
                    <button
                      type="button"
                      onClick={() => void navigate(`/admin/items/${item.id}/edit`)}
                      className="px-3 py-1.5 rounded-lg bg-bg-hover text-text-secondary text-xs font-medium hover:text-text-primary transition-colors"
                    >
                      編集
                    </button>
                    <button
                      type="button"
                      onClick={() => handleDelete(item.id, item.name)}
                      className="px-3 py-1.5 rounded-lg bg-bg-hover text-text-secondary text-xs font-medium hover:text-text-primary transition-colors"
                      disabled={deleteMutation.isPending}
                    >
                      削除
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
