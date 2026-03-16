import "../styles/global.css";
import { NavBar } from "./ui/NavBar";

export function Capture() {
  return (
    <div className="showcase-screen">
      <NavBar title="CAPTURE" />

      <div className="flex-1 flex flex-col items-center justify-center gap-6 px-6">
        <div className="w-[200px] h-[200px] rounded-full bg-bg-card" />
        <div className="flex flex-col items-center gap-1">
          <span className="text-2xl font-bold text-text-primary">Python</span>
          <span className="text-5xl font-bold text-accent">42%</span>
        </div>
        <button
          type="button"
          className="w-20 h-20 rounded-full bg-accent border-none text-[32px] cursor-pointer flex items-center justify-center hover:opacity-90"
        >
          🎯
        </button>
        <span className="text-sm text-text-secondary">tap to throw</span>
      </div>

      <div className="flex gap-3 w-full px-6 pb-6">
        <button
          type="button"
          className="flex-1 flex items-center justify-center gap-2 px-5 py-4 bg-bg-card rounded-2xl border-none text-sm font-bold text-text-primary cursor-pointer hover:bg-bg-hover"
        >
          Use Item
        </button>
        <button
          type="button"
          className="flex-1 flex items-center justify-center gap-2 px-5 py-4 bg-bg-card rounded-2xl border-none text-sm font-bold text-text-secondary cursor-pointer hover:bg-bg-hover"
        >
          Skip
        </button>
      </div>
    </div>
  );
}
