package constants

const (
	FlagNoShortName rune = -1 // Runes are actually int32. Negative values indicate unknown rune

	NoDrawLinePrefix = "  "
	// https://en.wikipedia.org/wiki/List_of_Unicode_characters#Box_Drawing
	UsageBoxUpDown        = "\u2503" // ┃
	UsageBoxLeftRightDown = "\u2533" // ┳
	UsageBoxLeftRight     = "\u2501" // ━
	UsageBoxRightDown     = "\u250f" // ┏
	UsageBoxLeftUp        = "\u251b" // ┛
	UsageBoxLeftDown      = "\u2513" // ┓
)

var (
	BoolStringValues = []string{
		"1",
		"t",
		"T",
		"true",
		"TRUE",
		"True",
		"0",
		"f",
		"F",
		"false",
		"FALSE",
		"False",
	}
)
