const IMAGE_PLACEHOLDER = "/images/collection-python.png";

const POKEMON_IMAGES: Record<string, string> = {
  go: "/images/collection-go.png",
  python: "/images/collection-python.png",
  typescript: "/images/collection-typescript.png",
  java: "/images/collection-java.jpg",
  moonbit: "/images/collection-moonbit.jpg",
  php: "/images/collection-php.jpg",
  swift: "/images/collection-swift.jpg",
  rust: "/images/raid-rust.png",
};

export function getPokemonImageUrl(pokemon: { name: string }): string {
  const normalizedName = pokemon.name.trim().toLowerCase().replace(/\s+/g, "-");
  if (!normalizedName || normalizedName === "???") {
    return IMAGE_PLACEHOLDER;
  }

  return POKEMON_IMAGES[normalizedName] ?? IMAGE_PLACEHOLDER;
}
