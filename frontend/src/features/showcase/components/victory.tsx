import { useNavigate } from "react-router";
import "../styles/global.css";

const rewards = [
  { value: "+350", label: "EXP" },
  { value: "x2", label: "Red Bull" },
  { value: "x1", label: "Cushion" },
] as const;

export function Victory() {
  const navigate = useNavigate();

  return (
    <div className="showcase-screen items-center gap-8 pt-12 px-6 pb-6">
      <img
        src="/images/victory-trophy.png"
        alt="Trophy"
        className="w-[120px] h-[120px] rounded-full object-cover"
      />
      <span className="text-xs font-bold tracking-widest text-accent">RAID CLEAR</span>
      <h1 className="text-4xl font-bold text-text-primary m-0">Victory</h1>
      <p className="text-sm text-text-secondary m-0">Python defeated in 2:31</p>

      <div className="flex flex-col gap-4 bg-bg-card rounded-2xl p-5 w-full">
        <span className="text-sm font-bold text-text-primary">Rewards</span>
        <div className="flex gap-3">
          {rewards.map((reward) => (
            <div
              key={reward.label}
              className="flex-1 flex flex-col items-center gap-1 bg-bg-primary rounded-lg p-4"
            >
              <span className="text-xl font-bold text-accent">{reward.value}</span>
              <span className="text-xs text-text-secondary">{reward.label}</span>
            </div>
          ))}
        </div>
      </div>

      <div className="flex items-center px-5 py-4 bg-bg-card rounded-2xl border-l-4 border-accent w-full">
        <span className="text-sm font-bold text-text-primary">MVP — RustLover42</span>
      </div>

      <button
        type="button"
        className="w-full h-14 bg-accent rounded-3xl text-base font-bold text-bg-primary cursor-pointer border-none hover:opacity-90"
        onClick={() => void navigate("/capture/1")}
      >
        CAPTURE
      </button>
    </div>
  );
}
