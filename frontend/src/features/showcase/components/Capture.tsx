import "../styles/global.css";
import styles from "./Capture.module.css";
import { NavBar } from "./ui/NavBar";

export function Capture() {
  return (
    <div className="showcase-screen">
      <NavBar title="CAPTURE" />

      <div className={styles.body}>
        <div className={styles.bossPreview} />
        <div className={styles.nameSection}>
          <span className={styles.bossName}>Python</span>
          <span className={styles.captureRate}>42%</span>
        </div>
        <button type="button" className={styles.throwButton}>
          🎯
        </button>
        <span className={styles.throwHint}>tap to throw</span>
      </div>

      <div className={styles.bottomRow}>
        <button type="button" className={styles.itemCard}>
          Use Item
        </button>
        <button type="button" className={styles.skipCard}>
          Skip
        </button>
      </div>
    </div>
  );
}
