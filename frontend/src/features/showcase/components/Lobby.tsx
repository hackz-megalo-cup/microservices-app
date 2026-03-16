import type { Trainer } from "../types";
import "../styles/global.css";
import styles from "./Lobby.module.css";
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

      <div className={styles.content}>
        <section className={styles.bossSection}>
          <div className={styles.bossPreview} />
          <div className={styles.nameRow}>
            <span className={styles.bossName}>Python</span>
            <span className={styles.typeBadge}>Dynamic</span>
          </div>
        </section>

        <div className={styles.qrCard}>
          <div className={styles.qrPlaceholder}>QR</div>
          <div className={styles.codeRow}>
            <span className={styles.code}>RAID-7X4K</span>
            <button type="button" className={styles.copyButton} aria-label="Copy code">
              📋
            </button>
          </div>
        </div>

        <section className={styles.trainersSection}>
          <div className={styles.trainersHeader}>
            <span className={styles.trainersLabel}>TRAINERS</span>
            <span className={styles.trainersCount}>4/6</span>
          </div>
          {mockTrainers.map((trainer) => (
            <div key={trainer.name} className={styles.trainerRow}>
              <div className={styles.trainerAvatar} />
              <span className={styles.trainerName}>{trainer.name}</span>
              <span className={styles.trainerPokemon}>{trainer.pokemon}</span>
              <div
                className={`${styles.statusDot} ${trainer.online ? styles.statusOnline : styles.statusOffline}`}
              />
            </div>
          ))}
        </section>

        <div className={styles.spacer} />

        <button type="button" className={styles.startButton}>
          ENTER RAID
        </button>
      </div>
    </div>
  );
}
