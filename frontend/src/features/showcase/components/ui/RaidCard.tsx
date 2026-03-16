import { useNavigate } from "react-router";
import styles from "./RaidCard.module.css";

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
    <button type="button" className={styles.card} onClick={() => void navigate(`/lobby/${id}`)}>
      <div className={styles.thumbnail}>
        {image ? (
          <img src={image} alt={name} className={styles.thumbImg} />
        ) : (
          <div className={styles.thumbPlaceholder} />
        )}
      </div>
      <div className={styles.info}>
        <span className={styles.name}>{name}</span>
        <span className={styles.meta}>{type}</span>
        <div className={styles.row}>
          <span className={styles.difficulty}>{difficulty}</span>
          <span className={styles.timer}>{timer}</span>
        </div>
      </div>
      <div className={styles.joinBadge}>Join</div>
    </button>
  );
}
