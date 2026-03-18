import { useNavigate } from "react-router";
import { useAdminPokemon } from "../../hooks/use-admin-pokemon";

export function PokemonList() {
  const navigate = useNavigate();
  const { pokemon, isLoading, error, deleteMutation } = useAdminPokemon();

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
        <p className="text-red-400">エラー: {error.message}</p>
      </div>
    );
  }

  return (
    <div className="p-8">
      <header className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">Pokemon</h1>
          <p className="text-sm text-text-secondary mt-1">Pokemon マスターデータの管理</p>
        </div>
        <button
          type="button"
          onClick={() => void navigate("/admin/pokemon/new")}
          className="px-4 py-2 bg-accent text-bg-primary text-sm font-semibold rounded-xl hover:opacity-90 transition-opacity"
        >
          新規作成
        </button>
      </header>

      <div className="bg-bg-card border border-bg-hover rounded-2xl overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-bg-hover">
              <th className="px-4 py-3 text-left text-text-secondary font-medium">名前</th>
              <th className="px-4 py-3 text-left text-text-secondary font-medium">タイプ</th>
              <th className="px-4 py-3 text-right text-text-secondary font-medium">HP</th>
              <th className="px-4 py-3 text-right text-text-secondary font-medium">Attack</th>
              <th className="px-4 py-3 text-right text-text-secondary font-medium">Speed</th>
              <th className="px-4 py-3 text-left text-text-secondary font-medium">特殊技名</th>
              <th className="px-4 py-3 text-right text-text-secondary font-medium">特殊技威力</th>
              <th className="px-4 py-3 text-right text-text-secondary font-medium">操作</th>
            </tr>
          </thead>
          <tbody>
            {pokemon.length === 0 ? (
              <tr>
                <td colSpan={8} className="px-4 py-8 text-center text-text-secondary">
                  データがありません
                </td>
              </tr>
            ) : (
              pokemon.map((p) => (
                <tr
                  key={p.id}
                  className="border-b border-bg-hover last:border-0 hover:bg-bg-hover transition-colors"
                >
                  <td className="px-4 py-3 text-text-primary font-medium">{p.name}</td>
                  <td className="px-4 py-3 text-text-secondary">{p.type}</td>
                  <td className="px-4 py-3 text-right text-text-secondary">{p.hp}</td>
                  <td className="px-4 py-3 text-right text-text-secondary">{p.attack}</td>
                  <td className="px-4 py-3 text-right text-text-secondary">{p.speed}</td>
                  <td className="px-4 py-3 text-text-secondary">{p.specialMoveName}</td>
                  <td className="px-4 py-3 text-right text-text-secondary">
                    {p.specialMoveDamage}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <div className="flex items-center justify-end gap-2">
                      <button
                        type="button"
                        onClick={() => void navigate(`/admin/pokemon/${p.id}/edit`)}
                        className="px-3 py-1 text-xs font-medium text-text-primary bg-bg-hover rounded-lg hover:bg-bg-primary transition-colors"
                      >
                        編集
                      </button>
                      <button
                        type="button"
                        onClick={() => handleDelete(p.id, p.name)}
                        disabled={deleteMutation.isPending}
                        className="px-3 py-1 text-xs font-medium text-red-400 bg-bg-hover rounded-lg hover:bg-bg-primary transition-colors disabled:opacity-50"
                      >
                        削除
                      </button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
