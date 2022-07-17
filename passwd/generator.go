package passwd

import "github.com/m1/go-generate-password/generator"

var config = generator.Config{
	Length:                  16,
	IncludeSymbols:          false,
	IncludeNumbers:          true,
	IncludeLowercaseLetters: true,
	IncludeUppercaseLetters: true,
	// ExcludeSimilarCharacters:   true,
	// ExcludeAmbiguousCharacters: true,
}

var g *generator.Generator

func init() {
	g, _ = generator.New(&config)
}

func Generator() string {
	s, _ := g.Generate()
	if s == nil {
		return "1234567890asdfghjkl@#$^TGFGFDGFGMGNDcghjlkloi|)_("
	}
	return *s
}
