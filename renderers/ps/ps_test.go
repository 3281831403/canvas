package ps

import (
	"bytes"
	"strings"
	"testing"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/test"
)

func TestPS(t *testing.T) {
	w := &bytes.Buffer{}
	ps := New(w, 100, 80, nil)
	ps.setPaint(canvas.Paint{Color: canvas.Red})
	//test.String(t, string(w.Bytes()), "")
}

// TestPSDashUnits verifies that the PostScript dash array is in millimeters and is independent of the
// stroke width.
func TestPSDashUnits(t *testing.T) {
	render := func(strokeWidth float64) string {
		c := canvas.New(100, 20)
		ctx := canvas.NewContext(c)
		ctx.SetFillColor(canvas.Transparent)
		ctx.SetStrokeColor(canvas.Black)
		ctx.SetStrokeWidth(strokeWidth)
		ctx.SetDashes(0.5, 4.0, 2.0)
		ctx.DrawPath(2, 10, canvas.Line(96, 0))
		buf := &bytes.Buffer{}
		r := New(buf, c.W, c.H, nil)
		c.RenderTo(r)
		test.Error(t, r.Close())
		return buf.String()
	}
	s1 := render(1.0)
	s2 := render(2.0)
	if !strings.Contains(s1, "[4 2].5 setdash") {
		t.Fatalf("missing mm dash array: %s", s1)
	}
	if extractPSDash(s1) != extractPSDash(s2) {
		t.Fatalf("dash array changed with stroke width: %q vs %q", extractPSDash(s1), extractPSDash(s2))
	}
}

func extractPSDash(s string) string {
	i := strings.Index(s, "[")
	if i < 0 {
		return ""
	}
	j := strings.Index(s[i:], " setdash")
	if j < 0 {
		return ""
	}
	return s[i : i+j]
}
