package network

type ID uint64

const (
	MainNet ID = iota + 1
	TestNet
	Aquarius
	Pisces
	Aries
	Taurus
	Gemini
	Cancer
	Leo
	Virgo
	Libra
	Scorpio
	Sagittarius
	Capricorn
)

var network = [...]string{
	MainNet:     "MainNet",
	TestNet:     "TestNet",
	Aquarius:    "Aquarius",
	Pisces:      "Pisces",
	Aries:       "Aries",
	Taurus:      "Taurus",
	Gemini:      "Gemini",
	Cancer:      "Cancer",
	Leo:         "Leo",
	Virgo:       "Virgo",
	Libra:       "Libra",
	Scorpio:     "Scorpio",
	Sagittarius: "Sagittarius",
	Capricorn:   "Capricorn",
}

func (i ID) String() string {
	if i >= MainNet && i <= Capricorn {
		return network[i]
	}

	return "Unknown"
}
