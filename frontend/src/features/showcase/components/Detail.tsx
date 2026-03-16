import { useParams } from "react-router";
import type { Pokemon } from "../types";
import "../styles/global.css";
import styles from "./Detail.module.css";
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
        <p style={{ textAlign: "center", padding: 24 }}>Pokemon not found</p>
      </div>
    );
  }

  return (
    <div className="showcase-screen">
      <NavBar title={pokemon.number} rightIcon="heart" />

      <div className={styles.heroArea}>
        <div className={styles.heroModel} />
      </div>

      <div className={styles.nameSection}>
        <h1 className={styles.name}>{pokemon.name}</h1>
        <div className={styles.badges}>
          {pokemon.types.map((type) => (
            <span key={type} className={styles.badge}>
              {type}
            </span>
          ))}
        </div>
      </div>

      <div className={styles.statsRow}>
        {pokemon.stats.map((stat) => (
          <div key={stat.label} className={styles.statCard}>
            <span className={styles.statValue}>{stat.value}</span>
            <span className={styles.statLabel}>{stat.label}</span>
          </div>
        ))}
      </div>

      <div className={styles.aboutWrapper}>
        <div className={styles.aboutCard}>
          <span className={styles.sectionTitle}>About</span>
          <p className={styles.aboutText}>{pokemon.about}</p>
        </div>

        <div className={styles.movesSection}>
          <span className={styles.sectionTitle}>Moves</span>
          {pokemon.moves.map((move) => (
            <div key={move.name} className={styles.moveRow}>
              <span className={styles.moveName}>{move.name}</span>
              <span className={styles.movePower}>{move.power}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
