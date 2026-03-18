import type { Pokemon as RpcPokemon } from "../../../gen/masterdata/v1/masterdata_pb";
import { getPokemonImageUrl } from "../../../lib/pokemon-image";
import type { Pokemon, PokemonStat } from "../types";

export { getPokemonImageUrl };

export function generatePokemonNumber(index: number): string {
  return `#${String(index + 1).padStart(3, "0")}`;
}

export function adaptPokemonToUi(
  serverPokemon: RpcPokemon,
  index: number,
  captured = true,
): Pokemon {
  const stats: PokemonStat[] = [
    { label: "HP", value: serverPokemon.hp },
    { label: "ATK", value: serverPokemon.attack },
    { label: "SPD", value: serverPokemon.speed },
  ];

  const moves = serverPokemon.specialMoveName
    ? [
        {
          name: serverPokemon.specialMoveName,
          type: "Normal",
          power: serverPokemon.specialMoveDamage,
        },
      ]
    : [];

  return {
    id: serverPokemon.id,
    name: serverPokemon.name,
    number: generatePokemonNumber(index),
    image: getPokemonImageUrl({ name: serverPokemon.name }),
    types: serverPokemon.type ? [serverPokemon.type] : [],
    stats,
    about: "Masterdata から取得したポケモン情報です。",
    moves,
    captured,
  };
}
