export interface Raid {
  id: string;
  name: string;
  type: string;
  players: string;
  timer: string;
  image: string;
}

export interface Trainer {
  name: string;
  pokemon: string;
  online: boolean;
}

export interface Participant {
  id: string;
  userId: string;
  name: string;
  pokemon: string;
  online: boolean;
}

export interface LobbyState {
  lobbyId: string;
  participants: Participant[];
  bossName?: string;
  bossImage?: string;
  bossType?: string;
  maxParticipants: number;
}

export interface PokemonStat {
  label: string;
  value: number;
}

export interface Move {
  name: string;
  type: string;
  power: number;
}

export interface Pokemon {
  id: string;
  name: string;
  number: string;
  image: string;
  types: string[];
  stats: PokemonStat[];
  about: string;
  moves: Move[];
  captured: boolean;
}
