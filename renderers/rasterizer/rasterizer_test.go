package rasterizer

import (
	"testing"

	"github.com/tdewolff/canvas"
)

// dashRuns renders a horizontal dashed line at the given stroke width and DPMM and returns the on-pixel
// run lengths along the line's center.
func dashRuns(t *testing.T, strokeWidth, dpmm float64) []int {
	t.Helper()
	c := canvas.New(100, 20)
	ctx := canvas.NewContext(c)
	ctx.SetFillColor(canvas.Transparent)
	ctx.SetStrokeColor(canvas.Black)
	ctx.SetStrokeWidth(strokeWidth)
	ctx.SetDashes(0, 10, 10)
	ctx.DrawPath(0, 10, canvas.Line(100, 0))

	img := Draw(c, canvas.DPMM(dpmm), canvas.DefaultColorSpace)
	y := img.Bounds().Dy() / 2
	runs := []int{}
	inRun, run := false, 0
	for x := 0; x < img.Bounds().Dx(); x++ {
		_, _, _, a := img.At(x, y).RGBA()
		if a > 0 {
			if !inRun {
				inRun, run = true, 0
			}
			run++
		} else if inRun {
			inRun = false
			runs = append(runs, run)
		}
	}
	if inRun {
		runs = append(runs, run)
	}
	return runs
}

// TestRasterizerDashUnits verifies that dash geometry is expressed in millimeters: the pixel runs are
// independent of the stroke width and scale with the rasterization resolution.
func TestRasterizerDashUnits(t *testing.T) {
	// identical dash runs at 1mm and 2mm stroke width (10mm on / 10mm off at 10 DPMM => ~100px runs)
	runs1 := dashRuns(t, 1.0, 10.0)
	runs2 := dashRuns(t, 2.0, 10.0)
	if len(runs1) == 0 || len(runs1) != len(runs2) {
		t.Fatalf("unexpected run counts: %v vs %v", runs1, runs2)
	}
	for i := range runs1 {
		if runs1[i] != runs2[i] {
			t.Fatalf("dash run %d changed with stroke width: %v vs %v", i, runs1, runs2)
		}
		if runs1[i] < 95 || 105 < runs1[i] {
			t.Fatalf("dash run %d not ~100px for a 10mm dash at 10DPMM: %v", i, runs1)
		}
	}

	// doubling the resolution must double the pixel runs (same physical dash length)
	runsHi := dashRuns(t, 1.0, 20.0)
	if len(runsHi) == 0 || runsHi[0] < 195 || 205 < runsHi[0] {
		t.Fatalf("dash runs did not scale with resolution: %v", runsHi)
	}
}
