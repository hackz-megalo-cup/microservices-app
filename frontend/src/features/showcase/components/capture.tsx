import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { useAuthContext } from "../../../lib/auth";
import "../../../styles/global.css";
import "./capture.css";
import { useCaptureItems } from "../hooks/use-capture-items";
import { NavBar } from "./ui/nav-bar";

// ── Types ────────────────────────────────────────────────────────────────────

type Phase = "idle" | "throwing" | "wobbling" | "burst" | "success" | "failed" | "escaped";

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

// Animation timing constants (in milliseconds)
const THROW_DURATION_MS = 870;
const WOBBLE_DELAY_MS = 200;
const BURST_DURATION_MS = 550;
const WOBBLE_DURATIONS: Record<number, number> = {
  1: 900,
  2: 1450,
  3: 2000,
};

// Interaction constants
const THROW_VELOCITY_THRESHOLD = 250;
const THROW_DRAG_THRESHOLD = 55;

// ── Helpers ──────────────────────────────────────────────────────────────────

function getCatchBonus(bonus: ThrowBonus): number {
  if (bonus === "excellent") {
    return 1.85;
  }
  if (bonus === "great") {
    return 1.5;
  }
  if (bonus === "nice") {
    return 1.15;
  }
  return 1.0;
}

function getThrowBonus(scale: number): ThrowBonus {
  if (scale < 0.35) {
    return "excellent";
  }
  if (scale < 0.6) {
    return "great";
  }
  if (scale < 0.8) {
    return "nice";
  }
  return "normal";
}

function getRingColor(rate: number): string {
  if (rate > 0.5) {
    return "#22c55e";
  }
  if (rate > 0.3) {
    return "#f59e0b";
  }
  return "#ef4444";
}

