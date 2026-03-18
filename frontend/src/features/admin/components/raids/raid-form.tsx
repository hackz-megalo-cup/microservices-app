import { useQuery } from "@connectrpc/connect-query";
import { useState } from "react";
import { useNavigate } from "react-router";
import { listPokemon } from "../../../../gen/masterdata/v1/masterdata-MasterdataService_connectquery";
import { useAdminRaids } from "../../hooks/use-admin-raids";

export function RaidForm() {
  const navigate = useNavigate();
  const { createMutation } = useAdminRaids();
  const pokemonQuery = useQuery(listPokemon, {});
  const pokemon = pokemonQuery.data?.pokemon ?? [];

  const [bossPokemonId, setBossPokemonId] = useState("");

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!bossPokemonId) {
      return;
    }
    await createMutation.mutateAsync({ bossPokemonId });
    void navigate("/admin/raids");
  }

  return (
    <div className="p-8">
      <header className="mb-8">
        <h1 className="text-2xl font-bold text-text-primary">レイド新規作成</h1>
        <p className="text-sm text-text-secondary mt-1">新しいレイドロビーを作成</p>
      </header>

      <div className="bg-bg-card border border-bg-hover rounded-2xl p-6 max-w-lg">
        <form onSubmit={(e) => void handleSubmit(e)}>
          <div className="mb-6">
            <label
              htmlFor="boss-pokemon"
              className="block text-sm font-medium text-text-primary mb-2"
            >
              ボスポケモン
            </label>
            {pokemonQuery.isPending ? (
              <p className="text-text-secondary text-sm">読み込み中...</p>
            ) : (
              <select
                id="boss-pokemon"
                value={bossPokemonId}
                onChange={(e) => setBossPokemonId(e.target.value)}
                required
                className="w-full px-4 py-2 rounded-xl bg-bg-primary border border-bg-hover text-text-primary text-sm focus:outline-none focus:border-accent"
              >
                <option value="">選択してください</option>
                {pokemon.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name}
                  </option>
                ))}
              </select>
            )}
          </div>

          <div className="flex gap-3">
            <button
              type="submit"
              disabled={createMutation.isPending || !bossPokemonId}
              className="px-4 py-2 rounded-xl bg-accent text-bg-primary text-sm font-medium hover:opacity-90 transition-opacity disabled:opacity-50"
            >
              {createMutation.isPending ? "作成中..." : "作成"}
            </button>
            <button
              type="button"
              onClick={() => void navigate(-1)}
              className="px-4 py-2 rounded-xl bg-bg-hover text-text-secondary text-sm font-medium hover:text-text-primary transition-colors"
            >
              キャンセル
            </button>
          </div>

          {createMutation.error && (
            <p className="mt-4 text-sm text-red-400">エラーが発生しました</p>
          )}
        </form>
      </div>
    </div>
  );
}
