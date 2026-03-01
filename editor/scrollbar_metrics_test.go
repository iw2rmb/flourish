package editor

import (
	"strings"
	"testing"
)

func TestScrollbarMetrics_AutoVisibilityByOverflow(t *testing.T) {
	t.Run("vertical fits", func(t *testing.T) {
		m := New(Config{Text: "0\n1"})
		m = m.SetSize(5, 2)

		metrics := metricsForTest(&m)
		if metrics.showV {
			t.Fatalf("vertical visibility: got true, want false")
		}
		if metrics.showH {
			t.Fatalf("horizontal visibility: got true, want false")
		}
		if got, want := metrics.contentWidth, 5; got != want {
			t.Fatalf("content width: got %d, want %d", got, want)
		}
		if got, want := metrics.contentHeight, 2; got != want {
			t.Fatalf("content height: got %d, want %d", got, want)
		}
	})

	t.Run("vertical overflow", func(t *testing.T) {
		m := New(Config{Text: "0\n1\n2"})
		m = m.SetSize(5, 2)

		metrics := metricsForTest(&m)
		if !metrics.showV {
			t.Fatalf("vertical visibility: got false, want true")
		}
		if metrics.showH {
			t.Fatalf("horizontal visibility: got true, want false")
		}
		if got, want := metrics.contentWidth, 4; got != want {
			t.Fatalf("content width: got %d, want %d", got, want)
		}
	})
}

func TestScrollbarMetrics_HorizontalWrapNoneOnly(t *testing.T) {
	t.Run("wrap none", func(t *testing.T) {
		m := New(Config{Text: "abcdef"})
		m = m.SetSize(5, 2)

		metrics := metricsForTest(&m)
		if !metrics.showH {
			t.Fatalf("horizontal visibility in WrapNone: got false, want true")
		}
	})

	t.Run("wrapped", func(t *testing.T) {
		m := New(Config{
			Text:     "abcdef",
			WrapMode: WrapGrapheme,
		})
		m = m.SetSize(5, 2)

		metrics := metricsForTest(&m)
		if metrics.showH {
			t.Fatalf("horizontal visibility in wrap mode: got true, want false")
		}
	})
}

func TestScrollbarMetrics_FixedPointAxisCoupling(t *testing.T) {
	text := strings.Join([]string{
		"12345",
		"12345",
		"12345",
	}, "\n")
	m := New(Config{Text: text})
	m = m.SetSize(5, 2)

	metrics := metricsForTest(&m)
	if !metrics.showV || !metrics.showH {
		t.Fatalf("coupled visibility: got showV=%v showH=%v, want both true", metrics.showV, metrics.showH)
	}
	if got, want := metrics.contentWidth, 4; got != want {
		t.Fatalf("content width after coupling: got %d, want %d", got, want)
	}
	if got, want := metrics.contentHeight, 1; got != want {
		t.Fatalf("content height after coupling: got %d, want %d", got, want)
	}
}

func TestScrollbarMetrics_ClampsOffsets(t *testing.T) {
	text := strings.Join([]string{
		"abcdefgh",
		"abcdefgh",
		"abcdefgh",
		"abcdefgh",
		"abcdefgh",
	}, "\n")
	m := New(Config{Text: text})
	m = m.SetSize(5, 3)
	m.viewport.SetYOffset(999)
	m.xOffset = 999

	metrics := metricsForTest(&m)

	maxY := 0
	if metrics.totalRows > metrics.contentHeight {
		maxY = metrics.totalRows - metrics.contentHeight
	}
	if got := metrics.yOffset; got != maxY {
		t.Fatalf("y offset clamp: got %d, want %d", got, maxY)
	}

	maxX := 0
	if metrics.totalCols > metrics.contentWidth {
		maxX = metrics.totalCols - metrics.contentWidth
	}
	if got := metrics.xOffset; got != maxX {
		t.Fatalf("x offset clamp: got %d, want %d", got, maxX)
	}
}

func TestResolveScrollbarThumb_StartMidEnd(t *testing.T) {
	pos, length := resolveScrollbarThumb(10, 20, 100, 0, 1)
	if pos != 0 || length != 2 {
		t.Fatalf("thumb at start: got (pos=%d,len=%d), want (0,2)", pos, length)
	}

	pos, length = resolveScrollbarThumb(10, 20, 100, 40, 1)
	if pos != 4 || length != 2 {
		t.Fatalf("thumb at mid: got (pos=%d,len=%d), want (4,2)", pos, length)
	}

	pos, length = resolveScrollbarThumb(10, 20, 100, 80, 1)
	if pos != 8 || length != 2 {
		t.Fatalf("thumb at end: got (pos=%d,len=%d), want (8,2)", pos, length)
	}
}

func TestResolveScrollbarThumb_MinThumbClamp(t *testing.T) {
	_, length := resolveScrollbarThumb(4, 1, 100, 0, 3)
	if got, want := length, 3; got != want {
		t.Fatalf("min thumb clamp: got %d, want %d", got, want)
	}
}

func metricsForTest(m *Model) scrollbarMetrics {
	lines := m.ensureLines()
	layout := m.ensureLayoutCache(lines)
	return m.resolveScrollbarMetrics(lines, layout)
}
