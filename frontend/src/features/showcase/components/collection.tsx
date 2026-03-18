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
  const { activeId } = useActivePokemon();
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
              className={
                entry.id === activeId
                  ? "flex flex-col items-center justify-end gap-1 rounded-2xl h-32 pb-3 overflow-hidden relative bg-bg-card border-2 border-yellow-400"
                  : "flex flex-col items-center justify-end gap-1 rounded-2xl h-32 pb-3 overflow-hidden relative bg-bg-card border-2 border-transparent"
              }
            >
              {entry.id === activeId && (
                <span className="absolute top-1 right-2 text-yellow-400 text-xs font-bold">✓</span>
              )}
              <button
                type="button"
                className="flex-1 flex flex-col items-center justify-end w-full cursor-pointer bg-transparent border-none text-text-primary hover:bg-bg-hover rounded-t-2xl"
                onClick={() => void navigate(`/collection/${entry.id}`)}
              >
                <div className="flex-1 flex items-center justify-center w-full overflow-hidden">
                  {entry.image ? (
                    <img src={entry.image} alt={entry.name} className="w-full h-16 object-cover" />
                  ) : null}
                </div>
                <span className="text-xs font-semibold">{entry.name}</span>
              </button>
              {entry.id !== activeId && (
                <button
                  type="button"
                  className="text-xs bg-accent text-white rounded-full px-3 py-0.5 border-none cursor-pointer mt-1 disabled:opacity-50"
                  disabled={setActiveMutation.isPending}
                  onClick={() => void setActiveMutation.mutate(entry.id)}
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
