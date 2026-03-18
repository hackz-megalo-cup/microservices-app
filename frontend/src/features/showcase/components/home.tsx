import "../../../styles/global.css";
import { useActivePokemon } from "../hooks/use-active-pokemon";
import { useOpenRaids } from "../hooks/use-open-raids";
import { RaidCard } from "./ui/raid-card";
import { TabBar } from "./ui/tab-bar";

export function Home() {
  const { raids, isLoading, error } = useOpenRaids();
  const { activePokemon } = useActivePokemon();

  const pokemonName = activePokemon?.name ?? "---";
  const pokemonImage = activePokemon?.image ?? "/images/hero-python.png";
  const pokemonType = activePokemon?.types[0] ?? "";

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

      <section
        className="flex flex-col items-center justify-center gap-3 h-[280px]"
        style={{ background: "radial-gradient(circle, var(--color-accent-glow), transparent)" }}
      >
        <img
          src={pokemonImage}
          alt={pokemonName}
          className="w-[200px] h-[200px] rounded-full object-cover"
        />
        <h1 className="text-2xl font-bold text-text-primary m-0">{pokemonName}</h1>
        {pokemonType && <p className="text-sm text-text-secondary m-0">{pokemonType}</p>}
      </section>

      <section className="flex flex-col gap-3 px-6">
        <span className="text-xs font-bold tracking-widest text-text-secondary">ACTIVE RAIDS</span>
        {isLoading && <p className="text-sm text-text-secondary">Loading raids...</p>}
        {error && <p className="text-sm text-red-500">Error: {error.message}</p>}
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
    </div>
  );
}
