import "../../../styles/global.css";
import { useAuthContext } from "../../../lib/auth";
import { useUserProfile } from "../../auth/hooks/use-user-profile";
import { TabBar } from "./ui/tab-bar";

export function Profile() {
  const { user } = useAuthContext();
  const userId = user?.id ?? "";

  const { profile, isLoading, error } = useUserProfile(userId);

  if (!userId) {
    return (
      <div className="showcase-screen">
        <header className="flex items-center justify-between px-6 py-3">
          <span className="text-xs font-bold tracking-widest text-text-secondary">PROFILE</span>
        </header>
        <div className="flex items-center justify-center flex-1">
          <p className="text-text-secondary">ユーザー情報を取得できません</p>
        </div>
        <TabBar active="PROFILE" />
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="showcase-screen">
        <header className="flex items-center justify-between px-6 py-3">
          <span className="text-xs font-bold tracking-widest text-text-secondary">PROFILE</span>
        </header>
        <div className="flex items-center justify-center flex-1">
          <div className="flex flex-col items-center gap-3">
            <div className="w-12 h-12 rounded-full bg-bg-hover animate-pulse" />
            <p className="text-text-secondary">読み込み中...</p>
          </div>
        </div>
        <TabBar active="PROFILE" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="showcase-screen">
        <header className="flex items-center justify-between px-6 py-3">
          <span className="text-xs font-bold tracking-widest text-text-secondary">PROFILE</span>
        </header>
        <div className="flex items-center justify-center flex-1">
          <p className="text-text-secondary">プロフィール取得に失敗しました</p>
        </div>
        <TabBar active="PROFILE" />
      </div>
    );
  }

  return (
    <div className="showcase-screen">
      <header className="flex items-center justify-between px-6 py-3">
        <span className="text-xs font-bold tracking-widest text-text-secondary">PROFILE</span>
      </header>

      <section
        className="flex flex-col items-center justify-center gap-3 h-[200px]"
        style={{ background: "radial-gradient(circle, var(--color-accent-glow), transparent)" }}
      >
        <div className="flex items-center justify-center w-[100px] h-[100px] rounded-full bg-bg-card border border-bg-hover text-5xl">
          👤
        </div>
        <h1 className="text-2xl font-bold text-text-primary m-0">
          {profile?.name ?? user?.name ?? "ゲスト"}
        </h1>
      </section>

      <section className="flex flex-col gap-4 p-4">
        <div className="bg-bg-card rounded-2xl p-4">
          <h2 className="text-lg font-bold text-text-primary mb-3">ユーザー情報</h2>
          <div className="flex flex-col gap-3">
            <div className="flex justify-between items-center">
              <span className="text-sm text-text-secondary">トレーナー名</span>
              <span className="text-sm font-bold text-text-primary">{profile?.name}</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-sm text-text-secondary">メールアドレス</span>
              <span className="text-sm font-bold text-text-primary">{profile?.email}</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-sm text-text-secondary">ロール</span>
              <span className="text-sm font-bold text-text-primary">{profile?.role}</span>
            </div>
            {profile?.createdAt && (
              <div className="flex justify-between items-center">
                <span className="text-sm text-text-secondary">アカウント作成</span>
                <span className="text-sm font-bold text-text-primary">{profile.createdAt}</span>
              </div>
            )}
          </div>
        </div>

        <div className="bg-bg-card rounded-2xl p-4">
          <h2 className="text-lg font-bold text-text-primary mb-3">設定</h2>
          <div className="flex flex-col gap-2">
            <button
              type="button"
              className="w-full h-12 bg-bg-hover rounded-lg text-text-primary font-bold cursor-pointer hover:opacity-80"
            >
              パスワード変更
            </button>
          </div>
        </div>
      </section>

      <div className="flex-1" />
      <TabBar active="PROFILE" />
    </div>
  );
}
