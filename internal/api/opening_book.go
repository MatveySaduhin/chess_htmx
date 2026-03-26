package api

import "strings"

type OpeningBookMove struct {
	SAN   string `json:"san"`
	Note  string `json:"note,omitempty"`
	White int    `json:"white,omitempty"`
	Draws int    `json:"draws,omitempty"`
	Black int    `json:"black,omitempty"`
}

type OpeningBookEntry struct {
	ECO   string            `json:"eco"`
	Name  string            `json:"name"`
	Moves []OpeningBookMove `json:"moves"`
}

type openingLine struct {
	Sequence []string
	Entry    OpeningBookEntry
}

var localOpeningBook = []openingLine{
	{
		Sequence: []string{"e4"},
		Entry: OpeningBookEntry{
			ECO:  "B00",
			Name: "King's Pawn Game",
			Moves: []OpeningBookMove{
				{SAN: "e5"},
				{SAN: "c5"},
				{SAN: "e6"},
				{SAN: "c6"},
				{SAN: "d6"},
			},
		},
	},
	{
		Sequence: []string{"d4"},
		Entry: OpeningBookEntry{
			ECO:  "A40",
			Name: "Queen's Pawn Game",
			Moves: []OpeningBookMove{
				{SAN: "d5"},
				{SAN: "Nf6"},
				{SAN: "e6"},
				{SAN: "g6"},
			},
		},
	},
	{
		Sequence: []string{"c4"},
		Entry: OpeningBookEntry{
			ECO:  "A10",
			Name: "English Opening",
			Moves: []OpeningBookMove{
				{SAN: "e5"},
				{SAN: "Nf6"},
				{SAN: "c5"},
				{SAN: "e6"},
			},
		},
	},
	{
		Sequence: []string{"Nf3"},
		Entry: OpeningBookEntry{
			ECO:  "A04",
			Name: "Réti Opening",
			Moves: []OpeningBookMove{
				{SAN: "d5"},
				{SAN: "Nf6"},
				{SAN: "c5"},
			},
		},
	},
	{
		Sequence: []string{"e4", "e5"},
		Entry: OpeningBookEntry{
			ECO:  "C20",
			Name: "Open Game",
			Moves: []OpeningBookMove{
				{SAN: "Nf3"},
				{SAN: "Bc4"},
				{SAN: "Nc3"},
				{SAN: "f4"},
			},
		},
	},
	{
		Sequence: []string{"e4", "c5"},
		Entry: OpeningBookEntry{
			ECO:  "B20",
			Name: "Sicilian Defense",
			Moves: []OpeningBookMove{
				{SAN: "Nf3"},
				{SAN: "Nc3"},
				{SAN: "c3"},
				{SAN: "d4"},
			},
		},
	},
	{
		Sequence: []string{"e4", "e6"},
		Entry: OpeningBookEntry{
			ECO:  "C00",
			Name: "French Defense",
			Moves: []OpeningBookMove{
				{SAN: "d4"},
				{SAN: "Nc3"},
				{SAN: "Nf3"},
			},
		},
	},
	{
		Sequence: []string{"e4", "c6"},
		Entry: OpeningBookEntry{
			ECO:  "B10",
			Name: "Caro-Kann Defense",
			Moves: []OpeningBookMove{
				{SAN: "d4"},
				{SAN: "Nc3"},
				{SAN: "Nf3"},
			},
		},
	},
	{
		Sequence: []string{"d4", "d5"},
		Entry: OpeningBookEntry{
			ECO:  "D00",
			Name: "Queen's Pawn Game",
			Moves: []OpeningBookMove{
				{SAN: "c4"},
				{SAN: "Nf3"},
				{SAN: "e3"},
			},
		},
	},
	{
		Sequence: []string{"d4", "Nf6"},
		Entry: OpeningBookEntry{
			ECO:  "A46",
			Name: "Indian Defense",
			Moves: []OpeningBookMove{
				{SAN: "c4"},
				{SAN: "Nf3"},
				{SAN: "g3"},
			},
		},
	},
	{
		Sequence: []string{"e4", "e5", "Nf3"},
		Entry: OpeningBookEntry{
			ECO:  "C40",
			Name: "King's Knight Opening",
			Moves: []OpeningBookMove{
				{SAN: "Nc6"},
				{SAN: "d6"},
				{SAN: "Nf6"},
			},
		},
	},
	{
		Sequence: []string{"e4", "e5", "Nf3", "Nc6", "Bc4"},
		Entry: OpeningBookEntry{
			ECO:  "C50",
			Name: "Italian Game",
			Moves: []OpeningBookMove{
				{SAN: "Bc5"},
				{SAN: "Nf6"},
				{SAN: "d6"},
			},
		},
	},
	{
		Sequence: []string{"e4", "e5", "Nf3", "Nc6", "Bb5"},
		Entry: OpeningBookEntry{
			ECO:  "C60",
			Name: "Ruy Lopez",
			Moves: []OpeningBookMove{
				{SAN: "a6"},
				{SAN: "Nf6"},
				{SAN: "d6"},
			},
		},
	},
	{
		Sequence: []string{"e4", "c5", "Nf3"},
		Entry: OpeningBookEntry{
			ECO:  "B27",
			Name: "Sicilian Defense: Open",
			Moves: []OpeningBookMove{
				{SAN: "d6"},
				{SAN: "Nc6"},
				{SAN: "e6"},
			},
		},
	},
	{
		Sequence: []string{"e4", "e6", "d4"},
		Entry: OpeningBookEntry{
			ECO:  "C00",
			Name: "French Defense",
			Moves: []OpeningBookMove{
				{SAN: "d5"},
			},
		},
	},
	{
		Sequence: []string{"e4", "e6", "d4", "d5", "Nc3"},
		Entry: OpeningBookEntry{
			ECO:  "C10",
			Name: "French Defense: Paulsen Variation",
			Moves: []OpeningBookMove{
				{SAN: "Bb4"},
				{SAN: "Nf6"},
				{SAN: "dxe4"},
			},
		},
	},
	{
		Sequence: []string{"e4", "c6", "d4"},
		Entry: OpeningBookEntry{
			ECO:  "B10",
			Name: "Caro-Kann Defense",
			Moves: []OpeningBookMove{
				{SAN: "d5"},
			},
		},
	},
	{
		Sequence: []string{"e4", "c6", "d4", "d5", "Nc3"},
		Entry: OpeningBookEntry{
			ECO:  "B12",
			Name: "Caro-Kann Defense: Classical Variation",
			Moves: []OpeningBookMove{
				{SAN: "dxe4"},
				{SAN: "g6"},
				{SAN: "Nf6"},
			},
		},
	},
	{
		Sequence: []string{"d4", "d5", "c4"},
		Entry: OpeningBookEntry{
			ECO:  "D06",
			Name: "Queen's Gambit",
			Moves: []OpeningBookMove{
				{SAN: "e6"},
				{SAN: "dxc4"},
				{SAN: "c6"},
			},
		},
	},
	{
		Sequence: []string{"d4", "d5", "c4", "e6"},
		Entry: OpeningBookEntry{
			ECO:  "D30",
			Name: "Queen's Gambit Declined",
			Moves: []OpeningBookMove{
				{SAN: "Nc3"},
				{SAN: "Nf3"},
				{SAN: "g3"},
			},
		},
	},
	{
		Sequence: []string{"d4", "Nf6", "c4"},
		Entry: OpeningBookEntry{
			ECO:  "A56",
			Name: "Indian Defense",
			Moves: []OpeningBookMove{
				{SAN: "g6"},
				{SAN: "e6"},
				{SAN: "d6"},
			},
		},
	},
	{
		Sequence: []string{"d4", "Nf6", "c4", "g6"},
		Entry: OpeningBookEntry{
			ECO:  "E60",
			Name: "King's Indian Defense",
			Moves: []OpeningBookMove{
				{SAN: "Nc3"},
				{SAN: "Nf3"},
				{SAN: "g3"},
			},
		},
	},
	{
		Sequence: []string{"d4", "Nf6", "c4", "e6"},
		Entry: OpeningBookEntry{
			ECO:  "E10",
			Name: "Queen's Indian / Nimzo-Indian Complex",
			Moves: []OpeningBookMove{
				{SAN: "Nc3"},
				{SAN: "Nf3"},
				{SAN: "g3"},
			},
		},
	},
}

func normalizeSANList(moves []string) []string {
	out := make([]string, 0, len(moves))
	for _, m := range moves {
		trimmed := strings.TrimSpace(m)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func isPrefix(full []string, prefix []string) bool {
	if len(prefix) > len(full) {
		return false
	}
	for i := range prefix {
		if full[i] != prefix[i] {
			return false
		}
	}
	return true
}

func lookupOpeningBySAN(history []string) OpeningBookEntry {
	history = normalizeSANList(history)

	best := OpeningBookEntry{
		ECO:   "",
		Name:  "Unknown opening",
		Moves: []OpeningBookMove{},
	}
	bestLen := 0

	for _, line := range localOpeningBook {
		if isPrefix(history, line.Sequence) && len(line.Sequence) > bestLen {
			best = line.Entry
			bestLen = len(line.Sequence)
		}
	}

	return best
}
