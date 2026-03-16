import styles from "./StatusBar.module.css";

export function StatusBar() {
  return (
    <div className={styles.bar}>
      <span className={styles.time}>9:41</span>
      <div className={styles.icons}>
        <span>▂▄▆█</span>
        <span>⏣</span>
        <span>🔋</span>
      </div>
    </div>
  );
}
