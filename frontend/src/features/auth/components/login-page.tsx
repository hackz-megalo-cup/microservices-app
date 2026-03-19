import { useState } from "react";
import { useNavigate } from "react-router";
import { useAuthContext } from "../hooks/use-auth-context";
import "../../../styles/global.css";

type Step = "initial" | "guest-name" | "login" | "register";

function getErrorMessage(err: unknown): string {
  if (err instanceof Error) {
    const msg = err.message.toLowerCase();
    if (msg.includes("already exists") || msg.includes("duplicate") || msg.includes("already registered")) {
      return "このメールアドレスは既に登録済みです";
    }
    if (msg.includes("invalid") || msg.includes("incorrect") || msg.includes("wrong") || msg.includes("unauthenticated")) {
      return "メールアドレスまたはパスワードが間違っています";
    }
    if (msg.includes("network") || msg.includes("fetch") || msg.includes("connect")) {
      return "ネットワークエラーが発生しました";
    }
    return err.message;
  }
  return "エラーが発生しました";
}

function validateEmail(email: string): string {
  if (!email.trim()) return "メールアドレスを入力してください";
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) return "正しいメールアドレスを入力してください";
  return "";
}

function validatePassword(password: string): string {
  if (!password) return "パスワードを入力してください";
  if (password.length < 8) return "パスワードは8文字以上で入力してください";
  return "";
}

function validateName(name: string): string {
  if (!name.trim()) return "名前を入力してください";
  return "";
}