function makeBonusLabel(bonus: ThrowBonus): string {
  if (bonus === "excellent") {
    return "EXCELLENT!";
  }
  if (bonus === "great") {
    return "GREAT!";
  }
  if (bonus === "nice") {
    return "NICE!";
  }
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
  const { user } = useAuthContext();
  const userId = user?.id ?? "";
  const {
    availableItems,
    isLoading,
    error,
    handleUseItem: handleUseItemApi,
    isPending: isMutationPending,
    refetch,
  } = useCaptureItems(userId);

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

  // Item modal state
  const [showItemModal, setShowItemModal] = useState(false);
  const [selectedItemForThrow, setSelectedItemForThrow] = useState<{
    itemId: string;
    bonus: number;
  } | null>(null);

  // CSS custom property for ball horizontal offset during throw
  const ballDxRef = useRef(0);

  const animFrameRef = useRef<number>(0);
  const circleStartRef = useRef<number>(Date.now());
  const timeoutIdsRef = useRef<number[]>([]);

  // Keep phaseRef in sync
  useEffect(() => {
    phaseRef.current = phase;
  }, [phase]);

  // ── Circle shrink animation ──
  useEffect(() => {
    if (phase !== "idle") {
      return;
    }

    circleStartRef.current = Date.now();

    const tick = () => {
      if (phaseRef.current !== "idle") {
        return;
      }
      const elapsed = (Date.now() - circleStartRef.current) % CIRCLE_CYCLE_MS;
      const t = elapsed / CIRCLE_CYCLE_MS;
      // 1.0 → 0.18, then reset (creates the shrinking ring effect)
      setCircleScale(1.0 - t * 0.82);
      animFrameRef.current = requestAnimationFrame(tick);
    };

    animFrameRef.current = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(animFrameRef.current);
  }, [phase]);

  // ── Cleanup timers on unmount ──
  useEffect(() => {
    return () => {
      cancelAnimationFrame(animFrameRef.current);
      timeoutIdsRef.current.forEach(clearTimeout);
      timeoutIdsRef.current = [];
    };
  }, []);

  // ── Throw logic ──
  const doThrow = useCallback(
    (bonus: number, itemId?: string) => {
      if (phaseRef.current !== "idle") {
        return;
      }

      const throwBonus = getThrowBonus(circleScale);
      setThrowBonus(throwBonus);
      const ballDx = 0; // Subtle horizontal arc
      ballDxRef.current = ballDx * 0.4;

      // If item is selected, use it
      if (itemId) {
        handleUseItemApi(itemId, bonus);
      }

      // Start throw phase
      setPokemonClass("capture-pokemon-absorb");
      setPhase("throwing");

      // After throw animation finishes → wobble phase
      const t1 = window.setTimeout(() => {
        const wobbles = Math.floor(Math.random() * 3) + 1;
        setWobbleCount(wobbles);
        setPhase("wobbling");

        // After wobbling → result
        const wobbleDuration = WOBBLE_DURATIONS[wobbles];
        const t2 = window.setTimeout(() => {
          const effectiveBonus = itemId
            ? getCatchBonus(throwBonus) * (1 + bonus)
            : getCatchBonus(throwBonus);
          const effectiveRate = BASE_CATCH_RATE * effectiveBonus;
          const success = Math.random() < effectiveRate;

          if (success) {
            setParticles(makeParticles());
            setPhase("success");
          } else {
            // Ball bursts open
            setPhase("burst");
            const t3 = window.setTimeout(() => {
              const fled = Math.random() < 0.35;
              setPokemonClass(fled ? "capture-pokemon-escape" : "capture-pokemon-breakfree");
              setPhase(fled ? "escaped" : "failed");
            }, BURST_DURATION_MS);
            timeoutIdsRef.current.push(t3);
          }
        }, wobbleDuration + WOBBLE_DELAY_MS);
        timeoutIdsRef.current.push(t2);
      }, THROW_DURATION_MS);
      timeoutIdsRef.current.push(t1);
    },
    [circleScale, handleUseItemApi],
  );

  // ── Pointer handlers ──
  const handlePointerDown = (e: React.PointerEvent<Element>) => {
    if (phase !== "idle") {
      return;
    }
    dragStart.current = { x: e.clientX, y: e.clientY, time: Date.now() };
    setIsDragging(true);
    e.currentTarget.setPointerCapture(e.pointerId);
  };

  const handlePointerMove = (e: React.PointerEvent<Element>) => {
    if (!isDragging) {
      return;
    }
    setBallDragOffset({
      x: e.clientX - dragStart.current.x,
      y: e.clientY - dragStart.current.y,
    });
  };

  const handlePointerUp = (e: React.PointerEvent<Element>) => {
    if (!isDragging) {
      return;
    }
    setIsDragging(false);
    setBallDragOffset({ x: 0, y: 0 });

    const dy = e.clientY - dragStart.current.y;
    const dt = Math.max((Date.now() - dragStart.current.time) / 1000, 0.05);
    const vy = dy / dt;

    // Throw when swiped upward fast enough or dragged up far enough
    if (vy < -THROW_VELOCITY_THRESHOLD || dy < -THROW_DRAG_THRESHOLD) {
      // Check if item is selected
      if (selectedItemForThrow) {
        doThrow(selectedItemForThrow.bonus, selectedItemForThrow.itemId);
        setSelectedItemForThrow(null);
      } else {
        // Just throw without item
        doThrow(0);
      }
    }
  };

  // ── Item selection handler ──
  const handleSelectItem = (itemId: string, bonus: number) => {
    setSelectedItemForThrow({ itemId, bonus });
    setShowItemModal(false);
  };

  // ── Retry ──
  const retry = () => {
    cancelAnimationFrame(animFrameRef.current);
    timeoutIdsRef.current.forEach(clearTimeout);
    timeoutIdsRef.current = [];
    setPhase("idle");
    setThrowBonus("normal");
    setWobbleCount(0);
    setBallDragOffset({ x: 0, y: 0 });
    setParticles([]);
    setPokemonClass("capture-pokemon-idle");
    setSelectedItemForThrow(null);
  };

  // ── Loading / Error states ──
  if (isLoading) {
    return (
      <div className="showcase-screen">
        <NavBar title="CAPTURE" />
        <div className="flex-1 flex items-center justify-center text-text-secondary text-sm">
          Loading...
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="showcase-screen">
        <NavBar title="CAPTURE" />
        <div className="flex-1 flex flex-col items-center justify-center gap-3 px-6">
          <p className="text-sm text-text-secondary m-0">{error.message}</p>
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

  // ── Derived values ──
  const ringColor = getRingColor(BASE_CATCH_RATE);
  const catchPercent = Math.round(BASE_CATCH_RATE * getCatchBonus(throwBonus) * 100);
  const bonusLabel = makeBonusLabel(throwBonus);
  const showRings = phase === "idle" || phase === "throwing";
  const showBall =
    phase === "idle" || phase === "throwing" || phase === "wobbling" || phase === "burst";

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
            {phase === "idle" && (
              <span className="capture-hint">
                {selectedItemForThrow ? "🎯 Ready!" : "Swipe up to throw!"}
              </span>
            )}

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
                <div
                  className={`capture-pokeball ${isDragging ? "capture-pokeball-dragging" : ""}`}
                >
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
            <button
              type="button"
              className="capture-item-btn"
              onClick={() => setShowItemModal(true)}
              disabled={isMutationPending}
            >
              {selectedItemForThrow ? "✓ Item Ready" : "🍓 Use Item"}
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

        {/* Item selection modal */}
        {showItemModal && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="bg-bg-primary rounded-3xl p-6 max-w-sm w-[90%] max-h-[80vh] flex flex-col">
              <h2 className="text-xl font-bold text-text-primary mb-4">Select Item</h2>

              {availableItems.length === 0 && (
                <p className="text-sm text-text-secondary text-center flex-1 flex items-center justify-center">
                  No items available
                </p>
              )}

              <div className="flex-1 overflow-y-auto space-y-2">
                {availableItems.map((item) => (
                  <button
                    key={item.id}
                    type="button"
                    className={`w-full rounded-2xl p-4 text-left border-none cursor-pointer transition ${
                      selectedItemForThrow?.itemId === item.id
                        ? "bg-accent text-text-primary"
                        : "bg-bg-card hover:bg-bg-hover text-text-primary"
                    }`}
                    onClick={() => handleSelectItem(item.id, item.captureRateBonus)}
                    disabled={isMutationPending}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex-1">
                        <p className="text-sm font-bold text-text-primary m-0">{item.name}</p>
                        <p className="text-xs text-text-secondary m-0">
                          Capture Rate +{Math.round(item.captureRateBonus * 100)}%
                        </p>
                      </div>
                      <span className="text-xs font-bold text-text-secondary bg-bg-primary px-2 py-1 rounded">
                        x{item.quantity}
                      </span>
                    </div>
                  </button>
                ))}
              </div>

              <button
                type="button"
                className="mt-4 w-full bg-bg-card rounded-2xl px-4 py-2 border-none text-sm font-bold text-text-secondary cursor-pointer hover:bg-bg-hover"
                onClick={() => setShowItemModal(false)}
                disabled={isMutationPending}
              >
                Cancel
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
