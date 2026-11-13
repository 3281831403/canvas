package canvas

import (
	"bytes"
	"fmt"
	"image/color"
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

// buildClassSVG returns an SVG with n rects that each use one of r classes, and a <style>
// element holding the class rules. The style element is placed at the given position:
// "start" before the shapes, "end" after them, or "both" in both places.
func buildClassSVG(n, r int, pos string) string {
	colors := []string{"#e74c3c", "#3498db", "#2ecc71", "#f1c40f", "#9b59b6", "#1abc9c"}
	var style strings.Builder
	style.WriteString("<style>")
	for i := 0; i < r; i++ {
		fmt.Fprintf(&style, ".c%d{fill:%s;stroke:%s;stroke-width:%d}", i, colors[i%len(colors)], colors[(i+3)%len(colors)], i%5+1)
	}
	style.WriteString("</style>")

	var sb strings.Builder
	sb.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="2000" height="2000">`)
	if pos == "start" || pos == "both" {
		sb.WriteString(style.String())
	}
	sb.WriteString(`<g id="art">`)
	for i := 0; i < n; i++ {
		fmt.Fprintf(&sb, `<rect class="c%d" x="%d" y="%d" width="18" height="18"/>`, i%r, (i%100)*20, (i/100)*20)
	}
	sb.WriteString(`</g>`)
	if pos == "end" || pos == "both" {
		sb.WriteString(style.String())
	}
	sb.WriteString(`</svg>`)
	return sb.String()
}

// TestParseSVGStylePosition checks that a <style> element applies to the entire document,
// no matter whether it occurs before or after the elements it styles.
func TestParseSVGStylePosition(t *testing.T) {
	const n, r = 500, 100
	ref, err := ParseSVG(strings.NewReader(buildClassSVG(n, r, "start")))
	test.Error(t, err)

	// the rules must actually be applied, so that the comparisons below are meaningful
	nonBlack := 0
	for _, layer := range ref.layers[0] {
		if layer.style.Fill.Color != Black {
			nonBlack++
		}
	}
	test.T(t, nonBlack, n)

	for _, pos := range []string{"end", "both"} {
		t.Run(pos, func(t *testing.T) {
			c, err := ParseSVG(strings.NewReader(buildClassSVG(n, r, pos)))
			test.Error(t, err)

			layers := c.layers[0]
			refLayers := ref.layers[0]
			test.T(t, len(layers), len(refLayers))
			for i := range refLayers {
				if layers[i].style.Fill != refLayers[i].style.Fill ||
					layers[i].style.Stroke != refLayers[i].style.Stroke ||
					layers[i].style.StrokeWidth != refLayers[i].style.StrokeWidth {
					t.Fatalf("layer %d differs from reference", i)
				}
			}
		})
	}
}

// TestParseSVGStyleLayerOrder checks that shapes keep their document order when a trailing
// style sheet is applied to them.
func TestParseSVGStyleLayerOrder(t *testing.T) {
	svg := `<svg width="100" height="100"><rect class="a" x="0" y="0" width="1" height="1"/><rect class="b" x="1" y="0" width="1" height="1"/><rect class="a" x="2" y="0" width="1" height="1"/><style>.a{fill:red}.b{fill:blue}</style></svg>`
	c, err := ParseSVG(strings.NewReader(svg))
	test.Error(t, err)

	layers := c.layers[0]
	test.T(t, len(layers), 3)
	test.T(t, layers[0].style.Fill.Color, color.RGBA{255, 0, 0, 255})
	test.T(t, layers[1].style.Fill.Color, color.RGBA{0, 0, 255, 255})
	test.T(t, layers[2].style.Fill.Color, color.RGBA{255, 0, 0, 255})
}

// TestParseSVGStyleSelectors checks that trailing style sheets work for every selector kind
// the parser supports, including descendant and child combinators.
func TestParseSVGStyleSelectors(t *testing.T) {
	style := `<style>
		#byid{fill:rgb(1,2,3)}
		.a{stroke:rgb(4,5,6)}
		rect.b{fill:rgb(7,8,9)}
		[data-k]{fill:rgb(10,11,12)}
		g > rect{z-index:1}
		g .d{fill:rgb(13,14,15)}
	</style>`
	body := `<g id="grp"><rect id="byid" class="a b" data-k="v" x="0" y="0" width="1" height="1"/><rect class="d" x="1" y="0" width="1" height="1"/></g>`
	svg := `<svg width="100" height="100">` + body + style + `</svg>`
	c, err := ParseSVG(strings.NewReader(svg))
	test.Error(t, err)

	layers := c.layers[0]
	test.T(t, len(layers), 2)
	// rect.b comes later than #byid and .a, so it wins for the first rect
	test.T(t, layers[0].style.Fill.Color, color.RGBA{7, 8, 9, 255})
	test.T(t, layers[0].style.Stroke.Color, color.RGBA{4, 5, 6, 255})
	test.T(t, layers[1].style.Fill.Color, color.RGBA{13, 14, 15, 255})
}

// TestParseSVGStylePrecedence checks that presentation attributes and style attributes
// override style sheet rules, and that the order of rules in the sheet is respected.
func TestParseSVGStylePrecedence(t *testing.T) {
	svg := `<svg width="100" height="100"><style>.a{fill:rgb(1,1,1)}.a{fill:rgb(2,2,2)}.b{fill:rgb(3,3,3)}</style>` +
		`<rect class="a" x="0" y="0" width="1" height="1"/>` +
		`<rect class="b" fill="rgb(4,4,4)" x="1" y="0" width="1" height="1"/>` +
		`<rect class="b" style="fill:rgb(5,5,5)" x="2" y="0" width="1" height="1"/></svg>`
	c, err := ParseSVG(strings.NewReader(svg))
	test.Error(t, err)

	layers := c.layers[0]
	test.T(t, len(layers), 3)
	test.T(t, layers[0].style.Fill.Color, color.RGBA{2, 2, 2, 255})
	test.T(t, layers[1].style.Fill.Color, color.RGBA{4, 4, 4, 255})
	test.T(t, layers[2].style.Fill.Color, color.RGBA{5, 5, 5, 255})
}

// TestParseSVGStyleInDefs checks that style sheets nested in <defs> (as emitted by Adobe
// Illustrator) apply to the rest of the document.
func TestParseSVGStyleInDefs(t *testing.T) {
	svg := `<svg width="100" height="100"><defs><style>.a{fill:rgb(20,21,22)}</style></defs><rect class="a" x="0" y="0" width="1" height="1"/></svg>`
	c, err := ParseSVG(strings.NewReader(svg))
	test.Error(t, err)

	test.T(t, c.layers[0][0].style.Fill.Color, color.RGBA{20, 21, 22, 255})
}

// BenchmarkParseSVGStyle measures parsing a document with many shapes and class rules. The
// sub-benchmarks differ only in where the style sheet is placed, so all of them should have
// the same cost.
func BenchmarkParseSVGStyle(b *testing.B) {
	for _, pos := range []string{"start", "end"} {
		data := []byte(buildClassSVG(20000, 2000, pos))
		b.Run(pos, func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := ParseSVG(bytes.NewReader(data)); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
