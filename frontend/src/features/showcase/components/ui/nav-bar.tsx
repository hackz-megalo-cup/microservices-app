import { useNavigate } from "react-router";

interface NavBarProps {
  title: string;
  rightIcon?: "share" | "search" | "heart" | "none";
}

const iconMap: Record<string, string> = {
  share: "↗",
  search: "🔍",
  heart: "♡",
};

export function NavBar({ title, rightIcon = "none" }: NavBarProps) {
  const navigate = useNavigate();

  return (
    <nav className="flex items-center justify-between h-14 px-4">
      <button
        type="button"
        className="flex items-center justify-center w-11 h-11 bg-transparent text-text-secondary text-lg cursor-pointer rounded-lg hover:bg-bg-hover"
        onClick={() => void navigate(-1)}
        aria-label="Go back"
      >
        ←
      </button>
      <span className="text-xs font-bold tracking-widest text-text-secondary uppercase">
        {title}
      </span>
      {rightIcon !== "none" ? (
        <button
          type="button"
          className="flex items-center justify-center w-11 h-11 bg-transparent text-text-secondary text-lg cursor-pointer rounded-lg hover:bg-bg-hover"
          aria-label={rightIcon}
        >
          {iconMap[rightIcon]}
        </button>
      ) : (
        <div className="w-11 h-11" />
      )}
    </nav>
  );
}
