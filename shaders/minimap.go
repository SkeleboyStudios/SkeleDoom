package shaders

import (
	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"
	"github.com/EngoEngine/engo/math"
	"github.com/EngoEngine/gl"
)

// MinimapShader is a HUD shader (no camera) that clips all rendering to a
// configurable scissor rectangle. Assign it to minimap wall entities so that
// wall lines are clipped to the minimap bounding box instead of bleeding over
// the rest of the UI.
//
// Register it in the scene's Preload with common.AddShader(shaders.MinimapShader),
// then call SetClipRect from MapSystem.New() with the bounding box coordinates.
var MinimapShader = &minimapShader{}

type minimapShader struct {
	program *gl.Program

	indicesRectangles    []uint16
	indicesRectanglesVBO *gl.Buffer

	inPosition int
	inColor    int

	matrixProjection *gl.UniformLocation
	matrixView       *gl.UniformLocation
	matrixModel      *gl.UniformLocation

	projectionMatrix []float32
	viewMatrix       []float32
	modelMatrix      []float32

	lastBuffer *gl.Buffer

	camera *common.CameraSystem

	// Scissor rectangle in HUD game-unit coordinates (origin at top-left of
	// the screen). Set via SetClipRect; zero width/height disables scissoring.
	clipX, clipY, clipW, clipH float32
}

// SetClipRect sets the scissor rectangle in HUD game-unit coordinates (origin
// at the top-left of the screen). Call this from MapSystem.New() with the
// minimap bounding box position and dimensions so that wall lines drawn with
// this shader are clipped to those bounds.
func (s *minimapShader) SetClipRect(x, y, w, h float32) {
	s.clipX = x
	s.clipY = y
	s.clipW = w
	s.clipH = h
}

func (s *minimapShader) Setup(w *ecs.World) error {
	var err error
	s.program, err = common.LoadShader(`
attribute vec2 in_Position;
attribute vec4 in_Color;

uniform mat3 matrixProjection;
uniform mat3 matrixView;
uniform mat3 matrixModel;

varying vec4 var_Color;

void main() {
  var_Color = in_Color;

  vec3 matr = matrixProjection * matrixView * matrixModel * vec3(in_Position, 1.0);
  gl_Position = vec4(matr.xy, 0, matr.z);
}
`, `
#ifdef GL_ES
#define LOWP lowp
precision mediump float;
#else
#define LOWP
#endif

varying vec4 var_Color;

void main (void) {
  gl_FragColor = var_Color;
}`)

	if err != nil {
		return err
	}

	s.indicesRectangles = []uint16{0, 1, 2, 0, 2, 3}
	s.indicesRectanglesVBO = engo.Gl.CreateBuffer()
	engo.Gl.BindBuffer(engo.Gl.ELEMENT_ARRAY_BUFFER, s.indicesRectanglesVBO)
	engo.Gl.BufferData(engo.Gl.ELEMENT_ARRAY_BUFFER, s.indicesRectangles, engo.Gl.STATIC_DRAW)

	s.inPosition = engo.Gl.GetAttribLocation(s.program, "in_Position")
	s.inColor = engo.Gl.GetAttribLocation(s.program, "in_Color")

	s.matrixProjection = engo.Gl.GetUniformLocation(s.program, "matrixProjection")
	s.matrixView = engo.Gl.GetUniformLocation(s.program, "matrixView")
	s.matrixModel = engo.Gl.GetUniformLocation(s.program, "matrixModel")

	s.projectionMatrix = make([]float32, 9)
	s.projectionMatrix[8] = 1

	s.viewMatrix = make([]float32, 9)
	s.viewMatrix[0] = 1
	s.viewMatrix[4] = 1
	s.viewMatrix[8] = 1

	s.modelMatrix = make([]float32, 9)
	s.modelMatrix[0] = 1
	s.modelMatrix[4] = 1
	s.modelMatrix[8] = 1

	return nil
}

