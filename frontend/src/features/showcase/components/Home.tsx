import type { Raid } from "../types";
import "../styles/global.css";
import styles from "./Home.module.css";
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

      <header className={styles.header}>
        <span className={styles.headerLabel}>POKÉMON</span>
        <button type="button" className={styles.bellButton} aria-label="Notifications">
          🔔
        </button>
      </header>

      <section className={styles.hero}>
        <div className={styles.bossImage} />
        <h1 className={styles.heroTitle}>Python</h1>
        <p className={styles.heroSubtitle}>Dynamic / Interpreted</p>
      </section>

      <section className={styles.raidSection}>
        <span className={styles.raidLabel}>ACTIVE RAIDS</span>
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

      <div className={styles.spacer} />

      <TabBar active="HOME" />
    </div>
  );
}
