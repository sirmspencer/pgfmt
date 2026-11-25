package format

import (
	"strings"

	pg "github.com/pganalyze/pg_query_go/v6"
)

// a placeholder fragment is read as a comment, so the printer places it
// beside the nearest printed line, which can move it past the words that
// followed it, as in `on conflict :target do nothing`. inquery pastes raw
// text there, so a moved fragment changes the query. each fragment must stay
// next to a token it was written next to.

// fragmentSpot is a placeholder fragment and the tokens on either side of it,
// lowercased, skipping comments. "" stands for the start or end of the statement.
type fragmentSpot struct {
	text, before, after string
	start, end          int // byte range of the fragment in the text it was found in
	afterStart          int // where the token after it starts, or the end of the text
}

// fragmentSpots finds the placeholder fragments in text, whose tokens were
// scanned from the same text with placeholders substituted
func fragmentSpots(text string, tokens []*pg.ScanToken) []fragmentSpot {
	tokenText := func(token *pg.ScanToken) string {
		word := strings.ToLower(text[token.Start:token.End])
		// a migration adds a semicolon the source did not have
		if word == ";" {
			return ""
		}
		return word
	}
	var spots []fragmentSpot
	for index, token := range tokens {
		if !isComment(token) || !strings.HasPrefix(text[token.Start:token.End], ":") {
			continue
		}
		spot := fragmentSpot{
			text:       strings.TrimRight(text[token.Start:token.End], " \t\r\n"),
			start:      int(token.Start),
			end:        int(token.End),
			afterStart: len(text),
		}
		for before := index - 1; before >= 0; before-- {
			if !isComment(tokens[before]) {
				spot.before = tokenText(tokens[before])
				break
			}
		}
		for after := index + 1; after < len(tokens); after++ {
			if !isComment(tokens[after]) {
				spot.after = tokenText(tokens[after])
				spot.afterStart = int(tokens[after].Start)
				break
			}
		}
		spots = append(spots, spot)
	}
	return spots
}

// placeFragments checks that every fragment in printed has a neighbor it had
// in the source, between source offsets from and to. a fragment that lost
// both neighbors moves to the one place where its two source neighbors are
// next to each other. it returns false when a fragment has no such place.
func (p *printer) placeFragments(printed string, from, to int32) (string, bool) {
	var wanted []fragmentSpot
	for _, spot := range fragmentSpots(p.src.orig, p.src.tokens) {
		if spot.start >= int(from) && spot.end <= int(to) {
			wanted = append(wanted, spot)
		}
	}
	if len(wanted) == 0 {
		return printed, true
	}
	for index := range wanted {
		// rescan after each move, since moving one fragment shifts the others
		spots, tokens, ok := p.printedFragments(printed)
		if !ok || len(spots) != len(wanted) {
			return printed, false
		}
		want, got := wanted[index], spots[index]
		if got.text != want.text {
			return printed, false
		}
		if got.before == want.before || got.after == want.after {
			continue
		}
		// take the fragment out, with the spaces in front of it, and its line
		// when it was alone there
		cut, cutEnd := got.start, got.end
		for cut > 0 && printed[cut-1] == ' ' {
			cut--
		}
		if (cut == 0 || printed[cut-1] == '\n') && cutEnd < len(printed) && printed[cutEnd] == '\n' {
			cutEnd++
		}
		rest := printed[:cut] + printed[cutEnd:]
		shift := cutEnd - cut
		// the place is between two consecutive tokens other than comments,
		// after any fragments already moved there, which keeps their order
		target := -1
		previous := -1
		for position, token := range tokens {
			if isComment(token) {
				continue
			}
			if previous >= 0 &&
				strings.ToLower(printed[tokens[previous].Start:tokens[previous].End]) == want.before &&
				strings.ToLower(printed[token.Start:token.End]) == want.after {
				if target >= 0 {
					return printed, false
				}
				target = int(tokens[position-1].End)
				if position-1 != previous && int(tokens[position-1].Start) == got.start {
					// the fragment being moved is already the one in front
					target = int(tokens[previous].End)
				}
			}
			previous = position
		}
		if target < 0 {
			return printed, false
		}
		if target > cut {
			target -= shift
		}
		// a fragment written on its own line goes on its own line, indented
		// like the line it follows
		separator := " "
		if p.src.ownLine(int32(want.start)) {
			lineStart := strings.LastIndexByte(rest[:target], '\n') + 1
			line := rest[lineStart:target]
			separator = "\n" + line[:len(line)-len(strings.TrimLeft(line, " "))]
		}
		printed = rest[:target] + separator + want.text + rest[target:]
	}
	return printed, true
}

// printedFragments scans printed text the same way as the source, so its
// fragments read as comments again
func (p *printer) printedFragments(printed string) ([]fragmentSpot, []*pg.ScanToken, bool) {
	// multi-line string literals hold a one-byte marker in place of each newline
	text, err := p.params.substitute(strings.ReplaceAll(printed, literalNewline, "\n"))
	if err != nil {
		return nil, nil, false
	}
	scan, err := pg.Scan(text)
	if err != nil {
		return nil, nil, false
	}
	// the substituted text has the same length, so offsets read the printed text
	return fragmentSpots(printed, scan.Tokens), scan.Tokens, true
}
