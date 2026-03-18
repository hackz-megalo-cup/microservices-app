import type { MouseEvent as ReactMouseEvent } from "react";
import { useCallback, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router";
import "../../../styles/global.css";
import { useAuthContext } from "../../../lib/auth";
import { useGameConnection } from "../hooks/use-game-connection";
import type { ServerMessage } from "../types";

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

let dmgSeq = 0;
let rippleSeq = 0;

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
  const [timeoutSec, setTimeoutSec] = useState(300);
  const [floatingDmgs, setFloatingDmgs] = useState<FloatingDmg[]>([]);
  const [ripples, setRipples] = useState<Ripple[]>([]);
  const [squashing, setSquashing] = useState(false);
  const requiredForSpecial = 10;
  const squashTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const spawnDmg = useCallback((value: number, isSpecial: boolean) => {
    const x = 10 + Math.random() * 60;
    const y = 5 + Math.random() * 50;
    const entry: FloatingDmg = { id: ++dmgSeq, value, x, y, isSpecial };
    setFloatingDmgs((prev) => [...prev.slice(-8), entry]);
    setTimeout(() => {
      setFloatingDmgs((prev) => prev.filter((d) => d.id !== entry.id));
    }, 800);
  }, []);

  const triggerSquash = useCallback((duration: number) => {
    setSquashing(true);
    if (squashTimer.current) {
      clearTimeout(squashTimer.current);
    }
    squashTimer.current = setTimeout(() => setSquashing(false), duration);
  }, []);

  const spawnRipple = useCallback((clientX: number, clientY: number, rect: DOMRect) => {
    const x = clientX - rect.left;
    const y = clientY - rect.top;
    const entry: Ripple = { id: ++rippleSeq, x, y };
    setRipples((prev) => [...prev.slice(-4), entry]);
    setTimeout(() => {
      setRipples((prev) => prev.filter((r) => r.id !== entry.id));
    }, 500);
  }, []);

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
          spawnDmg(msg.lastDmg, false);
          triggerSquash(120);
          break;
        case "special_used":
          setBossHp(msg.bossHp);
          spawnDmg(msg.dmg, true);
          triggerSquash(250);
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
    [id, navigate, spawnDmg, triggerSquash],
  );

  const { status, sendTap, sendSpecial } = useGameConnection({
    userId,
    lobbyId: id,
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
    <div className="showcase-screen">
      <style>
        {`
          @keyframes dmg-float {
            0% { opacity: 1; transform: translateY(0) scale(1); }
            50% { opacity: 1; transform: translateY(-30px) scale(1.2); }
            100% { opacity: 0; transform: translateY(-60px) scale(0.8); }
          }
          @keyframes dmg-float-special {
            0% { opacity: 1; transform: translateY(0) scale(1.5); }
            30% { opacity: 1; transform: translateY(-20px) scale(2); }
            100% { opacity: 0; transform: translateY(-80px) scale(1); }
          }
          @keyframes boss-squash {
            0% { transform: scaleX(1) scaleY(1); }
            30% { transform: scaleX(1.08) scaleY(0.92); }
            60% { transform: scaleX(0.97) scaleY(1.03); }
            100% { transform: scaleX(1) scaleY(1); }
          }
          @keyframes boss-squash-heavy {
            0% { transform: scaleX(1) scaleY(1); }
            20% { transform: scaleX(1.15) scaleY(0.85); }
            50% { transform: scaleX(0.94) scaleY(1.06); }
            75% { transform: scaleX(1.03) scaleY(0.97); }
            100% { transform: scaleX(1) scaleY(1); }
          }
          .squash {
            animation: boss-squash 0.12s ease-out;
          }
          .squash-heavy {
            animation: boss-squash-heavy 0.25s ease-out;
          }
          @keyframes tap-ripple {
            0% { transform: translate(-50%, -50%) scale(0); opacity: 0.5; }
            100% { transform: translate(-50%, -50%) scale(1); opacity: 0; }
          }
          @keyframes special-appear {
            0% { opacity: 0; transform: scale(0.5) translateY(20px); }
            60% { opacity: 1; transform: scale(1.1) translateY(-5px); }
            100% { opacity: 1; transform: scale(1) translateY(0); }
          }
          .special-btn-appear {
            animation: special-appear 0.3s ease-out;
          }
        `}
      </style>

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

      {/* Tap area — entire boss visual region */}
      <button
        type="button"
        onClick={handleTap}
        disabled={!isConnected || result !== null}
        className="flex-1 relative flex items-center justify-center cursor-pointer select-none overflow-hidden disabled:cursor-not-allowed"
      >
        <img
          src="/images/battle-python.png"
          alt="Raid Boss"
          className={`w-[280px] h-[280px] object-cover rounded-2xl pointer-events-none transition-transform ${squashing ? "squash" : ""}`}
        />

        {/* Tap ripples */}
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

        {/* Floating damage numbers */}
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
          {tapCount}/{requiredForSpecial}
        </p>

        {/* Special button — appears only when charged */}
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
