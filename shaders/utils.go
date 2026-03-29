package shaders

import (
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/gl"

	"image/color"
	"log"
	"math"
)

var (
	ViewShader = &viewShader{}
)

// PlayerOffset is the offset subtracted from the player's world position when
// computing wall/billboard-relative camera coordinates for 3D rendering.
// Exported so that systems outside this package (e.g. ItemSystem) can perform
// proximity checks in the same coordinate space as walls.
var PlayerOffset = engo.Point{X: 49, Y: 242}

type Wall struct {
	Line engo.Line
	Tex  *gl.Texture
	H    float32
}

func (w Wall) Texture() *gl.Texture { return w.Tex }

func (Wall) Width() float32                             { return 0 }
func (w Wall) Height() float32                          { return w.H }
func (Wall) View() (float32, float32, float32, float32) { return 0, 0, 1, 1 }
func (Wall) Close()                                     {}

// Billboard is a camera-facing rectangular sprite in the 3D view. It renders
// as a vertical quad that always faces the player (y-axis billboard).
// Pos is the world-map position in the same coordinate space as wall endpoints;
// W and H are the world-unit width and height of the sprite.
type Billboard struct {
	Pos  engo.Point
	W, H float32
	Tex  *gl.Texture
}

func (b Billboard) Texture() *gl.Texture                     { return b.Tex }
func (Billboard) Width() float32                             { return 0 }
func (b Billboard) Height() float32                          { return b.H }
func (Billboard) View() (float32, float32, float32, float32) { return 0, 0, 1, 1 }
func (Billboard) Close()                                     {}

func clipBehindPlayer(x0, y0, z0, x1, y1, z1 float32) (x2, y2, z2 float32) {
	d := y0 - y1
	if d == 0 {
		d = 1
	}
	s := y0 / d
	x2 = x0 + s*(x1-x0)
	y2 = y0 + s*(y1-y0)
	if y2 == 0 {
		y2 = 1
	}
	z2 = z0 + s*(z1-z0)
	return
}

func setBufferValue(buffer []float32, index int, value float32, changed *bool) {
	if buffer[index] != value {
		buffer[index] = value
		*changed = true
	}
}

// colorToFloat32 returns the float32 representation of the given color
func colorToFloat32(c color.Color) float32 {
	colorR, colorG, colorB, colorA := c.RGBA()
	colorR >>= 8
	colorG >>= 8
	colorB >>= 8
	colorA >>= 8

	red := colorR
	green := colorG << 8
	blue := colorB << 16
	alpha := colorA << 24

	return math.Float32frombits((alpha | blue | green | red) & 0xfeffffff)
}

func notImplemented(msg string) {
	warning(msg + " is not yet implemented on this platform")
}

func unsupportedType(v interface{}) {
	warning("type %T not supported", v)
}

func warning(format string, a ...interface{}) {
	log.Printf("[WARNING] "+format, a...)
}
