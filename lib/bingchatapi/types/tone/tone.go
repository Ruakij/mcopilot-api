package tone

// Tone is an enum that represents the possible tone options
type Type string

const (
	Unknown  Type = ""
	Creative Type = "Creative"
	Balanced Type = "Balanced"
	Precise  Type = "Precise"
)

func GetToneByString(toneStr string) Type {
	switch toneStr {
	case string(Creative):
		return Creative
	case string(Balanced):
		return Balanced
	case string(Precise):
		return Precise
	default:
		return Unknown
	}
}
