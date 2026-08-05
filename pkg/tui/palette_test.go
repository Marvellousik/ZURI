package tui_test

import (
	"bytes"
	"strings"
	"testing"

	"zuri-daemon/pkg/tui"
)

func TestCommandPalette_FilterAndRender(t *testing.T) {
	cp := tui.NewCommandPalette()

	// 1. Filter test
	matches := cp.Filter("repo")
	if len(matches) < 2 {
		t.Fatalf("expected at least 2 repo commands, got %d", len(matches))
	}

	// 2. Render Overlay Test
	var buf bytes.Buffer
	cp.RenderOverlay(&buf, "model")

	output := buf.String()
	if !strings.Contains(output, "ZURI COMMAND PALETTE") {
		t.Errorf("expected header in palette overlay, got: %s", output)
	}
	if !strings.Contains(output, "model switch") {
		t.Errorf("expected 'model switch' command in output, got: %s", output)
	}
}
