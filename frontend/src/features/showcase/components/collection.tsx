import { useNavigate } from "react-router";
import "../../../styles/global.css";
import { useActivePokemon } from "../hooks/use-active-pokemon";
import { useCollectionPokemon } from "../hooks/use-collection-pokemon";
import { useSetActivePokemon } from "../hooks/use-set-active-pokemon";
import { NavBar } from "./ui/nav-bar";
import { TabBar } from "./ui/tab-bar";

export function Collection() {
  const navigate = useNavigate();
  const { pokemon, isLoading, error, refetch } = useCollectionPokemon();
  const { activePokemonId } = useActivePokemon();
  const setActiveMutation = useSetActivePokemon();
  const capturedCount = pokemon.filter((entry) => entry.captured).length;

  if (isLoading) {
    return (
      <div className="showcase-screen">
        <NavBar title="COLLECTION" rightIcon="search" />
        <div className="flex-1 flex items-center justify-center text-text-secondary text-sm">
          Loading collection...
        </div>
        <TabBar active="COLLECTION" variant="collection" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="showcase-screen">
        <NavBar title="COLLECTION" rightIcon="search" />
        <div className="flex-1 flex flex-col items-center justify-center gap-3 px-6">
          <p className="text-sm text-text-secondary m-0">マスターデータ取得失敗: {error.message}</p>
          <button
            type="button"
            className="bg-bg-card text-text-primary rounded-full px-4 py-2 border-none cursor-pointer"
            onClick={() => void refetch()}
          >
            Retry
          </button>
        </div>
        <TabBar active="COLLECTION" variant="collection" />
      </div>
    );
  }

  return (
    <div className="showcase-screen">
      <NavBar title="COLLECTION" rightIcon="search" />

      <div className="flex items-end gap-2 py-2 px-6">
        <span className="text-5xl font-bold leading-none text-text-primary">{capturedCount}</span>
        <span className="text-base text-text-secondary pb-1">/ {pokemon.length} captured</span>
      </div>

      <div className="grid grid-cols-2 gap-3 px-4 py-1 flex-1">
        {pokemon.map((entry) =>
          entry.captured ? (
            <div
              key={entry.id}
              className={`flex flex-col items-center justify-end gap-1 bg-bg-card rounded-2xl h-32 pb-3 overflow-hidden relative ${activePokemonId === entry.id ? "ring-2 ring-yellow-400" : ""}`}
            >
              {activePokemonId === entry.id && (
                <span className="absolute top-1 right-1 text-xs bg-yellow-400 text-black font-bold px-1 rounded z-10">
                  ✓
                </span>
              )}
              <button
                type="button"
                className="flex-1 flex flex-col items-center justify-end w-full overflow-hidden border-none bg-transparent cursor-pointer text-text-primary hover:bg-bg-hover"
                onClick={() => void navigate(`/collection/${entry.id}`)}
              >
                <div className="flex-1 flex items-center justify-center w-full overflow-hidden">
                  {entry.image ? (
                    <img src={entry.image} alt={entry.name} className="w-full h-16 object-cover" />
                  ) : null}
                </div>
                <span className="text-xs font-semibold">{entry.name}</span>
              </button>
              {activePokemonId !== entry.id && (
                <button
                  type="button"
                  className="text-xs bg-bg-hover text-text-secondary rounded px-2 py-0.5 border-none cursor-pointer hover:text-text-primary w-[calc(100%-16px)] mt-1"
                  onClick={() => void setActiveMutation.mutate(entry.id)}
                  disabled={setActiveMutation.isPending}
                >
                  {setActiveMutation.isPending && setActiveMutation.variables === entry.id
                    ? "..."
                    : "SELECT"}
                </button>
              )}
            </div>
          ) : (
            <div
              key={entry.id}
              className="flex flex-col items-center justify-center gap-1 bg-locked rounded-2xl h-32 pb-3 overflow-hidden border-none text-text-primary cursor-default hover:bg-locked"
            >
              <span className="text-2xl opacity-50">🔒</span>
              <span className="text-xs font-semibold">{entry.name}</span>
            </div>
          ),
        )}
      </div>

      <div className="flex-1" />

      <TabBar active="COLLECTION" variant="collection" />
    </div>
  );
}
