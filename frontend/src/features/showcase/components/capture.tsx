import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";
import "../../../styles/global.css";
import "./capture.css";
import { NavBar } from "./ui/nav-bar";

// ── Types ────────────────────────────────────────────────────────────────────

type Phase =
  | "idle"
  | "throwing"
  | "wobbling"
  | "burst"
  | "success"
  | "failed"
  | "escaped";

type ThrowBonus = "excellent" | "great" | "nice" | "normal";

type Particle = {
  id: number;
  px: number; // final x offset (px)
  py: number; // final y offset (px)
  color: string;
};

// ── Constants ────────────────────────────────────────────────────────────────

const POKEMON_NAME = "Python";
const BASE_CATCH_RATE = 0.45;
const CIRCLE_CYCLE_MS = 2500;
const PARTICLE_COLORS = ["#06b6d4", "#22c55e", "#f59e0b", "#ec4899", "#a855f7", "#f97316"];

// ── Helpers ──────────────────────────────────────────────────────────────────

function getCatchBonus(bonus: ThrowBonus): number {
  if (bonus === "excellent") return 1.85;
  if (bonus === "great") return 1.5;
  if (bonus === "nice") return 1.15;
  return 1.0;
}

function getThrowBonus(scale: number): ThrowBonus {
  if (scale < 0.35) return "excellent";
  if (scale < 0.6) return "great";
  if (scale < 0.8) return "nice";
  return "normal";
}

function getRingColor(rate: number): string {
  if (rate > 0.5) return "#22c55e";
  if (rate > 0.3) return "#f59e0b";
  return "#ef4444";
}

function makeBonusLabel(bonus: ThrowBonus): string {
  if (bonus === "excellent") return "EXCELLENT!";
  if (bonus === "great") return "GREAT!";
  if (bonus === "nice") return "NICE!";
  return "";
}

function makeParticles(): Particle[] {
  return Array.from({ length: 22 }, (_, i) => {
    const angle = (i / 22) * Math.PI * 2;
    const dist = 55 + Math.random() * 65;
    return {
      id: i,
      px: Math.cos(angle) * dist,
      py: Math.sin(angle) * dist,
      color: PARTICLE_COLORS[i % PARTICLE_COLORS.length],
    };
  });
}

// ── Component ────────────────────────────────────────────────────────────────

