import { useState } from "react";
import { useNavigate } from "react-router";
import { useAuthContext } from "../context/auth-context";
import "../../showcase/styles/global.css";

export function LoginPage() {
  const { loginAsGuest } = useAuthContext();
  const navigate = useNavigate();
  const [step, setStep] = useState<"initial" | "name-input">("initial");
  const [name, setName] = useState("");
  const [error, setError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleGuestRegister = async () => {
    if (!name.trim()) {
      setError("名前を入力してね");
      return;
    }
    setError("");
    setIsSubmitting(true);
    try {
      await loginAsGuest(name.trim());
      navigate("/");
    } catch (err) {
      setError(err instanceof Error ? err.message : "登録に失敗しました");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="showcase-screen items-center justify-center gap-8 px-6">
      <div className="flex flex-col items-center gap-2">
        <span className="text-5xl">⚡</span>
        <h1 className="text-3xl font-bold text-text-primary">POKEMON RAID</h1>
        <p className="text-sm text-text-secondary">みんなで協力してポケモンを倒そう</p>
      </div>

      {step === "initial" && (
        <button
          type="button"
          onClick={() => setStep("name-input")}
          className="w-full max-w-xs h-14 bg-accent text-bg-primary font-bold text-lg rounded-3xl cursor-pointer hover:opacity-90 transition-opacity"
        >
          ゲスト登録する
        </button>
      )}

      {step === "name-input" && (
        <div className="flex flex-col items-center gap-4 w-full max-w-xs">
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="トレーナー名を入力"
            maxLength={20}
            className="w-full h-14 bg-bg-card text-text-primary text-center text-lg rounded-2xl px-4 border-none outline-none placeholder:text-text-secondary"
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                void handleGuestRegister();
              }
            }}
          />
          {error && <p className="text-red-400 text-sm m-0">{error}</p>}
          <button
            type="button"
            onClick={() => void handleGuestRegister()}
            disabled={isSubmitting}
            className="w-full h-14 bg-accent text-bg-primary font-bold text-lg rounded-3xl cursor-pointer hover:opacity-90 transition-opacity disabled:opacity-50"
          >
            {isSubmitting ? "登録中..." : "はじめる"}
          </button>
        </div>
      )}
    </div>
  );
}
