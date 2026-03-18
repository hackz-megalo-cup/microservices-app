import { useCallback, useState } from "react";
import { useNavigate, useParams } from "react-router";
import "../../../styles/global.css";
import { useAuthContext } from "../../../lib/auth";
import { useGameConnection } from "../hooks/use-game-connection";
import type { ServerMessage } from "../types";

export function BattlePage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { user } = useAuthContext();
  const userId = user?.id ?? crypto.randomUUID();

  // --- Battle state ---
  const [bossHp, setBossHp] = useState(0);
  const [bossMaxHp, setBossMaxHp] = useState(0);
  const [tapCount, setTapCount] = useState(0);
  const [result, setResult] = useState<string | null>(null);
  const [lastDmg, setLastDmg] = useState<number | null>(null);
  const [timeoutSec, setTimeoutSec] = useState(300);
  const requiredForSpecial = 10;

  const onMessage = useCallback(
    (msg: ServerMessage) => {
      switch (msg.t) {
        case "joined":
          setBossHp(msg.bossHp);
          setBossMaxHp(msg.bossMaxHp);
          if (msg.timeoutSec) {
            setTimeoutSec(msg.timeoutSec);
          }
          break;
        case "hp":
          setBossHp(msg.hp);
          setLastDmg(msg.lastDmg);
          setTimeout(() => setLastDmg(null), 600);
          break;
        case "special_used":
          setBossHp(msg.bossHp);
          setLastDmg(msg.dmg);
          setTimeout(() => setLastDmg(null), 800);
          break;
        case "finished":
          setResult(msg.result);
          setTimeout(() => navigate(`/victory/${id}`), 2000);
          break;
        case "time_sync":
          setTimeoutSec(msg.remainingSec);
          break;
      }
    },
    [id, navigate],
  );

  const { status, sendTap, sendSpecial } = useGameConnection({
    userId,
    lobbyId: id,
    onMessage,
  });

  const handleTap = () => {
    sendTap();
    setTapCount((c) => c + 1);
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
    <div className="showcase-screen">
      {/* Boss info + HP bar */}
      <section className="flex flex-col gap-3 px-6 pt-4">
        <div className="flex items-center justify-center gap-3">
          <span className="text-lg font-bold text-text-primary">RAID BOSS</span>
          <span className="text-xs font-bold text-accent bg-accent-glow px-3 py-1 rounded-lg">
            {timerDisplay}
          </span>
        </div>

        {/* Connection status */}
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

      {/* Boss visual area */}
      <div className="flex-1 relative flex items-center justify-center">
        <img
          src="/images/battle-python.png"
          alt="Raid Boss"
          className="w-[280px] h-[280px] object-cover rounded-2xl"
        />
        {lastDmg !== null && (
          <span className="absolute top-10 right-10 text-2xl font-bold text-accent animate-bounce">
            -{lastDmg}
          </span>
        )}
        {result && (
          <div className="absolute inset-0 flex items-center justify-center bg-bg-primary/80 rounded-2xl">
            <span className="text-4xl font-bold text-accent">
              {result === "win" ? "VICTORY!" : "TIME UP"}
            </span>
          </div>
        )}
      </div>

      {/* Attack controls */}
      <section className="flex flex-col items-center gap-4 px-6 pb-6">
        {/* Participant indicators */}
        <div className="flex gap-3">
          <div className="w-8 h-8 rounded-full bg-bg-card border-2 border-accent" />
          <div className="w-8 h-8 rounded-full bg-bg-card border-2 border-green" />
          <div className="w-8 h-8 rounded-full bg-bg-card border-2 border-text-secondary" />
          <div className="w-8 h-8 rounded-full bg-bg-card border-2 border-text-secondary" />
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
          Special: {tapCount}/{requiredForSpecial}
        </p>

        {/* Buttons */}
        <div className="w-full flex gap-3">
          <button
            type="button"
            onClick={handleTap}
            disabled={!isConnected || result !== null}
            className="flex-1 h-14 bg-accent rounded-3xl text-base font-bold text-bg-primary cursor-pointer hover:opacity-90 disabled:opacity-40 disabled:cursor-not-allowed transition active:scale-95"
          >
            ATTACK
          </button>
          <button
            type="button"
            onClick={handleSpecial}
            disabled={!canSpecial || result !== null}
            className="w-24 h-14 bg-green rounded-3xl text-sm font-bold text-bg-primary cursor-pointer hover:opacity-90 disabled:opacity-40 disabled:cursor-not-allowed transition active:scale-95"
          >
            SPECIAL
          </button>
        </div>
        <span className="text-xs text-text-secondary text-center">tap to attack</span>
      </section>
    </div>
  );
}
