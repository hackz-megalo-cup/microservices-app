import { useState } from "react";
import { useAdminTypeMatchups } from "../../hooks/use-admin-type-matchups";

const POKEMON_TYPES = [
  "normal",
  "fire",
  "water",
  "electric",
  "grass",
  "ice",
  "fighting",
  "poison",
  "ground",
  "flying",
  "psychic",
  "bug",
  "rock",
  "ghost",
  "dragon",
  "dark",
  "steel",
  "fairy",
];

export function TypeMatchupForm() {
  const { createMutation } = useAdminTypeMatchups();
  const [attackingType, setAttackingType] = useState("");
  const [defendingType, setDefendingType] = useState("");
  const [effectiveness, setEffectiveness] = useState("");

  function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    createMutation.mutate(
      {
        attackingType,
        defendingType,
        effectiveness: Number(effectiveness),
      },
      {
        onSuccess: () => {
          setAttackingType("");
          setDefendingType("");
          setEffectiveness("");
        },
      },
    );
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-wrap items-end gap-4">
      <div className="flex flex-col gap-1">
        <label className="text-xs text-text-secondary font-medium" htmlFor="attacking-type">
          攻撃タイプ
        </label>
        <select
          id="attacking-type"
          value={attackingType}
          onChange={(e) => setAttackingType(e.target.value)}
          required
          className="px-3 py-2 bg-bg-card border border-bg-hover rounded-xl text-text-primary text-sm focus:outline-none focus:border-accent"
        >
          <option value="">選択してください</option>
          {POKEMON_TYPES.map((t) => (
            <option key={t} value={t}>
              {t}
            </option>
          ))}
        </select>
      </div>

      <div className="flex flex-col gap-1">
        <label className="text-xs text-text-secondary font-medium" htmlFor="defending-type">
          防御タイプ
        </label>
        <select
          id="defending-type"
          value={defendingType}
          onChange={(e) => setDefendingType(e.target.value)}
          required
          className="px-3 py-2 bg-bg-card border border-bg-hover rounded-xl text-text-primary text-sm focus:outline-none focus:border-accent"
        >
          <option value="">選択してください</option>
          {POKEMON_TYPES.map((t) => (
            <option key={t} value={t}>
              {t}
            </option>
          ))}
        </select>
      </div>

      <div className="flex flex-col gap-1">
        <label className="text-xs text-text-secondary font-medium" htmlFor="effectiveness">
          倍率
        </label>
        <input
          id="effectiveness"
          type="number"
          step="0.25"
          value={effectiveness}
          onChange={(e) => setEffectiveness(e.target.value)}
          required
          placeholder="例: 2.0"
          className="px-3 py-2 bg-bg-card border border-bg-hover rounded-xl text-text-primary text-sm focus:outline-none focus:border-accent w-28"
        />
      </div>

      <button
        type="submit"
        disabled={createMutation.isPending}
        className="px-4 py-2 bg-accent text-bg-primary text-sm font-semibold rounded-xl hover:opacity-90 transition-opacity disabled:opacity-50"
      >
        作成
      </button>

      {createMutation.isError && (
        <p className="text-red-400 text-xs w-full">エラー: {createMutation.error.message}</p>
      )}
    </form>
  );
}
