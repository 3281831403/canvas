package canvas

import (
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/tdewolff/test"
)

func TestParseSVGRGBADecimalAlpha(t *testing.T) {
	var tests = []struct {
		name     string
		svg      string
		expected color.RGBA
	}{
		{
			"decimal alpha 0.5",
			`<svg width="100" height="100"><rect x="0" y="0" width="10" height="10" fill="rgba(255,0,0,0.5)"/></svg>`,
			color.RGBA{127, 0, 0, 127}, // premultiplied alpha
		},
		{
			"decimal alpha 1.0",
			`<svg width="100" height="100"><rect x="0" y="0" width="10" height="10" fill="rgba(255,0,0,1.0)"/></svg>`,
			color.RGBA{255, 0, 0, 255},
		},
		{
			"integer alpha 1 (fully opaque)",
			`<svg width="100" height="100"><rect x="0" y="0" width="10" height="10" fill="rgba(255,0,0,1)"/></svg>`,
			color.RGBA{255, 0, 0, 255},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := ParseSVG(strings.NewReader(tt.svg))
			test.Error(t, err)
			if len(c.layers) == 0 || len(c.layers[0]) == 0 {
				t.Fatal("no layers rendered")
			}
			layer := c.layers[0][0]
			test.T(t, layer.style.Fill.Color, tt.expected)
		})
	}
}

func TestParseSVGMarkerWithoutViewBox(t *testing.T) {
	svg := `<svg width="100" height="100">
		<defs>
			<marker id="arrow" markerWidth="6" markerHeight="6" refX="3" refY="3" orient="auto" markerUnits="strokeWidth">
				<path d="M0,0 L0,6 L6,3 z" fill="rgb(255,0,0)"/>
			</marker>
		</defs>
		<path d="M10,50 L90,50" stroke="black" stroke-width="2" marker-end="url(#arrow)"/>
	</svg>`

	c, err := ParseSVG(strings.NewReader(svg))
	test.Error(t, err)

	if c == nil {
		t.Fatal("canvas is nil")
	}

	totalLayers := 0
	for _, layers := range c.layers {
		totalLayers += len(layers)
	}
	if totalLayers < 2 {
		t.Fatalf("expected at least 2 layers (path + marker), got %d", totalLayers)
	}
}

func TestParseSVGMarkerWithViewBox(t *testing.T) {
	svg := `<svg width="100" height="100">
		<defs>
			<marker id="arrow" markerWidth="6" markerHeight="6" refX="3" refY="3" orient="auto" markerUnits="strokeWidth" viewBox="0 0 6 6">
				<path d="M0,0 L0,6 L6,3 z" fill="rgb(255,0,0)"/>
			</marker>
		</defs>
		<path d="M10,50 L90,50" stroke="black" stroke-width="2" marker-end="url(#arrow)"/>
	</svg>`

	c, err := ParseSVG(strings.NewReader(svg))
	test.Error(t, err)

	if c == nil {
		t.Fatal("canvas is nil")
	}

	totalLayers := 0
	for _, layers := range c.layers {
		totalLayers += len(layers)
	}
	if totalLayers < 2 {
		t.Fatalf("expected at least 2 layers (path + marker), got %d", totalLayers)
	}
}

func TestParseSVGPatternLineHatch(t *testing.T) {
	svg := `<svg width="100" height="100">
		<defs>
			<pattern id="hatch" patternUnits="userSpaceOnUse" width="6" height="6" patternTransform="rotate(45)">
				<line x1="0" y1="0" x2="0" y2="6" stroke="rgb(255,0,0)" stroke-width="1"/>
			</pattern>
		</defs>
		<circle cx="50" cy="50" r="40" fill="url(#hatch)"/>
	</svg>`

	c, err := ParseSVG(strings.NewReader(svg))
	test.Error(t, err)

	if len(c.layers) == 0 || len(c.layers[0]) == 0 {
		t.Fatal("no layers rendered")
	}

	layer := c.layers[0][0]
	if !layer.style.Fill.IsPattern() {
		t.Fatal("expected fill to be a pattern")
	}

	hatch, ok := layer.style.Fill.Pattern.(*HatchPattern)
	if !ok {
		t.Fatal("expected HatchPattern")
	}

	test.T(t, hatch.Fill.Color, color.RGBA{255, 0, 0, 255})
}

func TestParseSVGPatternCurrentColor(t *testing.T) {
	svg := `<svg width="100" height="100">
		<defs>
			<pattern id="hatch" patternUnits="userSpaceOnUse" width="6" height="6" patternTransform="rotate(45)">
				<line x1="0" y1="0" x2="0" y2="6" stroke="currentColor" stroke-width="1" stroke-opacity="0.7"/>
			</pattern>
		</defs>
		<circle cx="50" cy="50" r="40" fill="url(#hatch)" style="color:rgb(0,255,0)"/>
	</svg>`

	c, err := ParseSVG(strings.NewReader(svg))
	test.Error(t, err)

	if len(c.layers) == 0 || len(c.layers[0]) == 0 {
		t.Fatal("no layers rendered")
	}

	layer := c.layers[0][0]
	if !layer.style.Fill.IsPattern() {
		t.Fatal("expected fill to be a pattern")
	}

	hatch, ok := layer.style.Fill.Pattern.(*HatchPattern)
	if !ok {
		t.Fatal("expected HatchPattern")
	}

	expected := color.RGBA{0, 178, 0, 178}
	test.T(t, hatch.Fill.Color, expected)
}

