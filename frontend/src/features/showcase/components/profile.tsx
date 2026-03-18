import "../../../styles/global.css";
import { useAuthContext } from "../../auth/hooks/use-auth-context";
import { TabBar } from "./ui/tab-bar";

export function Profile() {
  const { user } = useAuthContext();

  return (
    <div className="showcase-screen">
      <header className="flex items-center justify-between px-6 py-3">
        <span className="text-xs font-bold tracking-widest text-text-secondary">PROFILE</span>
      </header>

      <section
        className="flex flex-col items-center justify-center gap-3 h-[280px]"
        style={{ background: "radial-gradient(circle, var(--color-accent-glow), transparent)" }}
      >
        <div className="flex items-center justify-center w-[100px] h-[100px] rounded-full bg-bg-card border border-bg-hover text-5xl">
          👤
        </div>
        <h1 className="text-2xl font-bold text-text-primary m-0">{user?.name ?? "ゲスト"}</h1>
      </section>

      <div className="flex-1" />

      <TabBar active="PROFILE" />
    </div>
  );
}
