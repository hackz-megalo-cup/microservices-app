import type { Raid } from "../types";
import "../styles/global.css";
import { RaidCard } from "./ui/RaidCard";
import { StatusBar } from "./ui/StatusBar";
import { TabBar } from "./ui/TabBar";

const mockRaids: Raid[] = [
  {
    id: "raid-1",
    name: "JavaScript",
    type: "Dynamic / JIT Compiled",
    difficulty: "5/10",
    timer: "12:34",
    image: "",
  },
  {
    id: "raid-2",
    name: "Rust",
    type: "Static / Compiled",
    difficulty: "8/10",
    timer: "05:12",
    image: "",
  },
  {
    id: "raid-3",
    name: "Go",
    type: "Static / Compiled",
    difficulty: "3/10",
    timer: "23:45",
    image: "",
  },
];

export function Home() {
  return (
    <div className="showcase-screen">
      <StatusBar />

      <header className="flex items-center justify-between px-6 py-3">
        <span className="text-xs font-bold tracking-widest text-text-secondary">POKÉMON</span>
        <button
          type="button"
          className="flex items-center justify-center w-11 h-11 bg-transparent text-xl cursor-pointer rounded-lg hover:bg-bg-hover"
          aria-label="Notifications"
        >
          🔔
        </button>
      </header>

      <section
        className="flex flex-col items-center justify-center gap-3 h-[280px]"
        style={{ background: "radial-gradient(circle, var(--color-accent-glow), transparent)" }}
      >
        <div className="w-[200px] h-[200px] rounded-full bg-bg-card" />
        <h1 className="text-2xl font-bold text-text-primary m-0">Python</h1>
        <p className="text-sm text-text-secondary m-0">Dynamic / Interpreted</p>
      </section>

      <section className="flex flex-col gap-3 px-6">
        <span className="text-xs font-bold tracking-widest text-text-secondary">ACTIVE RAIDS</span>
        {mockRaids.map((raid) => (
          <RaidCard
            key={raid.id}
            id={raid.id}
            name={raid.name}
            type={raid.type}
            difficulty={raid.difficulty}
            timer={raid.timer}
            image={raid.image}
          />
        ))}
      </section>

      <div className="flex-1" />

      <TabBar active="HOME" />
    </div>
  );
}