export function Capture() {
  const navigate = useNavigate();

  // Game state
  const [phase, setPhase] = useState<Phase>("idle");
  const phaseRef = useRef<Phase>("idle");
  const [circleScale, setCircleScale] = useState(1.0);
  const [throwBonus, setThrowBonus] = useState<ThrowBonus>("normal");
  const [wobbleCount, setWobbleCount] = useState(0);
  const [particles, setParticles] = useState<Particle[]>([]);
  const [pokemonClass, setPokemonClass] = useState("capture-pokemon-idle");

  // Drag state
  const [isDragging, setIsDragging] = useState(false);
  const [ballDragOffset, setBallDragOffset] = useState({ x: 0, y: 0 });
  const dragStart = useRef({ x: 0, y: 0, time: 0 });

  // CSS custom property for ball horizontal offset during throw
  const ballDxRef = useRef(0);

  const animFrameRef = useRef<number>(0);
  const circleStartRef = useRef<number>(Date.now());

  // Keep phaseRef in sync
  useEffect(() => {
    phaseRef.current = phase;
  }, [phase]);

  // ── Circle shrink animation ──
  useEffect(() => {
    if (phase !== "idle") return;

    circleStartRef.current = Date.now();

    const tick = () => {
      if (phaseRef.current !== "idle") return;
      const elapsed = (Date.now() - circleStartRef.current) % CIRCLE_CYCLE_MS;
      const t = elapsed / CIRCLE_CYCLE_MS;
      // 1.0 → 0.18, then reset (creates the shrinking ring effect)
      setCircleScale(1.0 - t * 0.82);
      animFrameRef.current = requestAnimationFrame(tick);
    };

    animFrameRef.current = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(animFrameRef.current);
  }, [phase]);

  // ── Throw logic ──
  const doThrow = useCallback(
    (dx: number) => {
      if (phaseRef.current !== "idle") return;

      const bonus = getThrowBonus(circleScale);
      setThrowBonus(bonus);
      ballDxRef.current = dx * 0.4; // Subtle horizontal arc

      // Start throw phase
      setPokemonClass("capture-pokemon-absorb");
      setPhase("throwing");

      // After throw animation finishes → wobble phase
      setTimeout(() => {
        const wobbles = Math.floor(Math.random() * 3) + 1;
        setWobbleCount(wobbles);
        setPhase("wobbling");

        // After wobbling → result
        const wobbleDuration = wobbles === 1 ? 900 : wobbles === 2 ? 1450 : 2000;
        setTimeout(() => {
          const effectiveRate = BASE_CATCH_RATE * getCatchBonus(bonus);
          const success = Math.random() < effectiveRate;

          if (success) {
            setParticles(makeParticles());
            setPhase("success");
          } else {
            // Ball bursts open
            setPhase("burst");
            setTimeout(() => {
              const fled = Math.random() < 0.35;
              setPokemonClass(fled ? "capture-pokemon-escape" : "capture-pokemon-breakfree");
              setPhase(fled ? "escaped" : "failed");
            }, 550);
          }
        }, wobbleDuration + 200);
      }, 870);
    },
    [circleScale],
  );

  // ── Pointer handlers ──
  const handlePointerDown = (e: React.PointerEvent) => {
    if (phase !== "idle") return;
    dragStart.current = { x: e.clientX, y: e.clientY, time: Date.now() };
    setIsDragging(true);
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
  };

  const handlePointerMove = (e: React.PointerEvent) => {
    if (!isDragging) return;
    setBallDragOffset({
      x: e.clientX - dragStart.current.x,
      y: e.clientY - dragStart.current.y,
    });
  };

  const handlePointerUp = (e: React.PointerEvent) => {
    if (!isDragging) return;
    setIsDragging(false);
    setBallDragOffset({ x: 0, y: 0 });

    const dy = e.clientY - dragStart.current.y;
    const dx = e.clientX - dragStart.current.x;
    const dt = Math.max((Date.now() - dragStart.current.time) / 1000, 0.05);
    const vy = dy / dt;

    // Throw when swiped upward fast enough or dragged up far enough
    if (vy < -250 || dy < -55) {
      doThrow(dx);
    }
  };

  // ── Retry ──
  const retry = () => {
    cancelAnimationFrame(animFrameRef.current);
    setPhase("idle");
    setThrowBonus("normal");
    setWobbleCount(0);
    setBallDragOffset({ x: 0, y: 0 });
    setParticles([]);
    setPokemonClass("capture-pokemon-idle");
  };

  // ── Derived values ──
  const ringColor = getRingColor(BASE_CATCH_RATE);
  const catchPercent = Math.round(BASE_CATCH_RATE * getCatchBonus(throwBonus) * 100);
  const bonusLabel = makeBonusLabel(throwBonus);
  const showRings = phase === "idle" || phase === "throwing";
  const showBall =
    phase === "idle" ||
    phase === "throwing" ||
    phase === "wobbling" ||
    phase === "burst";

  const ballWrapperClass = [
    "capture-ball-wrapper",
    phase === "throwing" ? "capture-ball-throwing" : "",
    phase === "wobbling"
      ? wobbleCount === 1
        ? "capture-ball-wobble-1"
        : wobbleCount === 2
          ? "capture-ball-wobble-2"
          : "capture-ball-wobble-3"
      : "",
    phase === "burst" ? "capture-ball-burst" : "",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div className="showcase-screen">
      <NavBar title="CAPTURE" />

      {/* ── Game area ── */}
      <div className="capture-game-area">
        {/* Background glow */}
        <div className="capture-bg-glow" />

        {/* Pokémon + rings */}
        <div className="capture-pokemon-section">
          <div className="capture-rings-wrapper">
            {/* Outer ring */}
            {showRings && <div className="capture-ring-outer" />}

            {/* Inner shrinking ring */}
            {showRings && (
              <div
                className="capture-ring-inner"
                style={{
                  transform: `scale(${circleScale})`,
                  borderColor: ringColor,
                  boxShadow: `0 0 10px ${ringColor}80`,
                }}
              />
            )}

            {/* Pokémon */}
            <img
              src="/images/capture-python.png"
              alt={POKEMON_NAME}
              className={`capture-pokemon-img ${pokemonClass}`}
            />
          </div>

          {/* Stats */}
          <div className="capture-stats-bar">
            <span className="capture-pokemon-name">{POKEMON_NAME}</span>
            <span className="capture-catch-rate" style={{ color: ringColor }}>
              {catchPercent}%
            </span>
          </div>

          {/* Throw bonus label */}
          {bonusLabel && phase === "throwing" && (
            <div className="capture-bonus-label">{bonusLabel}</div>
          )}
        </div>

        {/* Particles on success */}
        {phase === "success" && particles.length > 0 && (
          <div className="capture-particles-layer">
            {particles.map((p) => (
              <div
                key={p.id}
                className="capture-particle"
                style={
                  {
                    "--px": `${p.px}px`,
                    "--py": `${p.py}px`,
                    backgroundColor: p.color,
                    animationDelay: `${p.id * 18}ms`,
                  } as React.CSSProperties
                }
              />
            ))}
          </div>
        )}

        {/* Pokéball */}
        {showBall && (
          <div className="capture-ball-container">
            {phase === "idle" && <span className="capture-hint">Swipe up to throw!</span>}

            {/* Wrap div handles animation class; inner div handles drag transform */}
            <div
              className={ballWrapperClass}
              style={
                phase === "throwing"
                  ? ({ "--ball-dx": `${ballDxRef.current}px` } as React.CSSProperties)
                  : undefined
              }
            >
              <div
                style={
                  phase === "idle"
                    ? {
                        transform: `translate(${ballDragOffset.x}px, ${ballDragOffset.y}px)`,
                        transition: isDragging ? "none" : "transform 0.2s ease",
                      }
                    : undefined
                }
                onPointerDown={handlePointerDown}
                onPointerMove={handlePointerMove}
                onPointerUp={handlePointerUp}
              >
                <div className={`capture-pokeball ${isDragging ? "capture-pokeball-dragging" : ""}`}>
                  <div className="pokeball-top" />
                  <div className="pokeball-bottom" />
                  <div className="pokeball-band" />
                  <div className="pokeball-button" />
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Result overlay — Success */}
        {phase === "success" && (
          <div className="capture-result capture-result-success">
            <div className="capture-result-emoji">🎉</div>
            <div className="capture-result-title">Gotcha!</div>
            <div className="capture-result-subtitle">{POKEMON_NAME} was caught!</div>
            <div className="capture-result-rewards">
              <span>+500 EXP</span>
              <span>+3 Candy</span>
              <span>+50 Stardust</span>
            </div>
            <button
              type="button"
              className="capture-result-btn"
              onClick={() => void navigate("/collection")}
            >
              View Collection
            </button>
          </div>
        )}

        {/* Result overlay — Failed / Escaped */}
        {(phase === "failed" || phase === "escaped") && (
          <div className="capture-result">
            <div className="capture-result-emoji">{phase === "escaped" ? "💨" : "😤"}</div>
            <div className="capture-result-title">
              {phase === "escaped" ? "Oh no!" : "It broke free!"}
            </div>
            <div className="capture-result-subtitle">
              {phase === "escaped" ? `${POKEMON_NAME} fled away!` : "Keep trying!"}
            </div>
            {phase === "escaped" ? (
              <button
                type="button"
                className="capture-result-btn"
                onClick={() => void navigate(-1)}
              >
                Go Back
              </button>
            ) : (
              <button type="button" className="capture-result-btn" onClick={retry}>
                Try Again
              </button>
            )}
          </div>
        )}

        {/* Item & Run buttons (idle only) */}
        {phase === "idle" && (
          <div className="capture-item-row">
            <button type="button" className="capture-item-btn">
              🍓 Use Berry
            </button>
            <button
              type="button"
              className="capture-item-btn capture-item-btn-secondary"
              onClick={() => void navigate(-1)}
            >
              ✕ Run
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
