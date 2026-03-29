package shaders

import (
	"image"
	"image/color"
	"math"

	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/gl"
)

// CreateBrickTexture generates a simple brick-pattern texture procedurally and
// uploads it to the GPU. width and height should be powers of two (e.g. 64, 128).
// The returned *gl.Texture is ready to bind immediately.
//
// Must be called after the OpenGL context is initialised (i.e. from Setup, not Preload).
func CreateBrickTexture(width, height int) *gl.Texture {
	img := generateBrickImage(width, height)
	return uploadRGBATexture(img)
}

// generateBrickImage produces an *image.RGBA with a repeating brick pattern.
func generateBrickImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Palette
	mortar := color.RGBA{R: 160, G: 140, B: 130, A: 255}
	brickA := color.RGBA{R: 180, G: 70, B: 45, A: 255}
	brickB := color.RGBA{R: 155, G: 60, B: 38, A: 255}
	brickC := color.RGBA{R: 170, G: 75, B: 50, A: 255}

	brickPalette := []color.RGBA{brickA, brickB, brickC}

	const (
		brickW  = 32 // width of a single brick in pixels
		brickH  = 14 // height of a single brick in pixels
		mortarW = 2  // horizontal gap (end caps)
		mortarH = 2  // vertical gap
	)
	rowH := brickH + mortarH

	for y := 0; y < height; y++ {
		row := y / rowH
		isInMortarRow := (y % rowH) >= brickH

		// Alternate rows are offset by half a brick width for the classic running-bond pattern.
		offset := 0
		if row%2 == 1 {
			offset = (brickW + mortarW) / 2
		}

		for x := 0; x < width; x++ {
			if isInMortarRow {
				img.SetRGBA(x, y, mortar)
				continue
			}

			lx := (x + offset) % (brickW + mortarW)
			if lx >= brickW {
				// Vertical mortar joint
				img.SetRGBA(x, y, mortar)
				continue
			}

			// Pick a deterministic colour per brick so adjacent bricks differ slightly.
			brickIndex := ((x+offset)/(brickW+mortarW) + row*7) % len(brickPalette)

			// Add a subtle horizontal gradient across the brick face.
			base := brickPalette[brickIndex]
			shade := float32(lx) / float32(brickW)
			dr := int32(float32(base.R) - shade*12)
			dg := int32(float32(base.G) - shade*5)
			db := int32(float32(base.B) - shade*3)

			img.SetRGBA(x, y, color.RGBA{
				R: clampU8(dr),
				G: clampU8(dg),
				B: clampU8(db),
				A: 255,
			})
		}
	}

	return img
}

// uploadRGBATexture uploads an *image.RGBA to a new OpenGL texture object.
func uploadRGBATexture(img *image.RGBA) *gl.Texture {

	tex := engo.Gl.CreateTexture()
	engo.Gl.BindTexture(engo.Gl.TEXTURE_2D, tex)

	// Repeat in both directions so UVs > 1 tile naturally.
	engo.Gl.TexParameteri(engo.Gl.TEXTURE_2D, engo.Gl.TEXTURE_WRAP_S, engo.Gl.REPEAT)
	engo.Gl.TexParameteri(engo.Gl.TEXTURE_2D, engo.Gl.TEXTURE_WRAP_T, engo.Gl.REPEAT)

	// Nearest-neighbour filtering keeps the chunky pixel look at close range;
	// nearest mag filter preserves the pixel art look up close.
	engo.Gl.TexParameteri(engo.Gl.TEXTURE_2D, engo.Gl.TEXTURE_MIN_FILTER, engo.Gl.LINEAR)
	engo.Gl.TexParameteri(engo.Gl.TEXTURE_2D, engo.Gl.TEXTURE_MAG_FILTER, engo.Gl.NEAREST)

	// The engo GL wrapper extracts width/height from the image itself.
	// Signature: TexImage2D(target, level, internalFormat, format, kind int, data interface{})
	engo.Gl.TexImage2D(
		engo.Gl.TEXTURE_2D,
		0,
		engo.Gl.RGBA,
		engo.Gl.RGBA,
		engo.Gl.UNSIGNED_BYTE,
		img,
	)

	// Leave no texture bound so subsequent code starts from a clean state.
	engo.Gl.BindTexture(engo.Gl.TEXTURE_2D, nil)

	return tex
}

// clampU8 clamps an int32 to the [0, 255] range and returns a uint8.
func clampU8(v int32) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

