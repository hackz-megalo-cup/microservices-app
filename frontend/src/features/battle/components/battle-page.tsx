import { useQuery } from "@connectrpc/connect-query";
import type { MouseEvent as ReactMouseEvent } from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { listPokemon } from "../../../gen/masterdata/v1/masterdata-MasterdataService_connectquery";
import { listOpenRaids } from "../../../gen/raid_lobby/v1/raid_lobby-RaidLobbyService_connectquery";
import "../../../styles/global.css";
import { useAuthContext } from "../../../lib/auth";
import { useActivePokemon } from "../../showcase/hooks/use-active-pokemon";
import { useGameConnection } from "../hooks/use-game-connection";
import type { ServerMessage } from "../types";
import { preloadRaidBossModel, RaidBossModel, useRaidBossModel } from "./raid-boss-model";
import "./battle-page.css";

interface FloatingDmg {
  id: number;
  value: number;
  x: number;
  y: number;
  isSpecial: boolean;
}

interface Ripple {
  id: number;
  x: number;
  y: number;
}

export function BattlePage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { user } = useAuthContext();
  const userId = useMemo(() => user?.id ?? crypto.randomUUID(), [user?.id]);

  // --- Battle state ---
  const [bossHp, setBossHp] = useState(0);
  const [bossMaxHp, setBossMaxHp] = useState(0);
  const [tapCount, setTapCount] = useState(0);
  const [result, setResult] = useState<string | null>(null);
  const [timeoutSec, setTimeoutSec] = useState(300);
  const [floatingDmgs, setFloatingDmgs] = useState<FloatingDmg[]>([]);
  const [ripples, setRipples] = useState<Ripple[]>([]);
  const [participants, setParticipants] = useState<string[]>([]);
  const [squashing, setSquashing] = useState(false);
  const hitCount = useRef(0);
  const squashTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const dmgSeq = useRef(0);
  const rippleSeq = useRef(0);
  const requiredForSpecial = 10;

  // ボス情報を取得
  const openRaidsQuery = useQuery(listOpenRaids, { statusFilter: "" });
  const pokemonQuery = useQuery(listPokemon, {});
  const bossName = useMemo(() => {
    const raid = openRaidsQuery.data?.raids.find((r) => r.id === id);
    if (!raid || !pokemonQuery.data) {
      return undefined;
    }
    const boss = pokemonQuery.data.pokemon.find((p) => p.id === raid.bossPokemonId);
    return boss?.name;
  }, [openRaidsQuery.data, pokemonQuery.data, id]);

  const model = useRaidBossModel(bossName);

  // ボス名が確定したら該当モデルだけプリロード
  useEffect(() => {
    preloadRaidBossModel(bossName);
  }, [bossName]);

  // アクティブポケモンのステータスを取得
  const { activePokemon } = useActivePokemon(userId);
  const pokemonStats = useMemo(() => {
    if (!activePokemon) {
      return undefined;
    }
    const attack = activePokemon.stats.find((s) => s.label === "ATK")?.value ?? 100;
    const speed = activePokemon.stats.find((s) => s.label === "SPD")?.value ?? 50;
    const type = activePokemon.types[0] ?? "normal";
    const specialMoveName = activePokemon.moves[0]?.name ?? "Tackle";
    const specialMoveDamage = activePokemon.moves[0]?.power ?? 500;
    return {
      pokemonAttack: attack,
      pokemonSpeed: speed,
      pokemonType: type,
      specialMoveName,
      specialMoveDamage,
    };
  }, [activePokemon]);

  const spawnDmg = useCallback((value: number, isSpecial: boolean) => {
    const x = 10 + Math.random() * 60;
    const y = 5 + Math.random() * 50;
    dmgSeq.current++;
    const entryId = dmgSeq.current;
    const entry: FloatingDmg = { id: entryId, value, x, y, isSpecial };
    setFloatingDmgs((prev) => [...prev.slice(-8), entry]);
    setTimeout(() => {
      setFloatingDmgs((prev) => prev.filter((d) => d.id !== entryId));
    }, 800);
  }, []);

  const spawnRipple = useCallback((clientX: number, clientY: number, rect: DOMRect) => {
    const x = clientX - rect.left;
    const y = clientY - rect.top;
    rippleSeq.current++;
    const entryId = rippleSeq.current;
    const entry: Ripple = { id: entryId, x, y };
    setRipples((prev) => [...prev.slice(-4), entry]);
    setTimeout(() => {
      setRipples((prev) => prev.filter((r) => r.id !== entryId));
    }, 500);
  }, []);

  const onMessage = useCallback(
    (msg: ServerMessage) => {
      switch (msg.t) {
        case "joined":
          setBossHp(msg.bossHp);
          setBossMaxHp(msg.bossMaxHp);
          if (msg.timeoutSec !== undefined) {
            setTimeoutSec(msg.timeoutSec);
          }
          if (msg.participants) {
            setParticipants(msg.participants);
          }
          break;
        case "hp":
          setBossHp(msg.hp);
          spawnDmg(msg.lastDmg, false);
          hitCount.current++;
          if (hitCount.current % 3 === 0) {
            setSquashing(true);
            if (squashTimer.current) {
              clearTimeout(squashTimer.current);
            }
            squashTimer.current = setTimeout(() => setSquashing(false), 100);
          }
          break;
        case "special_used":
          setBossHp(msg.bossHp);
          spawnDmg(msg.dmg, true);
          setSquashing(true);
          if (squashTimer.current) {
            clearTimeout(squashTimer.current);
          }
          squashTimer.current = setTimeout(() => setSquashing(false), 150);
          break;
        case "finished":
          setResult(msg.result);
          setTimeout(() => navigate(`/victory/${id}`, { state: { elapsed: msg.elapsed } }), 2000);
          break;
        case "time_sync":
          setTimeoutSec(msg.remainingSec);
          break;
      }
    },
    [id, navigate, spawnDmg],
  );

  useEffect(() => {
    if (result !== null) {
      return;
    }
    const timer = setInterval(() => {
      setTimeoutSec((prev) => Math.max(0, prev - 1));
    }, 1000);
    return () => clearInterval(timer);
  }, [result]);

  const { status, sendTap, sendSpecial } = useGameConnection({
    userId,
    lobbyId: id,
    pokemonStats,
    onMessage,
  });

  const handleTap = (e: ReactMouseEvent<HTMLButtonElement>) => {
    sendTap();
    setTapCount((c) => c + 1);
    const rect = e.currentTarget.getBoundingClientRect();
    spawnRipple(e.clientX, e.clientY, rect);
  };

  const handleSpecial = () => {
    sendSpecial();
    setTapCount(0);
  };

  const hpPercent = bossMaxHp > 0 ? (bossHp / bossMaxHp) * 100 : 0;
  const isConnected = status === "connected";
  const canSpecial = isConnected && tapCount >= requiredForSpecial;

  const minutes = Math.floor(timeoutSec / 60);
  const seconds = timeoutSec % 60;
  const timerDisplay = `${minutes}:${String(seconds).padStart(2, "0")}`;

  return (
    <div
      className="showcase-screen"
      style={{
        backgroundImage: `url(${model.bg})`,
        backgroundSize: "cover",
        backgroundPosition: "center",
      }}
    >
      {/* Boss info + HP bar */}
      <section className="flex flex-col gap-3 px-6 pt-4">
        <div className="flex items-center justify-center gap-3">
          <span className="text-lg font-bold text-text-primary">RAID BOSS</span>
          <span className="text-xs font-bold text-accent bg-accent-glow px-3 py-1 rounded-lg">
            {timerDisplay}
          </span>
        </div>

        {status !== "connected" && (
          <div className="flex items-center justify-center gap-2">
            <span className="text-xs text-text-secondary">
              {status === "connecting"
                ? "Connecting..."
                : status === "error"
                  ? "Connection error"
                  : "Disconnected"}
            </span>
          </div>
        )}

        <div className="w-full h-3 bg-bg-card rounded-full overflow-hidden">
          <div
            className="h-full rounded-full transition-all duration-200"
            style={{
              width: `${hpPercent}%`,
              background: "linear-gradient(90deg, var(--color-accent), var(--color-accent-dark))",
            }}
          />
        </div>
        <p className="text-right text-sm font-mono text-text-secondary">
          {bossHp.toLocaleString()} / {bossMaxHp.toLocaleString()}
        </p>
      </section>

      {/* Tap area */}
      <button
        type="button"
        onClick={handleTap}
        disabled={!isConnected || result !== null}
        className="flex-1 relative flex items-center justify-center cursor-pointer select-none overflow-hidden disabled:cursor-not-allowed"
      >
        <div className="w-[280px] h-[280px] rounded-2xl pointer-events-none overflow-hidden">
          <RaidBossModel squashing={squashing} model={model} />
        </div>

        {ripples.map((r) => (
          <span
            key={r.id}
            className="absolute pointer-events-none rounded-full"
            style={{
              left: r.x,
              top: r.y,
              width: 120,
              height: 120,
              border: "2px solid var(--color-accent)",
              background: "radial-gradient(circle, var(--color-accent-glow) 0%, transparent 70%)",
              animation: "tap-ripple 0.5s ease-out forwards",
            }}
          />
        ))}

        {floatingDmgs.map((dmg) => (
          <span
            key={dmg.id}
            className="absolute pointer-events-none font-bold font-mono"
            style={{
              left: `${dmg.x}%`,
              top: `${dmg.y}%`,
              color: dmg.isSpecial ? "var(--color-green)" : "var(--color-accent)",
              fontSize: dmg.isSpecial ? "2rem" : "1.5rem",
              textShadow: "0 2px 8px rgba(0,0,0,0.7)",
              animation: dmg.isSpecial
                ? "dmg-float-special 0.8s ease-out forwards"
                : "dmg-float 0.8s ease-out forwards",
            }}
          >
            -{dmg.value}
          </span>
        ))}

        {result && (
          <div className="absolute inset-0 flex items-center justify-center bg-bg-primary/80 rounded-2xl">
            <span className="text-4xl font-bold text-accent">
              {result === "win" ? "VICTORY!" : "TIME UP"}
            </span>
          </div>
        )}
      </button>

      {/* Bottom controls */}
      <section className="flex flex-col items-center gap-3 px-6 pb-6">
        {/* Participant indicators */}
        <div className="flex gap-3">
          {participants.length > 0
            ? participants.map((pid) => (
                <div
                  key={pid}
                  className="w-8 h-8 rounded-full bg-bg-card border-2 border-accent"
                  title={pid}
                />
              ))
            : Array.from({ length: 4 }).map((_, i) => (
                <div
                  key={`placeholder-${String(i)}`}
                  className="w-8 h-8 rounded-full bg-bg-card border-2 border-text-secondary"
                />
              ))}
        </div>

        {/* Special gauge */}
        <div className="w-full h-2 bg-bg-card rounded-full overflow-hidden">
          <div
            className="h-full rounded-full transition-all duration-200"
            style={{
              width: `${Math.min((tapCount / requiredForSpecial) * 100, 100)}%`,
              background: canSpecial
                ? "linear-gradient(90deg, var(--color-green), var(--color-accent))"
                : "linear-gradient(90deg, var(--color-accent), var(--color-accent-dark))",
            }}
          />
        </div>
        <p className="text-xs text-text-secondary">
          {tapCount}/{requiredForSpecial}
        </p>

        {canSpecial && result === null && (
          <button
            type="button"
            onClick={handleSpecial}
            className="w-full h-14 bg-green rounded-3xl text-base font-bold text-bg-primary cursor-pointer hover:opacity-90 transition active:scale-95 special-btn-appear"
          >
            SPECIAL ATTACK
          </button>
        )}

        {!canSpecial && (
          <span className="text-xs text-text-secondary text-center">tap anywhere to attack</span>
        )}
      </section>
    </div>
  );
}
