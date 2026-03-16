import { useNavigate } from "react-router";
import "../styles/global.css";
import styles from "./Victory.module.css";

const rewards = [
  { icon: "⚡", value: "+500", label: "XP" },
  { icon: "💎", value: "+3", label: "Gems" },
  { icon: "🎟️", value: "+1", label: "Ticket" },
] as const;

export function Victory() {
  const navigate = useNavigate();

  return (
    <div className={`showcase-screen ${styles.screen}`}>
      <div className={styles.trophyFrame} />
      <span className={styles.clearLabel}>RAID CLEAR</span>
      <h1 className={styles.title}>Victory</h1>
      <p className={styles.subtitle}>Python defeated in 2:31</p>

      <div className={styles.rewardsCard}>
        <span className={styles.rewardsTitle}>Rewards</span>
        <div className={styles.rewardsRow}>
          {rewards.map((reward) => (
            <div key={reward.label} className={styles.rewardItem}>
              <span className={styles.rewardIcon}>{reward.icon}</span>
              <span className={styles.rewardValue}>{reward.value}</span>
              <span className={styles.rewardLabel}>{reward.label}</span>
            </div>
          ))}
        </div>
      </div>

      <div className={styles.mvpRow}>
        <span className={styles.mvpText}>MVP — RustLover42</span>
      </div>

      <button
        type="button"
        className={styles.captureButton}
        onClick={() => void navigate("/capture/1")}
      >
        CAPTURE
      </button>
    </div>
  );
}
