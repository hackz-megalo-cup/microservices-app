import { useNavigate } from "react-router";
import type { Pokemon } from "../types";
import "../styles/global.css";
import { NavBar } from "./ui/nav-bar";
import { TabBar } from "./ui/tab-bar";

const mockCollection: Pokemon[] = [
  {
    id: "1",
    name: "TypeScript",
    number: "#001",
    image: "/images/collection-typescript.png",
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
    image: "/images/collection-rust.png",
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
    image: "/images/collection-python.png",
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
    image: "/images/collection-go.png",
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
    image: "/images/collection-c.png",
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
    name: "Java",
    number: "#006",
    image: "",
    types: ["Static", "OOP"],
    stats: [
      { label: "HP", value: 90 },
      { label: "ATK", value: 60 },
      { label: "SPD", value: 50 },
    ],
    about: "A widely-used object-oriented language.",
    moves: [
      { name: "Garbage Collect", type: "Normal", power: 55 },
      { name: "Abstract Factory", type: "Normal", power: 65 },
      { name: "NullPointer", type: "Normal", power: 75 },
    ],
    captured: true,
  },
  {
    id: "7",
    name: "???",
    number: "#007",
    image: "",
    types: [],
    stats: [],
    about: "",
    moves: [],
    captured: false,
  },
  {
    id: "8",
    name: "???",
    number: "#008",
    image: "",
    types: [],
    stats: [],
    about: "",
    moves: [],
    captured: false,
  },
  {
    id: "9",
    name: "???",
    number: "#009",
    image: "",
    types: [],
    stats: [],
    about: "",
    moves: [],
    captured: false,
  },
  {
    id: "10",
    name: "???",
    number: "#010",
    image: "",
    types: [],
    stats: [],
    about: "",
    moves: [],
    captured: false,
  },
  {
    id: "11",
    name: "???",
    number: "#011",
    image: "",
    types: [],
    stats: [],
    about: "",
    moves: [],
    captured: false,
  },
  {
    id: "12",
    name: "???",
    number: "#012",
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

      <div className="flex items-end gap-2 py-2 px-6">
        <span className="text-5xl font-bold leading-none text-text-primary">{capturedCount}</span>
        <span className="text-base text-text-secondary pb-1">
          / {mockCollection.length} captured
        </span>
      </div>

      <div className="grid grid-cols-2 gap-3 px-4 py-1 flex-1">
        {mockCollection.map((pokemon) =>
          pokemon.captured ? (
            <button
              type="button"
              key={pokemon.id}
              className="flex flex-col items-center justify-end gap-1 bg-bg-card rounded-2xl h-32 pb-3 cursor-pointer overflow-hidden border-none text-text-primary hover:bg-bg-hover"
              onClick={() => void navigate(`/collection/${pokemon.id}`)}
            >
              <div className="flex-1 flex items-center justify-center w-full overflow-hidden">
                {pokemon.image ? (
                  <img
                    src={pokemon.image}
                    alt={pokemon.name}
                    className="w-full h-20 object-cover"
                  />
                ) : null}
              </div>
              <span className="text-xs font-semibold">{pokemon.name}</span>
            </button>
          ) : (
            <div
              key={pokemon.id}
              className="flex flex-col items-center justify-center gap-1 bg-locked rounded-2xl h-32 pb-3 overflow-hidden border-none text-text-primary cursor-default hover:bg-locked"
            >
              <span className="text-2xl opacity-50">🔒</span>
              <span className="text-xs font-semibold">{pokemon.name}</span>
            </div>
          ),
        )}
      </div>

      <div className="flex-1" />

      <TabBar active="COLLECTION" variant="collection" />
    </div>
  );
}
