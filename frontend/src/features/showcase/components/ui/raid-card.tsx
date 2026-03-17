import { useNavigate } from "react-router";

interface RaidCardProps {
  id: string;
  name: string;
  type: string;
  players: string;
  timer: string;
  image: string;
}

export function RaidCard({ id, name, type, players, timer, image }: RaidCardProps) {
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
        <div className="flex gap-3 items-center">
          <span className="flex items-center gap-1 text-xs text-text-secondary">
            <svg
              className="w-3 h-3"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              aria-label="Players"
              role="img"
            >
              <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
              <circle cx="9" cy="7" r="4" />
              <path d="M22 21v-2a4 4 0 0 0-3-3.87" />
              <path d="M16 3.13a4 4 0 0 1 0 7.75" />
            </svg>
            {players}
          </span>
          <span className="flex items-center gap-1 text-xs text-text-secondary">
            <svg
              className="w-3 h-3"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              aria-label="Timer"
              role="img"
            >
              <circle cx="12" cy="12" r="10" />
              <polyline points="12 6 12 12 16 14" />
            </svg>
            {timer}
          </span>
        </div>
      </div>
      <div className="shrink-0 bg-accent text-bg-primary text-xs font-bold rounded-3xl px-4 py-2">
        Join
      </div>
    </button>
  );
}
