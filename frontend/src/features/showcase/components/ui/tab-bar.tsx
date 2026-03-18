import { useNavigate } from "react-router";

interface Tab {
  label: string;
  icon: string;
  path: string;
}

const defaultTabs: Tab[] = [
  { label: "HOME", icon: "🏠", path: "/" },
  { label: "BATTLE", icon: "⚔️", path: "/battle/demo" },
  { label: "TEAM", icon: "📋", path: "/collection" },
  { label: "PROFILE", icon: "👤", path: "/profile" },
];

const collectionTabs: Tab[] = [
  { label: "HOME", icon: "🏠", path: "/" },
  { label: "COLLECTION", icon: "📋", path: "/collection" },
  { label: "TEAM", icon: "📋", path: "/collection" },
  { label: "PROFILE", icon: "👤", path: "/profile" },
];

interface TabBarProps {
  active: string;
  variant?: "default" | "collection";
}

export function TabBar({ active, variant = "default" }: TabBarProps) {
  const navigate = useNavigate();
  const tabs = variant === "collection" ? collectionTabs : defaultTabs;

  return (
    <nav className="px-4 pt-2 pb-4">
      <div className="flex items-center justify-around bg-bg-card border border-bg-hover rounded-3xl h-[62px] p-1">
        {tabs.map((tab) => (
          <button
            type="button"
            key={tab.label}
            className={`flex flex-col items-center justify-center gap-0.5 flex-1 h-[54px] bg-transparent text-text-secondary cursor-pointer rounded-[20px] transition-colors ${active === tab.label ? "bg-accent text-bg-primary" : ""}`}
            onClick={() => void navigate(tab.path)}
          >
            <span className="text-lg">{tab.icon}</span>
            <span className="text-[10px] font-bold tracking-wide">{tab.label}</span>
          </button>
        ))}
      </div>
    </nav>
  );
}
