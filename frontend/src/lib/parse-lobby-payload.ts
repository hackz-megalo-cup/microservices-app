// Type definitions for Raid Lobby participants
export interface Participant {
  id: string;
  userId: string;
  name: string;
  pokemon: string;
  online: boolean;
}

// Payload interfaces
export interface ParticipantPayload {
  participants: Participant[];
}

export interface BattleStartedPayload {
  battleSessionId: string;
}

export interface ParticipantEventPayload {
  participant: Participant;
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
 * raid.participant_snapshot / raid.user_joined イベント用
 */
export function parseParticipantEvent(payload: string): ParticipantEventPayload | null {
  try {
    const parsed: unknown = JSON.parse(payload);
    if (!isRecord(parsed)) {
      return null;
    }

    if (typeof parsed.participant_id !== "string" || typeof parsed.user_id !== "string") {
      return null;
    }

    const participant: Participant = {
      id: parsed.participant_id,
      userId: parsed.user_id,
      name: typeof parsed.name === "string" ? parsed.name : parsed.user_id,
      pokemon: typeof parsed.pokemon === "string" ? parsed.pokemon : "-",
      online: typeof parsed.online === "boolean" ? parsed.online : true,
    };
    return { participant };
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

    if (typeof parsed.session_id === "string") {
      const result: BattleStartedPayload = { battleSessionId: parsed.session_id };
      return result;
    }

    return null;
  } catch {
    return null;
  }
}
