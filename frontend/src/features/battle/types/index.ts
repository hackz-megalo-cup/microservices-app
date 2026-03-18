export type ConnectionState = "disconnected" | "connecting" | "connected" | "error";

// --- Server → Client messages ---

export interface JoinedMessage {
  t: "joined";
  sessionId: string;
  bossHp: number;
  bossMaxHp: number;
  participants: string[];
  timeoutSec: number;
}

export interface HpMessage {
  t: "hp";
  hp: number;
  maxHp: number;
  lastDmg: number;
  by: string;
}

export interface SpecialUsedMessage {
  t: "special_used";
  userId: string;
  moveName: string;
  dmg: number;
  bossHp: number;
}

export interface FinishedMessage {
  t: "finished";
  result: "win" | "timeout";
  bossHp: number;
  elapsed: number;
}

export interface TimeSyncMessage {
  t: "time_sync";
  remainingSec: number;
}

export type ServerMessage =
  | JoinedMessage
  | HpMessage
  | SpecialUsedMessage
  | FinishedMessage
  | TimeSyncMessage;

// --- Client → Server messages ---

export interface JoinCommand {
  t: "join";
  userId: string;
}

export interface TapCommand {
  t: "tap";
}

export interface SpecialCommand {
  t: "special";
  userId: string;
}

export type ClientMessage = JoinCommand | TapCommand | SpecialCommand;
