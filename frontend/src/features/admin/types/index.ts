export interface AdminRaid {
  id: string;
  bossName: string;
  currentParticipants: number;
  maxParticipants: number;
  status: string;
  createdAtMs: number | null;
}
