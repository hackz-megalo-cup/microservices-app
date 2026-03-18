import { useParams } from "react-router";
import type { Pokemon } from "../types";
import "../styles/global.css";
import { usePokemonDetail } from "../hooks/use-pokemon-detail";
import { NavBar } from "./ui/nav-bar";

export function Detail() {
  const { id } = useParams<{ id: string }>();
  const pokemonId = id ?? "";
  const { pokemon, isLoading, error, refetch } = usePokemonDetail(pokemonId);

  if (!pokemonId) {
    return (
      <div className="showcase-screen">
        <NavBar title="NOT FOUND" />
        <p className="text-center p-6">Pokemon not found</p>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="showcase-screen">
        <NavBar title="DETAIL" />
        <div className="flex-1 flex items-center justify-center text-text-secondary text-sm">
          Loading pokemon detail...
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="showcase-screen">
        <NavBar title="DETAIL" />
        <div className="flex-1 flex flex-col items-center justify-center gap-3 px-6">
          <p className="text-sm text-text-secondary m-0">ポケモン詳細取得失敗: {error.message}</p>
          <button
            type="button"
            className="bg-bg-card text-text-primary rounded-full px-4 py-2 border-none cursor-pointer"
            onClick={() => void refetch()}
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  if (!pokemon) {
    return (
      <div className="showcase-screen">
        <NavBar title="NOT FOUND" />
        <p className="text-center p-6">Pokemon not found</p>
      </div>
    );
  }

  return (
    <div className="showcase-screen">
      <NavBar title={pokemon.number} rightIcon="heart" />

      <div
        className="flex items-center justify-center h-[200px]"
        style={{ background: "radial-gradient(circle, var(--color-accent-glow), transparent)" }}
      >
        {pokemon.image ? (
          <img
            src={pokemon.image}
            alt={pokemon.name}
            className="w-[200px] h-[200px] rounded-full object-cover"
          />
        ) : (
          <div className="w-[200px] h-[200px] rounded-full bg-bg-card" />
        )}
      </div>

      <div className="flex flex-col items-center gap-3 px-6">
        <h1 className="text-2xl font-bold m-0">{pokemon.name}</h1>
        <div className="flex gap-2">
          {pokemon.types.map((type) => (
            <span
              key={type}
              className="text-xs text-text-secondary bg-bg-card px-4 py-2 rounded-full"
            >
              {type}
            </span>
          ))}
        </div>
      </div>

      <div className="flex gap-3 pt-4 px-6">
        {pokemon.stats.map((stat) => (
          <div
            key={stat.label}
            className="flex-1 flex flex-col items-center gap-1 bg-bg-card rounded-2xl p-4"
          >
            <span className="text-2xl font-bold text-accent">{stat.value}</span>
            <span className="text-xs font-semibold tracking-wide text-text-secondary">
              {stat.label}
            </span>
          </div>
        ))}
      </div>

      <div className="flex flex-col gap-3 px-6">
        <div className="flex flex-col gap-2 bg-bg-card rounded-2xl p-4">
          <span className="text-sm font-bold">About</span>
          <p className="text-xs text-text-secondary leading-relaxed m-0">{pokemon.about}</p>
        </div>

        <div className="flex flex-col gap-2">
          <span className="text-sm font-bold">Moves</span>
          {pokemon.moves.map((move) => (
            <div
              key={move.name}
              className="flex justify-between items-center bg-bg-card rounded-2xl px-4 py-3"
            >
              <span className="text-sm font-semibold">{move.name}</span>
              <span className="text-xs font-bold text-accent bg-bg-primary px-3 py-1 rounded-full">
                PWR {move.power}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
