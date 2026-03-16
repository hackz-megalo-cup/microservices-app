import "../styles/global.css";
import styles from "./Battle.module.css";
import { StatusBar } from "./ui/StatusBar";

export function Battle() {
  return (
    <div className="showcase-screen">
      <StatusBar />

      <section className={styles.topSection}>
        <div className={styles.nameRow}>
          <span className={styles.bossName}>Python</span>
          <span className={styles.timerBadge}>2:34</span>
        </div>
        <div className={styles.hpBarContainer}>
          <div className={styles.hpBarFill} />
        </div>
      </section>

      <div className={styles.bossArea}>
        <div className={styles.bossModel} />
        <span className={styles.damagePrimary}>-342</span>
        <span className={styles.damageSecondary}>-128</span>
      </div>

      <section className={styles.bottomSection}>
        <div className={styles.participantDots}>
          <div className={`${styles.dot} ${styles.dotAccent}`} />
          <div className={`${styles.dot} ${styles.dotGreen}`} />
          <div className={`${styles.dot} ${styles.dotSecondary}`} />
          <div className={`${styles.dot} ${styles.dotSecondary}`} />
        </div>
        <div className={styles.chargeBarContainer}>
          <div className={styles.chargeBarFill} />
        </div>
        <button type="button" className={styles.attackButton}>
          ATTACK
        </button>
        <span className={styles.hint}>tap to attack</span>
      </section>
    </div>
  );
}
