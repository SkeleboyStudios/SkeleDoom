package shaders

import (
	"image"
	"image/color"

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
