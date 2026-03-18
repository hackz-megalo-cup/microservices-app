import type { Participant } from "../types";

export interface ParticipantPayload {
  participants: Participant[];
}

export interface BattleStartedPayload {
  battleSessionId: string;
}

/**
 * StreamLobbyResponse の payload (JSON文字列) を安全にパース
 * participant_joined / participant_left イベント用
 */
export function parseParticipants(payload: string): ParticipantPayload | null {
  try {
    const parsed = JSON.parse(payload);
    if (Array.isArray(parsed.participants)) {
      return parsed as ParticipantPayload;
    }
    return null;
  } catch {
    return null;
  }
}

/**
 * StreamLobbyResponse の payload (JSON文字列) を安全にパース
 * battle_started イベント用
 */
export function parseBattleStarted(payload: string): BattleStartedPayload | null {
  try {
    const parsed = JSON.parse(payload);
    if (typeof parsed.battleSessionId === "string") {
      return parsed as BattleStartedPayload;
    }
    return null;
  } catch {
    return null;
  }
}
