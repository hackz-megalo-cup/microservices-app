import { useNavigate } from "react-router";

interface RaidCardProps {
  id: string;
  name: string;
  type: string;
  difficulty: string;
  timer: string;
  image: string;
}

export function RaidCard({ id, name, type, difficulty, timer, image }: RaidCardProps) {
  const navigate = useNavigate();

  return (
    <button
      type="button"
      className="flex items-center gap-3 bg-bg-card rounded-2xl p-4 shadow-card w-full cursor-pointer text-left transition-colors hover:bg-bg-hover"
      onClick={() => void navigate(`/lobby/${id}`)}
    >
      <div className="shrink-0 w-14 h-14 rounded-lg overflow-hidden bg-bg-hover">
        {image ? (
          <img src={image} alt={name} className="w-full h-full object-cover" />
        ) : (
          <div className="w-full h-full bg-bg-hover" />
        )}
      </div>
      <div className="flex flex-col gap-0.5 flex-1 min-w-0">
        <span className="text-base font-bold text-text-primary">{name}</span>
        <span className="text-xs text-text-secondary">{type}</span>
        <div className="flex gap-2 items-center">
          <span className="text-xs text-text-secondary">{difficulty}</span>
          <span className="text-xs text-accent font-semibold">{timer}</span>
        </div>
      </div>
      <div className="shrink-0 bg-accent text-bg-primary text-xs font-bold rounded-3xl px-4 py-2">
        Join
      </div>
    </button>
  );
}