export function LoginPage() {
  const { login, register, loginAsGuest } = useAuthContext();
  const navigate = useNavigate();
  const [step, setStep] = useState<Step>("initial");
  const [error, setError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Guest name form
  const [guestName, setGuestName] = useState("");

  // Login form
  const [loginEmail, setLoginEmail] = useState("");
  const [loginPassword, setLoginPassword] = useState("");

  // Register form
  const [registerEmail, setRegisterEmail] = useState("");
  const [registerPassword, setRegisterPassword] = useState("");
  const [registerName, setRegisterName] = useState("");

  const handleGuestSubmit = async () => {
    const nameErr = validateName(guestName);
    if (nameErr) { setError(nameErr); return; }
    setError("");
    setIsSubmitting(true);
    try {
      await loginAsGuest(guestName.trim());
      navigate("/");
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleLoginSubmit = async () => {
    const emailErr = validateEmail(loginEmail);
    if (emailErr) { setError(emailErr); return; }
    const passErr = validatePassword(loginPassword);
    if (passErr) { setError(passErr); return; }
    setError("");
    setIsSubmitting(true);
    try {
      await login(loginEmail.trim(), loginPassword);
      navigate("/");
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleRegisterSubmit = async () => {
    const emailErr = validateEmail(registerEmail);
    if (emailErr) { setError(emailErr); return; }
    const passErr = validatePassword(registerPassword);
    if (passErr) { setError(passErr); return; }
    const nameErr = validateName(registerName);
    if (nameErr) { setError(nameErr); return; }
    setError("");
    setIsSubmitting(true);
    try {
      await register(registerEmail.trim(), registerPassword, registerName.trim());
      navigate("/");
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setIsSubmitting(false);
    }
  };

  const goTo = (next: Step) => {
    setError("");
    setStep(next);
  };

  return (
    <div className="showcase-screen items-center justify-center gap-8 px-6">
      <div className="flex flex-col items-center gap-2">
        <span className="text-5xl">⚡</span>
        <h1 className="text-3xl font-bold text-text-primary">POKEMON RAID</h1>
        <p className="text-sm text-text-secondary">みんなで協力してポケモンを倒そう</p>
      </div>

      {step === "initial" && (
        <div className="flex flex-col items-center gap-4 w-full max-w-xs">
          <button
            type="button"
            onClick={() => goTo("guest-name")}
            className="w-full h-14 bg-accent text-bg-primary font-bold text-lg rounded-3xl cursor-pointer hover:opacity-90 transition-opacity"
          >
            ゲストでプレイ
          </button>
          <button
            type="button"
            onClick={() => goTo("login")}
            className="w-full h-14 bg-bg-card text-text-primary font-bold text-lg rounded-3xl cursor-pointer hover:opacity-90 transition-opacity border border-accent"
          >
            ログイン
          </button>
          <button
            type="button"
            onClick={() => goTo("register")}
            className="w-full h-14 bg-bg-card text-text-primary font-bold text-lg rounded-3xl cursor-pointer hover:opacity-90 transition-opacity border border-text-secondary"
          >
            新規登録
          </button>
        </div>
      )}

      {step === "guest-name" && (
        <div className="flex flex-col items-center gap-4 w-full max-w-xs">
          <input
            value={guestName}
            onChange={(e) => setGuestName(e.target.value)}
            placeholder="トレーナー名を入力"
            maxLength={20}
            className="w-full h-14 bg-bg-card text-text-primary text-center text-lg rounded-2xl px-4 border-none outline-none placeholder:text-text-secondary"
            onKeyDown={(e) => { if (e.key === "Enter") { void handleGuestSubmit(); } }}
          />
          {error && <p className="text-red-400 text-sm m-0">{error}</p>}
          <button
            type="button"
            onClick={() => void handleGuestSubmit()}
            disabled={isSubmitting}
            className="w-full h-14 bg-accent text-bg-primary font-bold text-lg rounded-3xl cursor-pointer hover:opacity-90 transition-opacity disabled:opacity-50"
          >
            {isSubmitting ? "登録中..." : "はじめる"}
          </button>
          <button
            type="button"
            onClick={() => goTo("initial")}
            className="text-text-secondary text-sm cursor-pointer hover:opacity-70 transition-opacity bg-transparent border-none"
          >
            ← 戻る
          </button>
        </div>
      )}

      {step === "login" && (
        <div className="flex flex-col items-center gap-4 w-full max-w-xs">
          <input
            type="email"
            value={loginEmail}
            onChange={(e) => setLoginEmail(e.target.value)}
            placeholder="メールアドレス"
            className="w-full h-14 bg-bg-card text-text-primary text-lg rounded-2xl px-4 border-none outline-none placeholder:text-text-secondary"
          />
          <input
            type="password"
            value={loginPassword}
            onChange={(e) => setLoginPassword(e.target.value)}
            placeholder="パスワード"
            className="w-full h-14 bg-bg-card text-text-primary text-lg rounded-2xl px-4 border-none outline-none placeholder:text-text-secondary"
            onKeyDown={(e) => { if (e.key === "Enter") { void handleLoginSubmit(); } }}
          />
          {error && <p className="text-red-400 text-sm m-0">{error}</p>}
          <button
            type="button"
            onClick={() => void handleLoginSubmit()}
            disabled={isSubmitting}
            className="w-full h-14 bg-accent text-bg-primary font-bold text-lg rounded-3xl cursor-pointer hover:opacity-90 transition-opacity disabled:opacity-50"
          >
            {isSubmitting ? "ログイン中..." : "ログイン"}
          </button>
          <button
            type="button"
            onClick={() => goTo("register")}
            className="text-text-secondary text-sm cursor-pointer hover:opacity-70 transition-opacity bg-transparent border-none"
          >
            アカウントをお持ちでない方は新規登録
          </button>
          <button
            type="button"
            onClick={() => goTo("initial")}
            className="text-text-secondary text-sm cursor-pointer hover:opacity-70 transition-opacity bg-transparent border-none"
          >
            ← 戻る
          </button>
        </div>
      )}

      {step === "register" && (
        <div className="flex flex-col items-center gap-4 w-full max-w-xs">
          <input
            type="email"
            value={registerEmail}
            onChange={(e) => setRegisterEmail(e.target.value)}
            placeholder="メールアドレス"
            className="w-full h-14 bg-bg-card text-text-primary text-lg rounded-2xl px-4 border-none outline-none placeholder:text-text-secondary"
          />
          <input
            type="password"
            value={registerPassword}
            onChange={(e) => setRegisterPassword(e.target.value)}
            placeholder="パスワード（8文字以上）"
            className="w-full h-14 bg-bg-card text-text-primary text-lg rounded-2xl px-4 border-none outline-none placeholder:text-text-secondary"
          />
          <input
            value={registerName}
            onChange={(e) => setRegisterName(e.target.value)}
            placeholder="トレーナー名"
            maxLength={20}
            className="w-full h-14 bg-bg-card text-text-primary text-lg rounded-2xl px-4 border-none outline-none placeholder:text-text-secondary"
            onKeyDown={(e) => { if (e.key === "Enter") { void handleRegisterSubmit(); } }}
          />
          {error && <p className="text-red-400 text-sm m-0">{error}</p>}
          <button
            type="button"
            onClick={() => void handleRegisterSubmit()}
            disabled={isSubmitting}
            className="w-full h-14 bg-accent text-bg-primary font-bold text-lg rounded-3xl cursor-pointer hover:opacity-90 transition-opacity disabled:opacity-50"
          >
            {isSubmitting ? "登録中..." : "登録する"}
          </button>
          <button
            type="button"
            onClick={() => goTo("login")}
            className="text-text-secondary text-sm cursor-pointer hover:opacity-70 transition-opacity bg-transparent border-none"
          >
            既にアカウントをお持ちの方はログイン
          </button>
          <button
            type="button"
            onClick={() => goTo("initial")}
            className="text-text-secondary text-sm cursor-pointer hover:opacity-70 transition-opacity bg-transparent border-none"
          >
            ← 戻る
          </button>
        </div>
      )}
    </div>
  );
}