func (s *minimapShader) Pre() {
	engo.Gl.Enable(engo.Gl.BLEND)
	engo.Gl.BlendFunc(engo.Gl.SRC_ALPHA, engo.Gl.ONE_MINUS_SRC_ALPHA)

	engo.Gl.UseProgram(s.program)
	engo.Gl.EnableVertexAttribArray(s.inPosition)
	engo.Gl.EnableVertexAttribArray(s.inColor)

	// HUD projection: no camera influence, origin mapped to top-left.
	if engo.ScaleOnResize() {
		s.projectionMatrix[0] = 1 / (engo.GameWidth() / 2)
		s.projectionMatrix[4] = 1 / (-engo.GameHeight() / 2)
	} else {
		s.projectionMatrix[0] = 1 / (engo.CanvasWidth() / (2 * engo.CanvasScale()))
		s.projectionMatrix[4] = 1 / (-engo.CanvasHeight() / (2 * engo.CanvasScale()))
	}

	if s.camera != nil {
		s.viewMatrix[1], s.viewMatrix[0] = math.Sincos(s.camera.Angle() * math.Pi / 180)
		s.viewMatrix[3] = -s.viewMatrix[1]
		s.viewMatrix[4] = s.viewMatrix[0]
		s.viewMatrix[6] = -s.camera.X()
		s.viewMatrix[7] = -s.camera.Y()
		s.viewMatrix[8] = s.camera.Z()
	} else {
		s.viewMatrix[6] = -1 / s.projectionMatrix[0]
		s.viewMatrix[7] = 1 / s.projectionMatrix[4]
	}

	engo.Gl.UniformMatrix3fv(s.matrixProjection, false, s.projectionMatrix)
	engo.Gl.UniformMatrix3fv(s.matrixView, false, s.viewMatrix)

	// Enable GL scissor test to clip rendering to the minimap bounding box.
	//
	// HUD coordinates use a top-left origin with y increasing downward.
	// GL scissor uses a bottom-left origin (physical pixels) with y increasing
	// upward, so we must flip the y axis and scale from logical to physical
	// pixels.
	if s.clipW > 0 && s.clipH > 0 {
		var scaleX, scaleY float32
		if engo.ScaleOnResize() {
			scaleX = engo.CanvasWidth() / engo.GameWidth()
			scaleY = engo.CanvasHeight() / engo.GameHeight()
		} else {
			scaleX = engo.CanvasScale()
			scaleY = engo.CanvasScale()
		}
		// Bottom-left corner of the scissor rect in physical pixels.
		fbX := int(s.clipX * scaleX)
		fbY := int(engo.CanvasHeight() - (s.clipY+s.clipH)*scaleY)
		fbW := int(s.clipW * scaleX)
		fbH := int(s.clipH * scaleY)

		engo.Gl.Enable(engo.Gl.SCISSOR_TEST)
		engo.Gl.Scissor(fbX, fbY, fbW, fbH)
	}
}

func (s *minimapShader) updateBuffer(ren *common.RenderComponent, space *common.SpaceComponent) {
	if len(ren.BufferContent) == 0 {
		ren.BufferContent = make([]float32, s.computeBufferSize(ren.Drawable))
	}
	if changed := s.generateBufferContent(ren, space, ren.BufferContent); !changed {
		return
	}

	if ren.Buffer == nil {
		ren.Buffer = engo.Gl.CreateBuffer()
	}
	engo.Gl.BindBuffer(engo.Gl.ARRAY_BUFFER, ren.Buffer)
	engo.Gl.BufferData(engo.Gl.ARRAY_BUFFER, ren.BufferContent, engo.Gl.STATIC_DRAW)
}

func (s *minimapShader) computeBufferSize(draw common.Drawable) int {
	switch shape := draw.(type) {
	case common.Triangle:
		return 65
	case common.Rectangle:
		return 90
	case common.Circle:
		return 3260
	case common.ComplexTriangles:
		return len(shape.Points) * 6
	case common.Curve:
		return 1800
	default:
		return 0
	}
}

