import "../styles/global.css";
import { StatusBar } from "./ui/StatusBar";

export function Battle() {
  return (
    <div className="showcase-screen">
      <StatusBar />

      <section className="flex flex-col gap-3 px-6">
        <div className="flex items-center justify-center gap-3">
          <span className="text-lg font-bold text-text-primary">Python</span>
          <span className="text-xs font-bold text-accent bg-accent-glow px-3 py-1 rounded-lg">
            2:34
          </span>
        </div>
        <div className="w-full h-2 bg-bg-card rounded overflow-hidden">
          <div
            className="h-full rounded"
            style={{
              width: "65%",
              background: "linear-gradient(90deg, var(--color-accent), var(--color-accent-dark))",
            }}
          />
        </div>
      </section>

      <div className="flex-1 relative flex items-center justify-center">
        <img
          src="/images/battle-python.png"
          alt="Python"
          className="w-[280px] h-[280px] object-cover rounded-2xl"
        />
        <span className="absolute top-10 right-10 text-2xl font-bold text-accent">-342</span>
        <span className="absolute top-20 left-10 text-xl font-bold text-text-secondary opacity-60">
          -128
        </span>
      </div>

      <section className="flex flex-col items-center gap-4 px-6 pb-6">
        <div className="flex gap-3">
          <div className="w-8 h-8 rounded-full bg-bg-card border-2 border-accent" />
          <div className="w-8 h-8 rounded-full bg-bg-card border-2 border-green" />
          <div className="w-8 h-8 rounded-full bg-bg-card border-2 border-text-secondary" />
          <div className="w-8 h-8 rounded-full bg-bg-card border-2 border-text-secondary" />
        </div>
        <div className="w-full h-2 bg-bg-card rounded overflow-hidden">
          <div
            className="h-full rounded"
            style={{
              width: "60%",
              background: "linear-gradient(90deg, var(--color-accent), var(--color-accent-dark))",
            }}
          />
        </div>
        <button
          type="button"
          className="w-full h-14 bg-accent rounded-3xl text-base font-bold text-bg-primary cursor-pointer hover:opacity-90"
        >
          ATTACK
        </button>
        <span className="text-xs text-text-secondary text-center">tap to attack</span>
      </section>
    </div>
  );
}
