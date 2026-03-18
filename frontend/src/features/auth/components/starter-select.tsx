import { useState } from "react";
import { useNavigate } from "react-router";
import { getPokemonImageUrl } from "../../../lib/pokemon-image";
import { useAuthContext } from "../hooks/use-auth-context";
import { useStarterSelect } from "../hooks/use-starter-select";
import "../../../styles/global.css";

const STARTERS = [
  {
    id: "00000000-0000-0000-0000-000000000001",
    name: "Go",
    type: "procedural",
    hp: 85,
    attack: 70,
    speed: 90,
    move: "ゴルーチン乱舞",
  },
  {
    id: "00000000-0000-0000-0000-000000000002",
    name: "Python",
    type: "dynamic",
    hp: 90,
    attack: 65,
    speed: 60,
    move: "インデント地獄",
  },
  {
    id: "00000000-0000-0000-0000-000000000009",
    name: "Whitespace",
    type: "functional",
    hp: 50,
    attack: 50,
    speed: 100,
    move: "虚空の一撃",
  },
] as const;

export function StarterSelect() {
  const { user } = useAuthContext();
  const userId = user?.id ?? "";
  const navigate = useNavigate();
  const { selectStarter, isPending, error } = useStarterSelect(userId);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const handleConfirm = async () => {
    if (!selectedId) {
      return;
    }
    try {
      await selectStarter(selectedId);
      navigate("/", { replace: true });
    } catch {
      // error is exposed via hook
    }
  };

  return (
    <div className="showcase-screen items-center justify-center gap-6 px-6 py-8">
      <div className="flex flex-col items-center gap-2">
        <span className="text-4xl">🎉</span>
        <h1 className="text-2xl font-bold text-text-primary m-0">パートナーを選ぼう！</h1>
        <p className="text-sm text-text-secondary m-0 text-center">最初のポケモンを1匹選んでね</p>
      </div>

      <div className="flex flex-col gap-3 w-full max-w-sm">
        {STARTERS.map((pokemon) => (
          <button
            key={pokemon.id}
            type="button"
            onClick={() => setSelectedId(pokemon.id)}
            disabled={isPending}
            className={`flex items-center gap-4 p-4 rounded-2xl border-2 cursor-pointer transition-all ${
              selectedId === pokemon.id
                ? "border-accent bg-accent/10"
                : "border-transparent bg-bg-card hover:bg-bg-hover"
            } disabled:opacity-50`}
          >
            <img
              src={getPokemonImageUrl({ name: pokemon.name })}
              alt={pokemon.name}
              className="w-16 h-16 rounded-full object-cover"
            />
            <div className="flex flex-col items-start gap-1 flex-1">
              <span className="text-lg font-bold text-text-primary">{pokemon.name}</span>
              <span className="text-xs text-text-secondary">{pokemon.type}</span>
              <div className="flex gap-3 text-xs text-text-secondary">
                <span>HP {pokemon.hp}</span>
                <span>ATK {pokemon.attack}</span>
                <span>SPD {pokemon.speed}</span>
              </div>
              <span className="text-xs text-accent">⚡ {pokemon.move}</span>
            </div>
            {selectedId === pokemon.id && <span className="text-2xl">✓</span>}
          </button>
        ))}
      </div>

      {error && <p className="text-red-400 text-sm m-0">{error.message}</p>}

      <button
        type="button"
        onClick={() => void handleConfirm()}
        disabled={!selectedId || isPending}
        className="w-full max-w-sm h-14 bg-accent text-bg-primary font-bold text-lg rounded-3xl cursor-pointer hover:opacity-90 transition-opacity disabled:opacity-50"
      >
        {isPending ? "準備中..." : "この子にする！"}
      </button>
    </div>
  );
}
