import { useNavigate } from "react-router";

interface DashboardCard {
  label: string;
  icon: string;
  description: string;
  path: string;
}

const CARDS: DashboardCard[] = [
  {
    label: "Pokemon",
    icon: "🐉",
    description: "Pokemon のマスターデータを管理",
    path: "/admin/pokemon",
  },
  {
    label: "アイテム",
    icon: "🎒",
    description: "アイテムのマスターデータを管理",
    path: "/admin/items",
  },
  {
    label: "タイプ相性",
    icon: "⚔️",
    description: "タイプ相性テーブルを管理",
    path: "/admin/type-chart",
  },
  {
    label: "レイド",
    icon: "🏟️",
    description: "レイドイベントを管理",
    path: "/admin/raids",
  },
];

export function AdminDashboard() {
  const navigate = useNavigate();

  return (
    <div className="p-8">
      <header className="mb-8">
        <h1 className="text-2xl font-bold text-text-primary">管理ダッシュボード</h1>
        <p className="text-sm text-text-secondary mt-1">マスターデータの管理・更新</p>
      </header>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-2">
        {CARDS.map((card) => (
          <button
            key={card.path}
            type="button"
            onClick={() => void navigate(card.path)}
            className="flex items-start gap-4 p-6 bg-bg-card border border-bg-hover rounded-2xl text-left cursor-pointer hover:border-accent transition-colors shadow-card"
          >
            <span className="text-3xl">{card.icon}</span>
            <div className="flex flex-col gap-1">
              <span className="text-base font-semibold text-text-primary">{card.label}</span>
              <span className="text-sm text-text-secondary">{card.description}</span>
            </div>
          </button>
        ))}
      </div>
    </div>
  );
}