func (s *minimapShader) generateBufferContent(ren *common.RenderComponent, space *common.SpaceComponent, buffer []float32) bool {
	w := space.Width
	h := space.Height

	var changed bool

	tint := colorToFloat32(ren.Color)

	switch shape := ren.Drawable.(type) {
	case common.Triangle:
		switch shape.TriangleType {
		case common.TriangleIsosceles:
			setBufferValue(buffer, 0, w/2, &changed)
			setBufferValue(buffer, 2, tint, &changed)

			setBufferValue(buffer, 3, w, &changed)
			setBufferValue(buffer, 4, h, &changed)
			setBufferValue(buffer, 5, tint, &changed)

			setBufferValue(buffer, 7, h, &changed)
			setBufferValue(buffer, 8, tint, &changed)

			if shape.BorderWidth > 0 {
				borderTint := colorToFloat32(shape.BorderColor)
				b := shape.BorderWidth
				// Use sinV/cosV to avoid shadowing the receiver s.
				sinV, cosV := math.Sincos(math.Atan(2 * h / w))

				pts := [][]float32{
					// Left
					{w / 2, 0},
					{0, h},
					{b, h},
					{b, h},
					{(w / 2) + b*cosV, b * sinV},
					{w / 2, 0},
					// Right
					{w / 2, 0},
					{w, h},
					{w - b, h},
					{w - b, h},
					{(w / 2) - b*cosV, b * sinV},
					{w / 2, 0},
					// Bottom
					{0, h},
					{w, h},
					{b * cosV, h - b*sinV},
					{b * cosV, h - b*sinV},
					{w - b*cosV, h - b*sinV},
					{w, h},
				}

				for i, p := range pts {
					setBufferValue(buffer, 9+3*i, p[0], &changed)
					setBufferValue(buffer, 10+3*i, p[1], &changed)
					setBufferValue(buffer, 11+3*i, borderTint, &changed)
				}
			}

		case common.TriangleRight:
			setBufferValue(buffer, 2, tint, &changed)

			setBufferValue(buffer, 3, w, &changed)
			setBufferValue(buffer, 4, h, &changed)
			setBufferValue(buffer, 5, tint, &changed)

			setBufferValue(buffer, 7, h, &changed)
			setBufferValue(buffer, 8, tint, &changed)

			if shape.BorderWidth > 0 {
				borderTint := colorToFloat32(shape.BorderColor)
				b := shape.BorderWidth

				pts := [][]float32{
					// Left
					{0, 0},
					{0, h},
					{b, h},
					{b, h},
					{b, b * h / w},
					{0, 0},
					// Right
					{0, 0},
					{w, h},
					{w - b, h},
					{w - b, h},
					{0, b},
					{0, 0},
					// Bottom
					{0, h},
					{w, h},
					{w - b*w/h, h - b},
					{w - b*w/h, h - b},
					{0, h - b},
					{0, h},
				}

				for i, p := range pts {
					setBufferValue(buffer, 9+3*i, p[0], &changed)
					setBufferValue(buffer, 10+3*i, p[1], &changed)
					setBufferValue(buffer, 11+3*i, borderTint, &changed)
				}
			}
		}

	case common.Circle:
		if shape.Arc == 0 {
			shape.Arc = 360
		}
		theta := float32(2.0*math.Pi/360.0) * shape.Arc / 360
		cx := w / 2
		bx := shape.BorderWidth
		cy := h / 2
		by := shape.BorderWidth
		var borderTint float32
		hasBorder := shape.BorderWidth > 0
		if hasBorder {
			borderTint = colorToFloat32(shape.BorderColor)
		}
		setBufferValue(buffer, 0, w/2, &changed)
		setBufferValue(buffer, 1, h/2, &changed)
		setBufferValue(buffer, 2, tint, &changed)
		if hasBorder {
			setBufferValue(buffer, 1086, w-bx, &changed)
			setBufferValue(buffer, 1087, h/2, &changed)
			setBufferValue(buffer, 1088, borderTint, &changed)
		}
		for i := 1; i < 362; i++ {
			sinV, cosV := math.Sincos(float32(i) * theta)
			setBufferValue(buffer, i*3, cx+(cx-bx)*cosV, &changed)
			setBufferValue(buffer, i*3+1, cy+(cy-by)*sinV, &changed)
			setBufferValue(buffer, i*3+2, tint, &changed)
			if hasBorder {
				setBufferValue(buffer, i*6+1086, cx+cx*cosV, &changed)
				setBufferValue(buffer, i*6+1087, cy+cy*sinV, &changed)
				setBufferValue(buffer, i*6+1088, borderTint, &changed)
				setBufferValue(buffer, i*6+1089, cx+(cx-bx)*cosV, &changed)
				setBufferValue(buffer, i*6+1090, cy+(cy-by)*sinV, &changed)
				setBufferValue(buffer, i*6+1091, borderTint, &changed)
			}
		}

	case common.Rectangle:
		setBufferValue(buffer, 2, tint, &changed)

		setBufferValue(buffer, 3, w, &changed)
		setBufferValue(buffer, 5, tint, &changed)

		setBufferValue(buffer, 6, w, &changed)
		setBufferValue(buffer, 7, h, &changed)
		setBufferValue(buffer, 8, tint, &changed)

		setBufferValue(buffer, 9, w, &changed)
		setBufferValue(buffer, 10, h, &changed)
		setBufferValue(buffer, 11, tint, &changed)

		setBufferValue(buffer, 13, h, &changed)
		setBufferValue(buffer, 14, tint, &changed)

		setBufferValue(buffer, 17, tint, &changed)

		if shape.BorderWidth > 0 {
			borderTint := colorToFloat32(shape.BorderColor)
			b := shape.BorderWidth
			pts := [][]float32{
				// Top
				{0, 0}, {w, 0}, {w, b}, {w, b}, {0, b}, {0, 0},
				// Right
				{w - b, b}, {w, b}, {w, h - b}, {w, h - b}, {w - b, h - b}, {w - b, b},
				// Bottom
				{w, h - b}, {w, h}, {0, h}, {0, h}, {0, h - b}, {w, h - b},
				// Left
				{0, b}, {b, b}, {b, h - b}, {b, h - b}, {0, h - b}, {0, b},
			}
			for i, p := range pts {
				setBufferValue(buffer, 18+3*i, p[0], &changed)
				setBufferValue(buffer, 19+3*i, p[1], &changed)
				setBufferValue(buffer, 20+3*i, borderTint, &changed)
			}
		}

	case common.ComplexTriangles:
		var index int
		for _, point := range shape.Points {
			setBufferValue(buffer, index, point.X*w, &changed)
			setBufferValue(buffer, index+1, point.Y*h, &changed)
			setBufferValue(buffer, index+2, tint, &changed)
			index += 3
		}

		if shape.BorderWidth > 0 {
			borderTint := colorToFloat32(shape.BorderColor)
			for _, point := range shape.Points {
				setBufferValue(buffer, index, point.X*w, &changed)
				setBufferValue(buffer, index+1, point.Y*h, &changed)
				setBufferValue(buffer, index+2, borderTint, &changed)
				index += 3
			}
		}

	case common.Curve:
		lw := shape.LineWidth
		pts := make([][]float32, 0)
		for i := 0; i < 100; i++ {
			pt := make([]float32, 2)
			t := float32(i) / 100
			switch len(shape.Points) {
			case 0:
				pt[0] = t * w
				pt[1] = t * h
			case 1:
				pt[0] = 2*(1-t)*t*shape.Points[0].X + t*t*w
				pt[1] = 2*(1-t)*t*shape.Points[0].Y + t*t*h
			case 2:
				pt[0] = 3*(1-t)*(1-t)*t*shape.Points[0].X + 3*(1-t)*t*t*shape.Points[1].X + t*t*t*w
				pt[1] = 3*(1-t)*(1-t)*t*shape.Points[0].Y + 3*(1-t)*t*t*shape.Points[1].Y + t*t*t*h
			default:
				unsupportedType(ren.Drawable)
			}
			pts = append(pts, pt)
		}
		for i := 0; i < len(pts)-1; i++ {
			num := pts[i+1][1] - pts[i][1]
			if engo.FloatEqual(num, 0) { // horizontal segment
				setBufferValue(buffer, i*18, pts[i][0], &changed)
				setBufferValue(buffer, i*18+1, pts[i][1]-lw, &changed)
				setBufferValue(buffer, i*18+2, tint, &changed)
				setBufferValue(buffer, i*18+3, pts[i+1][0], &changed)
				setBufferValue(buffer, i*18+4, pts[i+1][1]-lw, &changed)
				setBufferValue(buffer, i*18+5, tint, &changed)
				setBufferValue(buffer, i*18+6, pts[i+1][0], &changed)
				setBufferValue(buffer, i*18+7, pts[i+1][1]+lw, &changed)
				setBufferValue(buffer, i*18+8, tint, &changed)
				setBufferValue(buffer, i*18+9, pts[i+1][0], &changed)
				setBufferValue(buffer, i*18+10, pts[i+1][1]+lw, &changed)
				setBufferValue(buffer, i*18+11, tint, &changed)
				setBufferValue(buffer, i*18+12, pts[i][0], &changed)
				setBufferValue(buffer, i*18+13, pts[i][1]+lw, &changed)
				setBufferValue(buffer, i*18+14, tint, &changed)
				setBufferValue(buffer, i*18+15, pts[i][0], &changed)
				setBufferValue(buffer, i*18+16, pts[i][1]-lw, &changed)
				setBufferValue(buffer, i*18+17, tint, &changed)
				continue
			}
			denom := pts[i+1][0] - pts[i+1][0] // mirrors upstream legacyShader
			if engo.FloatEqual(denom, 0) {     // vertical segment
				setBufferValue(buffer, i*18, pts[i+1][0]-lw, &changed)
				setBufferValue(buffer, i*18+1, pts[i+1][1], &changed)
				setBufferValue(buffer, i*18+2, tint, &changed)
				setBufferValue(buffer, i*18+3, pts[i+1][0]+lw, &changed)
				setBufferValue(buffer, i*18+4, pts[i+1][1], &changed)
				setBufferValue(buffer, i*18+5, tint, &changed)
				setBufferValue(buffer, i*18+6, pts[i][0]+lw, &changed)
				setBufferValue(buffer, i*18+7, pts[i][1], &changed)
				setBufferValue(buffer, i*18+8, tint, &changed)
				setBufferValue(buffer, i*18+9, pts[i][0]+lw, &changed)
				setBufferValue(buffer, i*18+10, pts[i][1], &changed)
				setBufferValue(buffer, i*18+11, tint, &changed)
				setBufferValue(buffer, i*18+12, pts[i][0]-lw, &changed)
				setBufferValue(buffer, i*18+13, pts[i][1], &changed)
				setBufferValue(buffer, i*18+14, tint, &changed)
				setBufferValue(buffer, i*18+15, pts[i+1][0]-lw, &changed)
				setBufferValue(buffer, i*18+16, pts[i+1][1], &changed)
				setBufferValue(buffer, i*18+17, tint, &changed)
				continue
			}
			m1 := num / denom
			m2 := -1 / m1
			dx := math.Sqrt(lw*lw/(1+m2*m2)) / 2
			dy := m2 * dx
			setBufferValue(buffer, i*18, pts[i][0]-dx, &changed)
			setBufferValue(buffer, i*18+1, pts[i][1]-dy, &changed)
			setBufferValue(buffer, i*18+2, tint, &changed)
			setBufferValue(buffer, i*18+3, pts[i+1][0]-dx, &changed)
			setBufferValue(buffer, i*18+4, pts[i+1][1]-dy, &changed)
			setBufferValue(buffer, i*18+5, tint, &changed)
			setBufferValue(buffer, i*18+6, pts[i+1][0]+dx, &changed)
			setBufferValue(buffer, i*18+7, pts[i+1][1]+dy, &changed)
			setBufferValue(buffer, i*18+8, tint, &changed)
			setBufferValue(buffer, i*18+9, pts[i+1][0]+dx, &changed)
			setBufferValue(buffer, i*18+10, pts[i+1][1]+dy, &changed)
			setBufferValue(buffer, i*18+11, tint, &changed)
			setBufferValue(buffer, i*18+12, pts[i][0]+dx, &changed)
			setBufferValue(buffer, i*18+13, pts[i][1]+dy, &changed)
			setBufferValue(buffer, i*18+14, tint, &changed)
			setBufferValue(buffer, i*18+15, pts[i][0]-dx, &changed)
			setBufferValue(buffer, i*18+16, pts[i][1]-dy, &changed)
			setBufferValue(buffer, i*18+17, tint, &changed)
		}

	default:
		unsupportedType(ren.Drawable)
	}

	return changed
}

