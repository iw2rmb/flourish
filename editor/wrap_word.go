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
