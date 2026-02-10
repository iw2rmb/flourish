package editor

import (
	"sort"
	"strings"
)

type VirtualRole int

const (
	VirtualRoleGhost   VirtualRole = iota // inline suggestion / completion preview
	VirtualRoleOverlay                    // generic inserted text (dim/annotation)
)

// VirtualDeletion hides a half-open rune range [StartCol, EndCol) within a single logical line.
//
// Columns are rune indices in the raw buffer line (before any deletions).
type VirtualDeletion struct {
	StartCol int
	EndCol   int
}

// VirtualInsertion inserts view-only text at a rune column within a single logical line.
//
// Col is a rune index in the raw buffer line (before any deletions).
type VirtualInsertion struct {
	Col  int
	Text string
	Role VirtualRole
}

type VirtualText struct {
	Insertions []VirtualInsertion
	Deletions  []VirtualDeletion
}

type VirtualTextContext struct {
	Row      int
	LineText string // raw buffer line text (unwrapped)

	// Cursor/selection state for conditional transforms.
	CursorCol int // rune index in raw buffer line
	HasCursor bool

	SelectionStartCol int // rune index in raw buffer line
	SelectionEndCol   int // rune index in raw buffer line
	HasSelection      bool

	// Useful for caching.
	DocID      string
	DocVersion uint64
}

type VirtualTextProvider func(ctx VirtualTextContext) VirtualText

func normalizeVirtualText(vt VirtualText, rawLineLen int) VirtualText {
	rawLineLen = maxInt(rawLineLen, 0)

	// Deletions: clamp, drop empty, sort, merge.
	if len(vt.Deletions) > 0 {
		dels := make([]VirtualDeletion, 0, len(vt.Deletions))
		for _, d := range vt.Deletions {
			start := clampInt(d.StartCol, 0, rawLineLen)
			end := clampInt(d.EndCol, 0, rawLineLen)
			if end < start {
				start, end = end, start
			}
			if start == end {
				continue
			}
			dels = append(dels, VirtualDeletion{StartCol: start, EndCol: end})
		}
		sort.Slice(dels, func(i, j int) bool {
			if dels[i].StartCol != dels[j].StartCol {
				return dels[i].StartCol < dels[j].StartCol
			}
			return dels[i].EndCol < dels[j].EndCol
		})
		merged := make([]VirtualDeletion, 0, len(dels))
		for _, d := range dels {
			if len(merged) == 0 {
				merged = append(merged, d)
				continue
			}
			last := &merged[len(merged)-1]
			if d.StartCol <= last.EndCol {
				last.EndCol = maxInt(last.EndCol, d.EndCol)
				continue
			}
			merged = append(merged, d)
		}
		vt.Deletions = merged
	}

	// Insertions: clamp cols, enforce single-line, stable sort by col.
	if len(vt.Insertions) > 0 {
		ins := make([]VirtualInsertion, 0, len(vt.Insertions))
		for _, in := range vt.Insertions {
			col := clampInt(in.Col, 0, rawLineLen)
			text := sanitizeSingleLine(in.Text)
			if text == "" {
				continue
			}
			ins = append(ins, VirtualInsertion{Col: col, Text: text, Role: in.Role})
		}

		// If an insertion anchor falls inside a deleted range, anchor at the deleted range start.
		if len(vt.Deletions) > 0 && len(ins) > 0 {
			for i := range ins {
				for _, d := range vt.Deletions {
					if ins[i].Col >= d.StartCol && ins[i].Col < d.EndCol {
						ins[i].Col = d.StartCol
						break
					}
				}
			}
		}

		sort.SliceStable(ins, func(i, j int) bool {
			return ins[i].Col < ins[j].Col
		})
		vt.Insertions = ins
	}

	return vt
}

func sanitizeSingleLine(s string) string {
	if s == "" {
		return ""
	}
	// v0: insertions must be single-line; drop newline characters.
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	return s
}

func clampInt(v, min, max int) int {
	if max < min {
		return min
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
