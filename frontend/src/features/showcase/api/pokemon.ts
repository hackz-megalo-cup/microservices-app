import type { Pokemon as RpcPokemon } from "../../../gen/masterdata/v1/masterdata_pb";
import type { Pokemon, PokemonStat } from "../types";

const IMAGE_PLACEHOLDER = "/images/collection-placeholder.png";

export function generatePokemonNumber(index: number): string {
  return `#${String(index + 1).padStart(3, "0")}`;
}

export function getPokemonImageUrl(pokemon: Pick<Pokemon, "name">): string {
  const normalizedName = pokemon.name.trim().toLowerCase().replace(/\s+/g, "-");
  if (!normalizedName || normalizedName === "???") {
    return IMAGE_PLACEHOLDER;
  }

  return `/images/pokemon-${normalizedName}.png`;
}

export function adaptPokemonToUi(serverPokemon: RpcPokemon, index: number): Pokemon {
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
    captured: true,
  };
}
