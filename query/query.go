package query

import (
	"fmt"
	"strings"

	e "github.com/julianlk522/fitm/error"
)

type Query struct {
	Text  string
	Args  []interface{}
	Error error
}

func GetPeriodClause(period string) (clause string, err error) {
	var days int
	switch period {
	case "day":
		days = 1
	case "week":
		days = 7
	case "month":
		days = 30
	case "year":
		days = 365
	default:
		return "", e.ErrInvalidPeriod
	}

	return fmt.Sprintf("submit_date >= date('now', '-%d days')", days), nil
}

func EscapeCatsReservedChars(cats []string) {
	for i := 0; i < len(cats); i++ {
		cats[i] = SurroundReservedCharsWithDoubleQuotes(cats[i])
	}
}

func SurroundReservedCharsWithDoubleQuotes(cat string) string {
	return reserved_chars_double_quotes_replacer.Replace(cat)
}

var reserved_chars_double_quotes_replacer = strings.NewReplacer(
	// ! seems to work already with no modifications
	".", `"."`,
	"-", `"-"`,
	// + seems to work
	"'", `"'"`,
	// double quotes seems to work
	"#", `"#"`,
	"$", `"$"`,
	"%", `"%"`,
	"&", `"&"`,
	"\\", `"\"`,
	"/", `"/"`,
	"(", `"("`,
	")", `")"`,
	"[", `"["`,
	"]", `"]"`,
	"{", `"{"`,
	"}", `"}"`,
	"|", `"|"`,
	":", `":"`,
	";", `";"`,
	"=", `"="`,
	"?", `"?"`,
	"@", `"@"`,
)