func (s *minimapShader) Draw(ren *common.RenderComponent, space *common.SpaceComponent) {
	if s.lastBuffer != ren.Buffer || ren.Buffer == nil {
		s.updateBuffer(ren, space)

		engo.Gl.BindBuffer(engo.Gl.ARRAY_BUFFER, ren.Buffer)
		engo.Gl.VertexAttribPointer(s.inPosition, 2, engo.Gl.FLOAT, false, 12, 0)
		engo.Gl.VertexAttribPointer(s.inColor, 4, engo.Gl.UNSIGNED_BYTE, true, 12, 8)

		s.lastBuffer = ren.Buffer
	}

	if space.Rotation != 0 {
		sinV, cosV := math.Sincos(space.Rotation * math.Pi / 180)
		s.modelMatrix[0] = ren.Scale.X * engo.GetGlobalScale().X * cosV
		s.modelMatrix[1] = ren.Scale.X * engo.GetGlobalScale().X * sinV
		s.modelMatrix[3] = ren.Scale.Y * engo.GetGlobalScale().Y * -sinV
		s.modelMatrix[4] = ren.Scale.Y * engo.GetGlobalScale().Y * cosV
	} else {
		s.modelMatrix[0] = ren.Scale.X * engo.GetGlobalScale().X
		s.modelMatrix[1] = 0
		s.modelMatrix[3] = 0
		s.modelMatrix[4] = ren.Scale.Y * engo.GetGlobalScale().Y
	}

	s.modelMatrix[6] = space.Position.X * engo.GetGlobalScale().X
	s.modelMatrix[7] = space.Position.Y * engo.GetGlobalScale().Y

	engo.Gl.UniformMatrix3fv(s.matrixModel, false, s.modelMatrix)

	switch shape := ren.Drawable.(type) {
	case common.Triangle:
		num := 3
		if shape.BorderWidth > 0 {
			num = 21
		}
		engo.Gl.DrawArrays(engo.Gl.TRIANGLES, 0, num)
	case common.Rectangle:
		num := 6
		if shape.BorderWidth > 0 {
			num = 30
		}
		engo.Gl.DrawArrays(engo.Gl.TRIANGLES, 0, num)
	case common.Circle:
		if shape.BorderWidth > 0 {
			engo.Gl.DrawArrays(engo.Gl.TRIANGLE_STRIP, 364, 722)
		}
		engo.Gl.DrawArrays(engo.Gl.TRIANGLE_FAN, 0, 362)
	case common.ComplexTriangles:
		engo.Gl.DrawArrays(engo.Gl.TRIANGLES, 0, len(shape.Points))
		if shape.BorderWidth > 0 {
			// HUD mode: border width is in game units, no camera z-scale needed.
			engo.Gl.LineWidth(shape.BorderWidth)
			engo.Gl.DrawArrays(engo.Gl.LINE_LOOP, len(shape.Points), len(shape.Points))
		}
	case common.Curve:
		engo.Gl.DrawArrays(engo.Gl.TRIANGLES, 0, 600)
	default:
		unsupportedType(ren.Drawable)
	}
}

func (s *minimapShader) Post() {
	s.lastBuffer = nil

	engo.Gl.DisableVertexAttribArray(s.inPosition)
	engo.Gl.DisableVertexAttribArray(s.inColor)

	engo.Gl.BindBuffer(engo.Gl.ARRAY_BUFFER, nil)
	engo.Gl.BindBuffer(engo.Gl.ELEMENT_ARRAY_BUFFER, nil)

	// Always disable scissor test so subsequent shaders draw unclipped.
	engo.Gl.Disable(engo.Gl.SCISSOR_TEST)
	engo.Gl.Disable(engo.Gl.BLEND)
}

func (s *minimapShader) SetCamera(c *common.CameraSystem) {
	s.camera = c
}
