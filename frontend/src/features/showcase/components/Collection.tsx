import { useNavigate } from "react-router";
import type { Pokemon } from "../types";
import "../styles/global.css";
import styles from "./Collection.module.css";
import { NavBar } from "./ui/NavBar";
import { TabBar } from "./ui/TabBar";

const mockCollection: Pokemon[] = [
  {
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
    about: "A statically typed superset of JavaScript that compiles to clean, readable code.",
    moves: [
      { name: "Type Check", type: "Normal", power: 45 },
      { name: "Compile", type: "Normal", power: 60 },
      { name: "Refactor", type: "Normal", power: 80 },
    ],
    captured: true,
  },
  {
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
    about: "A systems programming language focused on safety and performance.",
    moves: [
      { name: "Borrow Check", type: "Normal", power: 50 },
      { name: "Unsafe Block", type: "Normal", power: 90 },
      { name: "Pattern Match", type: "Normal", power: 65 },
    ],
    captured: true,
  },
  {
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
    about: "A versatile interpreted language known for readability and simplicity.",
    moves: [
      { name: "List Comprehension", type: "Normal", power: 55 },
      { name: "Decorator", type: "Normal", power: 70 },
      { name: "GIL Release", type: "Normal", power: 85 },
    ],
    captured: true,
  },
  {
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
    about: "A compiled language designed for simplicity and concurrency.",
    moves: [
      { name: "Goroutine", type: "Normal", power: 60 },
      { name: "Channel Send", type: "Normal", power: 55 },
      { name: "Defer", type: "Normal", power: 40 },
    ],
    captured: true,
  },
  {
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
    about: "The foundational systems language powering operating systems worldwide.",
    moves: [
      { name: "Pointer Arithmetic", type: "Normal", power: 80 },
      { name: "Malloc", type: "Normal", power: 70 },
      { name: "Segfault", type: "Normal", power: 100 },
    ],
    captured: true,
  },
  {
    id: "6",
    name: "???",
    number: "#006",
    image: "",
    types: [],
    stats: [],
    about: "",
    moves: [],
    captured: false,
  },
];

export function Collection() {
  const navigate = useNavigate();
  const capturedCount = mockCollection.filter((p) => p.captured).length;

  return (
    <div className="showcase-screen">
      <NavBar title="COLLECTION" rightIcon="search" />

      <div className={styles.hero}>
        <span className={styles.count}>{capturedCount}</span>
        <span className={styles.total}>/ {mockCollection.length} captured</span>
      </div>

      <div className={styles.grid}>
        {mockCollection.map((pokemon) =>
          pokemon.captured ? (
            <button
              type="button"
              key={pokemon.id}
              className={styles.card}
              onClick={() => void navigate(`/collection/${pokemon.id}`)}
            >
              <div className={styles.cardImage} />
              <span className={styles.cardName}>{pokemon.name}</span>
            </button>
          ) : (
            <div key={pokemon.id} className={`${styles.card} ${styles.locked}`}>
              <span className={styles.lockIcon}>🔒</span>
              <span className={styles.cardName}>{pokemon.name}</span>
            </div>
          ),
        )}
      </div>

      <div className={styles.spacer} />

      <TabBar active="TEAM" />
    </div>
  );
}
