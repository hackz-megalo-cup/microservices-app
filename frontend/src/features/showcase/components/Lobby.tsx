import type { Trainer } from "../types";
import "../styles/global.css";
import { NavBar } from "./ui/NavBar";

const mockTrainers: Trainer[] = [
  { name: "RustLover42", pokemon: "Ferris", online: true },
  { name: "GoGopher99", pokemon: "Gopher", online: true },
  { name: "JSNinja", pokemon: "Node", online: false },
  { name: "SwiftSamurai", pokemon: "Swift", online: false },
];

export function Lobby() {
  return (
    <div className="showcase-screen">
      <NavBar title="RAID LOBBY" rightIcon="share" />

      <div className="flex flex-col gap-3 px-4 pb-4 flex-1">
        <section className="flex flex-col items-center gap-3 py-2">
          <div className="w-40 h-40 rounded-full bg-bg-card" />
          <div className="flex items-center gap-2">
            <span className="text-xl font-bold text-text-primary">Python</span>
            <span className="text-xs text-accent bg-bg-card px-3 py-1 rounded-lg">Dynamic</span>
          </div>
        </section>

        <div className="flex flex-col items-center gap-3 bg-bg-card rounded-2xl p-6">
          <div className="w-20 h-20 flex items-center justify-center text-2xl font-bold text-accent">
            QR
          </div>
          <div className="flex items-center gap-2">
            <span className="text-lg font-bold text-text-primary">RAID-7X4K</span>
            <button
              type="button"
              className="bg-transparent text-base cursor-pointer p-1"
              aria-label="Copy code"
            >
              📋
            </button>
          </div>
        </div>

        <section className="flex flex-col gap-3">
          <div className="flex items-center justify-between">
            <span className="text-xs font-bold tracking-widest text-text-secondary">TRAINERS</span>
            <span className="text-xs font-bold tracking-widest text-text-secondary">4/6</span>
          </div>
          {mockTrainers.map((trainer) => (
            <div
              key={trainer.name}
              className="flex items-center gap-3 bg-bg-card rounded-2xl px-4 py-3"
            >
              <div className="w-8 h-8 rounded-full bg-bg-hover shrink-0" />
              <span className="text-sm text-text-primary flex-1">{trainer.name}</span>
              <span className="text-xs text-text-secondary">{trainer.pokemon}</span>
              <div
                className={`w-2 h-2 rounded-full shrink-0 ${trainer.online ? "bg-accent" : "bg-text-secondary"}`}
              />
            </div>
          ))}
        </section>

        <div className="flex-1" />

        <button
          type="button"
          className="w-full h-14 bg-accent rounded-3xl text-base font-bold text-bg-primary cursor-pointer hover:opacity-90"
        >
          ENTER RAID
        </button>
      </div>
    </div>
  );
}
