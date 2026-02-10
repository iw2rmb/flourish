package editor

func findWordWrapBreak(units []wrapUnit, start, overflow int) (int, bool) {
	if start < 0 {
		start = 0
	}
	if overflow > len(units) {
		overflow = len(units)
	}
	if start >= overflow {
		return 0, false
	}

	lastBreak := -1
	i := start
	for i < overflow {
		if !units[i].isWhitespace {
			i++
			continue
		}
		j := i + 1
		for j < overflow && units[j].isWhitespace {
			j++
		}
		lastBreak = j
		i = j
	}

	if lastBreak <= start {
		return 0, false
	}
	return lastBreak, true
}

// adjustBreakForLeadingPunctuation mirrors Spok's heuristic:
// when no word boundary exists, avoid wrapping so the next line starts with a
// punctuation-only fragment when possible.
func adjustBreakForLeadingPunctuation(units []wrapUnit, start, end int) int {
	if start < 0 {
		start = 0
	}
	if end > len(units) {
		end = len(units)
	}
	if end <= start {
		return minInt(start+1, len(units))
	}

	for end < len(units) && end-start > 1 {
		if !units[end].isPunct || units[end-1].isWhitespace {
			break
		}
		end--
	}
	if end <= start {
		return minInt(start+1, len(units))
	}
	return end
}
