package battle

import "github.com/google/uuid"

type Participant struct {
	UserID             uuid.UUID
	PokemonAttack      int32
	PokemonSpeed       int32
	PokemonType        string
	SpecialMoveName    string
	SpecialMoveDamage  int32
	TapCount           int32
	RequiredForSpecial int32
}

type TypeMatchup map[string]map[string]float32 // attacking_type -> defending_type -> effectiveness
