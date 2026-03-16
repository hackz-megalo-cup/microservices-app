package battle

func CalcTapDamage(attack int32, atkType, defType string, matchups TypeMatchup) int32 {
	eff := effectiveness(atkType, defType, matchups)
	return int32(float32(attack) * eff)
}

func CalcSpecialDamage(specialDamage int32, atkType, defType string, matchups TypeMatchup) int32 {
	eff := effectiveness(atkType, defType, matchups)
	return int32(float32(specialDamage) * eff)
}

func RequiredTapsForSpecial(speed int32, baseTaps, coefficient int32) int32 {
	r := baseTaps - (speed / coefficient)
	if r < 1 {
		return 1
	}
	return r
}

func effectiveness(atkType, defType string, matchups TypeMatchup) float32 {
	if m, ok := matchups[atkType]; ok {
		if eff, ok := m[defType]; ok {
			return eff
		}
	}
	return 1.0
}
