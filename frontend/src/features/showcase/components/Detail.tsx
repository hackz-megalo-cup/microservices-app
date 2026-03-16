import { useParams } from "react-router";
import type { Pokemon } from "../types";
import "../styles/global.css";
import { NavBar } from "./ui/NavBar";

const mockPokemonMap: Record<string, Pokemon> = {
  "1": {
    id: "1",
    name: "TypeScript",
    number: "#001",
    image: "",
    types: ["Static", "Class-Based"],
    stats: [
      { label: "HP", value: 85 },
      { label: "ATK", value: 72 },
      { label: "SPD", value: 91 },
    ],
    about:
      "A statically typed superset of JavaScript that compiles to clean, readable code. Known for its powerful type system and class-based architecture.",
    moves: [
      { name: "Type Check", type: "Normal", power: 45 },
      { name: "Compile", type: "Normal", power: 60 },
      { name: "Refactor", type: "Normal", power: 80 },
    ],
    captured: true,
  },
  "2": {
    id: "2",
    name: "Rust",
    number: "#002",
    image: "",
    types: ["Static", "Systems"],
    stats: [
      { label: "HP", value: 95 },
      { label: "ATK", value: 88 },
      { label: "SPD", value: 78 },
    ],
    about:
      "A systems programming language focused on safety, concurrency, and performance. Famous for its borrow checker.",
    moves: [
      { name: "Borrow Check", type: "Normal", power: 50 },
      { name: "Unsafe Block", type: "Normal", power: 90 },
      { name: "Pattern Match", type: "Normal", power: 65 },
    ],
    captured: true,
  },
  "3": {
    id: "3",
    name: "Python",
    number: "#003",
    image: "",
    types: ["Dynamic", "Interpreted"],
    stats: [
      { label: "HP", value: 80 },
      { label: "ATK", value: 65 },
      { label: "SPD", value: 60 },
    ],
    about:
      "A versatile interpreted language known for readability and simplicity. Widely used in data science and web development.",
    moves: [
      { name: "List Comprehension", type: "Normal", power: 55 },
      { name: "Decorator", type: "Normal", power: 70 },
      { name: "GIL Release", type: "Normal", power: 85 },
    ],
    captured: true,
  },
  "4": {
    id: "4",
    name: "Go",
    number: "#004",
    image: "",
    types: ["Static", "Compiled"],
    stats: [
      { label: "HP", value: 88 },
      { label: "ATK", value: 70 },
      { label: "SPD", value: 95 },
    ],
    about:
      "A compiled language designed for simplicity and concurrency. Known for goroutines and fast compilation.",
    moves: [
      { name: "Goroutine", type: "Normal", power: 60 },
      { name: "Channel Send", type: "Normal", power: 55 },
      { name: "Defer", type: "Normal", power: 40 },
    ],
    captured: true,
  },
  "5": {
    id: "5",
    name: "C",
    number: "#005",
    image: "",
    types: ["Static", "Low-Level"],
    stats: [
      { label: "HP", value: 70 },
      { label: "ATK", value: 95 },
      { label: "SPD", value: 99 },
    ],
    about:
      "The foundational systems language powering operating systems worldwide. Minimal abstraction, maximum control.",
    moves: [
      { name: "Pointer Arithmetic", type: "Normal", power: 80 },
      { name: "Malloc", type: "Normal", power: 70 },
      { name: "Segfault", type: "Normal", power: 100 },
    ],
    captured: true,
  },
};

export function Detail() {
  const { id } = useParams<{ id: string }>();
  const pokemon = mockPokemonMap[id ?? "1"];

  if (!pokemon) {
    return (
      <div className="showcase-screen">
        <NavBar title="NOT FOUND" />
        <p className="text-center p-6">Pokemon not found</p>
      </div>
    );
  }

  return (
    <div className="showcase-screen">
      <NavBar title={pokemon.number} rightIcon="heart" />

      <div
        className="flex items-center justify-center h-[200px]"
        style={{ background: "radial-gradient(circle, var(--color-accent-glow), transparent)" }}
      >
        <div className="w-[200px] h-[200px] rounded-full bg-bg-card" />
      </div>

      <div className="flex flex-col items-center gap-3 px-6">
        <h1 className="text-2xl font-bold m-0">{pokemon.name}</h1>
        <div className="flex gap-2">
          {pokemon.types.map((type) => (
            <span
              key={type}
              className="text-xs text-text-secondary bg-bg-card px-4 py-2 rounded-full"
            >
              {type}
            </span>
          ))}
        </div>
      </div>

      <div className="flex gap-3 pt-4 px-6">
        {pokemon.stats.map((stat) => (
          <div
            key={stat.label}
            className="flex-1 flex flex-col items-center gap-1 bg-bg-card rounded-2xl p-4"
          >
            <span className="text-2xl font-bold text-accent">{stat.value}</span>
            <span className="text-xs font-semibold tracking-wide text-text-secondary">
              {stat.label}
            </span>
          </div>
        ))}
      </div>

      <div className="flex flex-col gap-3 px-6">
        <div className="flex flex-col gap-2 bg-bg-card rounded-2xl p-4">
          <span className="text-sm font-bold">About</span>
          <p className="text-xs text-text-secondary leading-relaxed m-0">{pokemon.about}</p>
        </div>

        <div className="flex flex-col gap-2">
          <span className="text-sm font-bold">Moves</span>
          {pokemon.moves.map((move) => (
            <div
              key={move.name}
              className="flex justify-between items-center bg-bg-card rounded-2xl px-4 py-3"
            >
              <span className="text-sm">{move.name}</span>
              <span className="text-sm font-bold text-accent">{move.power}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
