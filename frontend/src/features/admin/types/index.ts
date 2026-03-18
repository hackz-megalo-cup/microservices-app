export interface AdminRaid {
  id: string;
  bossName: string;
  currentParticipants: number;
  maxParticipants: number;
  status: string;
  createdAtMs: number | null;
}

export type EffectField = {
  _key: string;
  effectType: string;
  targetType: string;
  captureRateBonus: number;
  flavorText: string;
};
