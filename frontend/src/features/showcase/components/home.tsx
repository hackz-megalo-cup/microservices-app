import { useState } from "react";
import "../../../styles/global.css";
import { useAuthContext } from "../../../lib/auth";
import { useActivePokemon } from "../hooks/use-active-pokemon";
import { useLobbyOverview } from "../hooks/use-lobby-overview";
import { useOpenRaids } from "../hooks/use-open-raids";
import { getPokemonImageUrl } from "../api/pokemon";
import { RaidCard } from "./ui/raid-card";
import { TabBar } from "./ui/tab-bar";

export function Home() {
  const { user } = useAuthContext();
  const userId = user?.id ?? "";

  const { raids, isLoading: raidsLoading, error: raidsError } = useOpenRaids();
  const {
    items,
    pokedex,
    caughtCount,
    totalPokemonCount,
    isLoading: overviewLoading,
    error: overviewError,
  } = useLobbyOverview(userId);
  const {
    activePokemon,
    isLoading: activePokemonLoading,
    setActivePokemon,
    isSettingPokemon,
  } = useActivePokemon(userId);

  const [showPokemonSelector, setShowPokemonSelector] = useState(false);

  const caughtPokemon = pokedex.filter((entry) => entry.caught);
  const totalItems = items.reduce((sum, item) => sum + item.quantity, 0);

  const heroImage = activePokemon?.image ?? "/images/collection-placeholder.png";
  const heroName = activePokemon?.name ?? "???";
  const heroType = activePokemon?.types?.[0] ?? "";

  return (
    <div className="showcase-screen">
      <header className="flex items-center justify-between px-6 py-3">
        <span className="text-xs font-bold tracking-widest text-text-secondary">POKÉMON</span>
        <button
          type="button"
          className="flex items-center justify-center w-11 h-11 bg-transparent text-xl cursor-pointer rounded-lg hover:bg-bg-hover"
          aria-label="Notifications"
        >
          🔔
        </button>
      </header>

      {/* Hero Section: Active Pokemon */}
      <section
        className="flex flex-col items-center justify-center gap-3 h-[280px] relative"
        style={{ background: "radial-gradient(circle, var(--color-accent-glow), transparent)" }}
      >
        {activePokemonLoading ? (
          <div className="w-[200px] h-[200px] rounded-full bg-bg-hover animate-pulse" />
        ) : (
          <img
            src={heroImage}
            alt={heroName}
            className="w-[200px] h-[200px] rounded-full object-cover"
          />
        )}
        <h1 className="text-2xl font-bold text-text-primary m-0">{heroName}</h1>
        {heroType && <p className="text-sm text-text-secondary m-0">{heroType}</p>}
        <button
          type="button"
          className="text-xs font-bold bg-bg-card text-text-secondary px-4 py-2 rounded-full border-none cursor-pointer hover:bg-bg-hover"
          onClick={() => setShowPokemonSelector(true)}
          disabled={isSettingPokemon}
        >
          {isSettingPokemon ? "変更中..." : "ポケモン変更"}
        </button>
      </section>

      {/* Stats Section: Lobby Overview */}
      <section className="px-6 py-3">
        {overviewError && (
          <p className="text-xs text-red-500 mb-2">概要取得失敗: {overviewError.message}</p>
        )}
        <div className="grid grid-cols-3 gap-3">
          <div className="bg-bg-card rounded-2xl p-3 flex flex-col items-center gap-1">
            <span className="text-xl">🎒</span>
            {overviewLoading ? (
              <div className="h-5 w-8 bg-bg-hover rounded animate-pulse" />
            ) : (
              <span className="text-lg font-bold text-text-primary">{totalItems}</span>
            )}
            <span className="text-xs text-text-secondary">Items</span>
          </div>
          <div className="bg-bg-card rounded-2xl p-3 flex flex-col items-center gap-1">
            <span className="text-xl">📖</span>
            {overviewLoading ? (
              <div className="h-5 w-12 bg-bg-hover rounded animate-pulse" />
            ) : (
              <span className="text-lg font-bold text-text-primary">
                {caughtCount}/{totalPokemonCount}
              </span>
            )}
            <span className="text-xs text-text-secondary">Pokédex</span>
          </div>
          <div className="bg-bg-card rounded-2xl p-3 flex flex-col items-center gap-1">
            <span className="text-xl">⚔️</span>
            {raidsLoading ? (
              <div className="h-5 w-6 bg-bg-hover rounded animate-pulse" />
            ) : (
              <span className="text-lg font-bold text-text-primary">{raids.length}</span>
            )}
            <span className="text-xs text-text-secondary">Raids</span>
          </div>
        </div>
      </section>

      {/* Active Raids Section */}
      <section className="flex flex-col gap-3 px-6">
        <span className="text-xs font-bold tracking-widest text-text-secondary">ACTIVE RAIDS</span>
        {raidsLoading && <p className="text-sm text-text-secondary">Loading raids...</p>}
        {raidsError && <p className="text-sm text-red-500">Error: {raidsError.message}</p>}
        {raids.map((raid) => (
          <RaidCard
            key={raid.id}
            id={raid.id}
            name={raid.name}
            type={raid.type}
            players={raid.players}
            timer={raid.timer}
            image={raid.image}
          />
        ))}
      </section>

      <div className="flex-1" />

      <TabBar active="HOME" />

      {/* Pokemon Selector Sheet */}
      {showPokemonSelector && (
        <div
          className="fixed inset-0 bg-black/60 z-50 flex items-end"
          onClick={() => setShowPokemonSelector(false)}
          onKeyDown={(e) => e.key === "Escape" && setShowPokemonSelector(false)}
          role="dialog"
          aria-modal="true"
          aria-label="ポケモン選択"
        >
          <div
            className="w-full bg-bg-primary rounded-t-3xl p-6 max-h-[70vh] flex flex-col gap-4"
            onClick={(e) => e.stopPropagation()}
            onKeyDown={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between">
              <span className="text-sm font-bold tracking-widest text-text-secondary">
                ポケモン選択
              </span>
              <button
                type="button"
                className="text-text-secondary text-xl bg-transparent border-none cursor-pointer"
                onClick={() => setShowPokemonSelector(false)}
              >
                ✕
              </button>
            </div>
            {overviewLoading ? (
              <p className="text-sm text-text-secondary text-center py-4">Loading...</p>
            ) : caughtPokemon.length === 0 ? (
              <p className="text-sm text-text-secondary text-center py-4">
                捕まえたポケモンがいません
              </p>
            ) : (
              <div className="grid grid-cols-3 gap-3 overflow-y-auto">
                {caughtPokemon.map((entry) => (
                  <button
                    key={entry.pokemonId}
                    type="button"
                    className="flex flex-col items-center gap-1 bg-bg-card rounded-2xl p-3 border-none cursor-pointer hover:bg-bg-hover disabled:opacity-50"
                    disabled={isSettingPokemon}
                    onClick={() => {
                      setActivePokemon(entry.pokemonId);
                      setShowPokemonSelector(false);
                    }}
                  >
                    <img
                      src={getPokemonImageUrl({ name: entry.pokemonName })}
                      alt={entry.pokemonName}
                      className="w-14 h-14 object-cover"
                    />
                    <span className="text-xs font-semibold text-text-primary text-center leading-tight">
                      {entry.pokemonName}
                    </span>
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