// firstDashedLayer returns the first path layer that carries a dash pattern.
func firstDashedLayer(t *testing.T, c *Canvas) layer {
	t.Helper()
	for _, layers := range c.layers {
		for _, l := range layers {
			if l.style.IsDashed() {
				return l
			}
		}
	}
	t.Fatal("no dashed layer found")
	return layer{}
}

// TestParseSVGDashUnits checks that stroke-dasharray and stroke-dashoffset honor the SVG unit rules:
// unitless/px values are CSS user units, absolute units (mm, in, ...) keep their physical length, and
// percentages resolve against the viewport diagonal. All values end up in canvas millimeters after
// the view transformation.
func TestParseSVGDashUnits(t *testing.T) {
	// unitless width/height: one user unit is one millimeter
	t.Run("unitless values in user units", func(t *testing.T) {
		doc := `<svg width="100" height="100" viewBox="0 0 100 100"><line x1="0" y1="50" x2="100" y2="50" stroke="black" stroke-dasharray="4 2" stroke-dashoffset="1"/></svg>`
		c, err := ParseSVG(strings.NewReader(doc))
		test.Error(t, err)
		l := firstDashedLayer(t, c)
		test.Floats(t, l.style.Dashes, []float64{4.0, 2.0})
		test.Float(t, l.style.DashOffset, 1.0)
	})

	t.Run("px values equal user units", func(t *testing.T) {
		doc := `<svg width="100" height="100" viewBox="0 0 100 100"><line x1="0" y1="50" x2="100" y2="50" stroke="black" stroke-dasharray="4px 2px"/></svg>`
		c, err := ParseSVG(strings.NewReader(doc))
		test.Error(t, err)
		l := firstDashedLayer(t, c)
		test.Floats(t, l.style.Dashes, []float64{4.0, 2.0})
	})

	t.Run("absolute mm values keep physical length", func(t *testing.T) {
		doc := `<svg width="100mm" height="100mm" viewBox="0 0 100 100"><line x1="0" y1="50" x2="100" y2="50" stroke="black" stroke-dasharray="5mm 3mm" stroke-dashoffset="1mm"/></svg>`
		c, err := ParseSVG(strings.NewReader(doc))
		test.Error(t, err)
		l := firstDashedLayer(t, c)
		scale := math.Sqrt(math.Abs(l.m.Det())) // user-unit to millimeter scale implied by the view
		test.FloatDiff(t, l.style.Dashes[0]*scale, 5.0, 1e-9)
		test.FloatDiff(t, l.style.Dashes[1]*scale, 3.0, 1e-9)
		test.FloatDiff(t, l.style.DashOffset*scale, 1.0, 1e-9)
	})

	t.Run("in and comma-separated units", func(t *testing.T) {
		doc := `<svg width="100mm" height="100mm" viewBox="0 0 100 100"><line x1="0" y1="50" x2="100" y2="50" stroke="black" stroke-dasharray="1in,0.5in"/></svg>`
		c, err := ParseSVG(strings.NewReader(doc))
		test.Error(t, err)
		l := firstDashedLayer(t, c)
		scale := math.Sqrt(math.Abs(l.m.Det()))
		test.FloatDiff(t, l.style.Dashes[0]*scale, 25.4, 1e-9)
		test.FloatDiff(t, l.style.Dashes[1]*scale, 12.7, 1e-9)
	})

	t.Run("percentage against diagonal", func(t *testing.T) {
		doc := `<svg width="100" height="100" viewBox="0 0 100 100"><line x1="0" y1="50" x2="100" y2="50" stroke="black" stroke-dasharray="10%"/></svg>`
		c, err := ParseSVG(strings.NewReader(doc))
		test.Error(t, err)
		l := firstDashedLayer(t, c)
		// diagonal of a 100x100 user-unit viewport is 100 user units; 10% is 10 user units = 10mm
		test.FloatDiff(t, l.style.Dashes[0], 10.0, 1e-9)
	})

	t.Run("none clears dashes", func(t *testing.T) {
		doc := `<svg width="100" height="100" viewBox="0 0 100 100"><line x1="0" y1="50" x2="100" y2="50" stroke="black" stroke-dasharray="none"/></svg>`
		c, err := ParseSVG(strings.NewReader(doc))
		test.Error(t, err)
		for _, layers := range c.layers {
			for _, l := range layers {
				test.That(t, !l.style.IsDashed())
			}
		}
	})
}

// TestParseSVGCanvasSizeMM verifies that an SVG width/height in CSS pixels produces a canvas sized in
// millimeters rather than treating the pixel count as millimeters.
func TestParseSVGCanvasSizeMM(t *testing.T) {
	c, err := ParseSVG(strings.NewReader(`<svg width="96" height="192" viewBox="0 0 96 192"></svg>`))
	test.Error(t, err)
	test.FloatDiff(t, c.W, 25.4, 1e-9)
	test.FloatDiff(t, c.H, 50.8, 1e-9)

	c, err = ParseSVG(strings.NewReader(`<svg width="100mm" height="50mm" viewBox="0 0 100 50"></svg>`))
	test.Error(t, err)
	test.FloatDiff(t, c.W, 100.0, 1e-9)
	test.FloatDiff(t, c.H, 50.0, 1e-9)
}
