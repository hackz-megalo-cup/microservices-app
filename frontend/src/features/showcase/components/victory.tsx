import { useQuery } from "@connectrpc/connect-query";
import { useMemo } from "react";
import { useLocation, useNavigate, useParams } from "react-router";
import { listPokemon } from "../../../gen/masterdata/v1/masterdata-MasterdataService_connectquery";
import { listOpenRaids } from "../../../gen/raid_lobby/v1/raid_lobby-RaidLobbyService_connectquery";
import { getPokemonImageUrl } from "../../../lib/pokemon-image";
import "../../../styles/global.css";
import type { VictoryRouteState } from "../../battle/types";

const rewards = [
  { value: "+350", label: "EXP" },
  { value: "x2", label: "Red Bull" },
  { value: "x1", label: "Cushion" },
] as const;

function formatElapsed(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}:${String(s).padStart(2, "0")}`;
}

export function Victory() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const victoryState = (location.state as VictoryRouteState | null) ?? null;
  const elapsed = victoryState?.elapsed;
  const battleSessionId = victoryState?.battleSessionId;

  // ボス情報を動的に取得
  const openRaidsQuery = useQuery(listOpenRaids, { statusFilter: "" });
  const pokemonQuery = useQuery(listPokemon, {});
  const bossInfo = useMemo(() => {
    const raid = openRaidsQuery.data?.raids.find((r) => r.id === id);
    if (!raid || !pokemonQuery.data) {
      return null;
    }
    const boss = pokemonQuery.data.pokemon.find((p) => p.id === raid.bossPokemonId);
    if (!boss) {
      return null;
    }
    return { name: boss.name, image: getPokemonImageUrl({ name: boss.name }) };
  }, [openRaidsQuery.data, pokemonQuery.data, id]);

  const bossName = victoryState?.bossName ?? bossInfo?.name ?? "Boss";
  const bossImage =
    (victoryState?.bossName ? getPokemonImageUrl({ name: victoryState.bossName }) : null) ??
    bossInfo?.image ??
    "/images/collection-python.png";
  const elapsedDisplay = elapsed !== undefined ? formatElapsed(elapsed) : "?:??";

  return (
    <div className="showcase-screen items-center gap-8 pt-12 px-6 pb-6">
      <img
        src={bossImage}
        alt={bossName}
        className="w-[120px] h-[120px] rounded-full object-cover"
      />
      <span className="text-xs font-bold tracking-widest text-accent">RAID CLEAR</span>
      <h1 className="text-4xl font-bold text-text-primary m-0">Victory</h1>
      <p className="text-sm text-text-secondary m-0">
        {bossName} defeated in {elapsedDisplay}
      </p>

      <div className="flex flex-col gap-4 bg-bg-card rounded-2xl p-5 w-full">
        <span className="text-sm font-bold text-text-primary">Rewards</span>
        <div className="flex gap-3">
          {rewards.map((reward) => (
            <div
              key={reward.label}
              className="flex-1 flex flex-col items-center gap-1 bg-bg-primary rounded-lg p-4"
            >
              <span className="text-xl font-bold text-accent">{reward.value}</span>
              <span className="text-xs text-text-secondary">{reward.label}</span>
            </div>
          ))}
        </div>
      </div>

      <button
        type="button"
        className="w-full h-14 bg-accent rounded-3xl text-base font-bold text-bg-primary cursor-pointer border-none hover:opacity-90"
        onClick={() => void navigate(`/capture/${battleSessionId ?? id}`)}
      >
        CAPTURE
      </button>
    </div>
  );
}
