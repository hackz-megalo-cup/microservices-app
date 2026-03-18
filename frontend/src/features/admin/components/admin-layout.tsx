import "../../../styles/global.css";
import { NavLink, Outlet } from "react-router";

interface NavItem {
  label: string;
  icon: string;
  path: string;
}

const NAV_ITEMS: NavItem[] = [
  { label: "ダッシュボード", icon: "🏠", path: "/admin" },
  { label: "Pokemon", icon: "🐉", path: "/admin/pokemon" },
  { label: "アイテム", icon: "🎒", path: "/admin/items" },
  { label: "タイプ相性", icon: "⚔️", path: "/admin/type-chart" },
  { label: "レイド", icon: "🏟️", path: "/admin/raids" },
];

export function AdminLayout() {
  return (
    <div className="showcase-screen flex-row">
      <aside className="w-56 min-h-screen bg-bg-card border-r border-bg-hover flex flex-col shrink-0">
        <div className="px-6 py-5 border-b border-bg-hover">
          <span className="text-xs font-bold tracking-widest text-text-secondary">ADMIN</span>
        </div>
        <nav className="flex flex-col gap-1 p-3 flex-1">
          {NAV_ITEMS.map((item) => (
            <NavLink
              key={item.path}
              to={item.path}
              end={item.path === "/admin"}
              className={({ isActive }) =>
                `flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-medium transition-colors ${
                  isActive
                    ? "bg-accent text-bg-primary"
                    : "text-text-secondary hover:bg-bg-hover hover:text-text-primary"
                }`
              }
            >
              <span className="text-base">{item.icon}</span>
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div className="p-3 border-t border-bg-hover">
          <NavLink
            to="/"
            className="flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-medium text-text-secondary hover:bg-bg-hover hover:text-text-primary transition-colors"
          >
            <span className="text-base">←</span>
            ゲームに戻る
          </NavLink>
        </div>
      </aside>
      <main className="flex-1 overflow-auto">
        <Outlet />
      </main>
    </div>
  );
}
