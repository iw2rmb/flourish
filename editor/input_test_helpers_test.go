package editor

import (
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
)

func testKeyCode(code rune, mods ...tea.KeyMod) tea.KeyPressMsg {
	return tea.KeyPressMsg{
		Code: code,
		Mod:  testKeyMod(mods...),
	}
}

func testKeyText(text string, mods ...tea.KeyMod) tea.KeyPressMsg {
	code := tea.KeyExtended
	if r, _ := utf8.DecodeRuneInString(text); r != utf8.RuneError {
		code = r
	}
	return tea.KeyPressMsg{
		Text: text,
		Code: code,
		Mod:  testKeyMod(mods...),
	}
}

func testMouseClick(x, y int, button tea.MouseButton, mods ...tea.KeyMod) tea.MouseClickMsg {
	return tea.MouseClickMsg{
		X:      x,
		Y:      y,
		Button: button,
		Mod:    testKeyMod(mods...),
	}
}

func testMouseMotion(x, y int, button tea.MouseButton, mods ...tea.KeyMod) tea.MouseMotionMsg {
	return tea.MouseMotionMsg{
		X:      x,
		Y:      y,
		Button: button,
		Mod:    testKeyMod(mods...),
	}
}

func testMouseRelease(x, y int, button tea.MouseButton, mods ...tea.KeyMod) tea.MouseReleaseMsg {
	return tea.MouseReleaseMsg{
		X:      x,
		Y:      y,
		Button: button,
		Mod:    testKeyMod(mods...),
	}
}

func testMouseWheel(x, y int, button tea.MouseButton, mods ...tea.KeyMod) tea.MouseWheelMsg {
	return tea.MouseWheelMsg{
		X:      x,
		Y:      y,
		Button: button,
		Mod:    testKeyMod(mods...),
	}
}

func testKeyMod(mods ...tea.KeyMod) tea.KeyMod {
	var out tea.KeyMod
	for _, mod := range mods {
		out |= mod
	}
	return out
}
