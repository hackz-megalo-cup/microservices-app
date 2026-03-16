import { useNavigate } from "react-router";
import styles from "./TabBar.module.css";

interface TabBarProps {
  active: string;
}

const tabs = [
  { label: "HOME", icon: "🏠", path: "/" },
  { label: "BATTLE", icon: "⚔️", path: "/battle/demo" },
  { label: "TEAM", icon: "📋", path: "/collection" },
  { label: "PROFILE", icon: "👤", path: "/api-test" },
] as const;

export function TabBar({ active }: TabBarProps) {
  const navigate = useNavigate();

  return (
    <nav className={styles.container}>
      <div className={styles.pill}>
        {tabs.map((tab) => (
          <button
            type="button"
            key={tab.label}
            className={`${styles.tab} ${active === tab.label ? styles.active : ""}`}
            onClick={() => void navigate(tab.path)}
          >
            <span className={styles.icon}>{tab.icon}</span>
            <span className={styles.label}>{tab.label}</span>
          </button>
        ))}
      </div>
    </nav>
  );
}
