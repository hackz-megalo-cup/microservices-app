import type { Participant } from "../types";

export interface ParticipantPayload {
  participants: Participant[];
}

export interface BattleStartedPayload {
  battleSessionId: string;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isParticipant(value: unknown): value is Participant {
  if (!isRecord(value)) {
    return false;
  }

  return (
    typeof value.id === "string" &&
    typeof value.userId === "string" &&
    typeof value.name === "string" &&
    typeof value.pokemon === "string" &&
    typeof value.online === "boolean"
  );
}

/**
 * StreamLobbyResponse の payload (JSON文字列) を安全にパース
 * participant_joined / participant_left イベント用
 */
export function parseParticipants(payload: string): ParticipantPayload | null {
  try {
    const parsed: unknown = JSON.parse(payload);
    if (!isRecord(parsed)) {
      return null;
    }

    const participants = parsed.participants;
    if (Array.isArray(participants) && participants.every(isParticipant)) {
      const result: ParticipantPayload = { participants };
      return result;
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
    const parsed: unknown = JSON.parse(payload);
    if (!isRecord(parsed)) {
      return null;
    }

    if (typeof parsed.battleSessionId === "string") {
      const result: BattleStartedPayload = { battleSessionId: parsed.battleSessionId };
      return result;
    }

    return null;
  } catch {
    return null;
  }
}
