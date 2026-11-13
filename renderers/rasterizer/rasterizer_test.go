package rasterizer

import (
	"image"
	"image/color"
	"testing"

	"github.com/tdewolff/canvas"
)

// TestColorSpaceConsistency draws the same 50% gray as a solid fill, a
// gradient, a gradient inside a hatch pattern, and an image (drawn twice) in
// fixed rectangles, and samples fixed points: all must render to the same
// value, no matter how often the canvas is rendered, and the gradient and
// image passed in must not be modified.
func TestColorSpaceConsistency(t *testing.T) {
	gray := color.RGBA{128, 128, 128, 255}

	grad := canvas.NewLinearGradient(canvas.Point{0, 0}, canvas.Point{10, 0})
	grad.Add(0.0, gray)
	grad.Add(1.0, gray)

	hatch := canvas.NewLineHatch(grad, 0.0, 2.0, 2.0)

	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.SetRGBA(x, y, gray)
		}
	}

	newCanvas := func() *canvas.Canvas {
		c := canvas.New(50.0, 10.0)
		ctx := canvas.NewContext(c)
		ctx.SetFillColor(gray)
		ctx.DrawPath(0.0, 0.0, canvas.Rectangle(10.0, 10.0))
		ctx.SetFillGradient(grad)
		ctx.DrawPath(10.0, 0.0, canvas.Rectangle(10.0, 10.0))
		ctx.SetFillPattern(hatch)
		ctx.DrawPath(20.0, 0.0, canvas.Rectangle(10.0, 10.0))
		ctx.DrawImage(30.0, 0.0, img, canvas.DPMM(1.0))
		ctx.DrawImage(40.0, 0.0, img, canvas.DPMM(1.0)) // same image again
		return c
	}

	samplePoints := []image.Point{{5, 5}, {15, 5}, {25, 5}, {35, 5}, {45, 5}}
	render := func() [5]color.RGBA {
		raster := Draw(newCanvas(), canvas.DPMM(1.0), canvas.SRGBColorSpace{})
		var samples [5]color.RGBA
		for i, p := range samplePoints {
			samples[i] = raster.RGBAAt(p.X, p.Y)
		}
		return samples
	}

	first := render()
	for i, got := range first {
		if got != first[0] {
			t.Errorf("sample %d: got %v, want %v (same as solid fill)", i, got, first[0])
		}
		if d := int(got.R) - int(gray.R); d < -1 || 1 < d {
			t.Errorf("sample %d: got %v, want approximately %v", i, got, gray)
		}
	}

	// rendering again must give identical results (no double conversion)
	for n := 0; n < 9; n++ {
		if got := render(); got != first {
			t.Fatalf("render %d: got %v, want %v", n+2, got, first)
		}
	}

	// the gradient and image passed in must be unmodified
	if got := grad.Grad.At(0.0); got != gray {
		t.Errorf("gradient was modified: got stop color %v, want %v", got, gray)
	}
	if got := grad.Grad.At(1.0); got != gray {
		t.Errorf("gradient was modified: got stop color %v, want %v", got, gray)
	}
	if got := img.RGBAAt(5, 5); got != gray {
		t.Errorf("image was modified: got pixel %v, want %v", got, gray)
	}
}

// TestColorSpaceDefaultUnchanged ensures the default (linear) color space
// passes colors through untouched.
func TestColorSpaceDefaultUnchanged(t *testing.T) {
	gray := color.RGBA{128, 128, 128, 255}
	grad := canvas.NewLinearGradient(canvas.Point{0, 0}, canvas.Point{10, 0})
	grad.Add(0.0, gray)
	grad.Add(1.0, gray)

	c := canvas.New(20.0, 10.0)
	ctx := canvas.NewContext(c)
	ctx.SetFillColor(gray)
	ctx.DrawPath(0.0, 0.0, canvas.Rectangle(10.0, 10.0))
	ctx.SetFillGradient(grad)
	ctx.DrawPath(10.0, 0.0, canvas.Rectangle(10.0, 10.0))

	raster := Draw(c, canvas.DPMM(1.0), canvas.DefaultColorSpace)
	for _, p := range []image.Point{{5, 5}, {15, 5}} {
		if got := raster.RGBAAt(p.X, p.Y); got != gray {
			t.Errorf("got %v, want %v", got, gray)
		}
	}
}
