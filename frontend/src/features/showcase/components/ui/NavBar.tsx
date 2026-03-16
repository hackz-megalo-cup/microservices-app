import { useNavigate } from "react-router";
import styles from "./NavBar.module.css";

interface NavBarProps {
  title: string;
  rightIcon?: "share" | "search" | "heart" | "none";
}

const iconMap: Record<string, string> = {
  share: "↗",
  search: "🔍",
  heart: "♡",
};

export function NavBar({ title, rightIcon = "none" }: NavBarProps) {
  const navigate = useNavigate();

  return (
    <nav className={styles.bar}>
      <button
        type="button"
        className={styles.iconButton}
        onClick={() => void navigate(-1)}
        aria-label="Go back"
      >
        ←
      </button>
      <span className={styles.title}>{title}</span>
      {rightIcon !== "none" ? (
        <button type="button" className={styles.iconButton} aria-label={rightIcon}>
          {iconMap[rightIcon]}
        </button>
      ) : (
        <div className={styles.spacer} />
      )}
    </nav>
  );
}
