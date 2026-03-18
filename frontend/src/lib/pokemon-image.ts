const IMAGE_PLACEHOLDER = "/images/collection-python.png";

export function getPokemonImageUrl(pokemon: { name: string }): string {
  const normalizedName = pokemon.name.trim().toLowerCase().replace(/\s+/g, "-");
  if (!normalizedName || normalizedName === "???") {
    return IMAGE_PLACEHOLDER;
  }

  return `/images/pokemon-${normalizedName}.png`;
}
