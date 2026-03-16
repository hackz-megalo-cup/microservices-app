import { useNavigate } from "react-router";

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
