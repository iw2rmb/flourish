package grapheme

import (
	"strings"
	"unicode"

	"github.com/rivo/uniseg"
)

// Split returns grapheme clusters for text in visual order.
func Split(text string) []string {
	if text == "" {
		return nil
	}
	g := uniseg.NewGraphemes(text)
	out := make([]string, 0, len([]rune(text)))
	for g.Next() {
		out = append(out, g.Str())
	}
	return out
}

// Count returns the number of grapheme clusters in text.
func Count(text string) int {
	if text == "" {
		return 0
	}
	g := uniseg.NewGraphemes(text)
	n := 0
	for g.Next() {
		n++
	}
	return n
}

// Slice returns the grapheme-safe substring for [start, end).
func Slice(text string, start, end int) string {
	if text == "" {
		return ""
	}
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}

	g := uniseg.NewGraphemes(text)
	idx := 0
	var sb strings.Builder
	for g.Next() {
		if idx >= end {
			break
		}
		if idx >= start {
			sb.WriteString(g.Str())
		}
		idx++
	}
	if start >= idx {
		return ""
	}
	return sb.String()
}

// Join concatenates grapheme clusters into a single string.
func Join(clusters []string) string {
	if len(clusters) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, c := range clusters {
		sb.WriteString(c)
	}
	return sb.String()
}

// IsSpace reports whether all runes in cluster are Unicode whitespace.
func IsSpace(cluster string) bool {
	if cluster == "" {
		return false
	}
	for _, r := range cluster {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

// IsPunct reports whether all runes in cluster are Unicode punctuation.
func IsPunct(cluster string) bool {
	if cluster == "" {
		return false
	}
	for _, r := range cluster {
		if !unicode.IsPunct(r) {
			return false
		}
	}
	return true
}