// CreatePotionTexture generates a pixel-art green health potion texture and
// uploads it to the GPU. size should be a power of two (e.g. 64).
// Must be called after the OpenGL context is initialised (i.e. from Setup).
func CreatePotionTexture(size int) *gl.Texture {
	img := generatePotionImage(size)
	return uploadRGBATexture(img)
}

// generatePotionImage produces an *image.RGBA with a pixel-art potion sprite.
// The design scales with size; it is tuned for 64×64.
func generatePotionImage(size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Clear to fully transparent.
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.SetRGBA(x, y, color.RGBA{})
		}
	}

	// Scale helper: map a coordinate designed for 64-px to the actual size.
	sc := func(v float64) float64 { return v * float64(size) / 64.0 }

	// ── Colours ──────────────────────────────────────────────────────────
	outline := color.RGBA{0x12, 0x3d, 0x1a, 0xff}
	liquid := color.RGBA{0x1e, 0xd4, 0x3c, 0xff}
	glassTop := color.RGBA{0x90, 0xf0, 0xa8, 0xcc}
	highlight := color.RGBA{0xd4, 0xff, 0xdc, 0xcc}
	cork := color.RGBA{0xcc, 0x99, 0x55, 0xff}
	corkLine := color.RGBA{0x7a, 0x4e, 0x1a, 0xff}

	// ── Flask body (circle centred at 32,44 r=18) ────────────────────────
	bcx := sc(32)
	bcy := sc(44)
	br := sc(18)

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			d := math.Sqrt((px-bcx)*(px-bcx) + (py-bcy)*(py-bcy))
			if d < br-sc(2) {
				// Fill: top ~35 % is glass tint, rest is liquid.
				relY := (py - (bcy - br)) / (2 * br)
				if relY < 0.35 {
					img.SetRGBA(x, y, glassTop)
				} else {
					img.SetRGBA(x, y, liquid)
				}
			} else if d < br {
				img.SetRGBA(x, y, outline)
			}
		}
	}

	// ── Neck (x 27–37, y 26–36 in 64-space) ─────────────────────────────
	nx0 := int(sc(27))
	nx1 := int(sc(37))
	ny0 := int(sc(26))
	ny1 := int(sc(36))

	for y := ny0; y <= ny1 && y < size; y++ {
		for x := nx0; x <= nx1 && x < size; x++ {
			if x < 0 || y < 0 {
				continue
			}
			if x == nx0 || x == nx1 || y == ny0 {
				img.SetRGBA(x, y, outline)
			} else {
				relY := float64(y-ny0) / float64(ny1-ny0+1)
				if relY < 0.4 {
					img.SetRGBA(x, y, glassTop)
				} else {
					img.SetRGBA(x, y, liquid)
				}
			}
		}
	}

	// ── Cork (x 25–39, y 16–26 in 64-space) ─────────────────────────────
	kx0 := int(sc(25))
	kx1 := int(sc(39))
	ky0 := int(sc(16))
	ky1 := int(sc(26))

	for y := ky0; y <= ky1 && y < size; y++ {
		for x := kx0; x <= kx1 && x < size; x++ {
			if x < 0 || y < 0 {
				continue
			}
			if x == kx0 || x == kx1 || y == ky0 || y == ky1 {
				img.SetRGBA(x, y, corkLine)
			} else {
				img.SetRGBA(x, y, cork)
			}
		}
	}

	// ── Specular highlight (small ellipse, upper-left of body) ───────────
	hlcx := bcx - br*0.36
	hlcy := bcy - br*0.36
	hlrx := br * 0.20
	hlry := br * 0.13

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			dx := (px - hlcx) / hlrx
			dy := (py - hlcy) / hlry
			if dx*dx+dy*dy < 1.0 {
				if img.RGBAAt(x, y).A > 0 {
					img.SetRGBA(x, y, highlight)
				}
			}
		}
	}

	// ── Bubbles inside liquid area ───────────────────────────────────────
	type bubble struct{ cx, cy, r float64 }
	bubbles := []bubble{
		{bcx + br*0.10, bcy + br*0.20, br * 0.08},
		{bcx - br*0.22, bcy + br*0.08, br * 0.06},
		{bcx + br*0.30, bcy - br*0.02, br * 0.07},
	}
	bubbleColor := color.RGBA{0x55, 0xff, 0x88, 0xff}
	for _, b := range bubbles {
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				px, py := float64(x)+0.5, float64(y)+0.5
				dx := px - b.cx
				dy := py - b.cy
				if dx*dx+dy*dy < b.r*b.r {
					if img.RGBAAt(x, y) == liquid {
						img.SetRGBA(x, y, bubbleColor)
					}
				}
			}
		}
	}

	return img
}
