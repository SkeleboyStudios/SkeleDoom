package shaders

import (
	"image/color"

	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"
	"github.com/EngoEngine/engo/math"
	"github.com/EngoEngine/gl"
)

type viewShader struct {
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

	player       *common.SpaceComponent
	playerOffset engo.Point
	fov          float32
}

func (s *viewShader) Setup(w *ecs.World) error {
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

	// Create and populate indices buffer
	s.indicesRectangles = []uint16{0, 1, 2, 0, 2, 3}
	s.indicesRectanglesVBO = engo.Gl.CreateBuffer()
	engo.Gl.BindBuffer(engo.Gl.ELEMENT_ARRAY_BUFFER, s.indicesRectanglesVBO)
	engo.Gl.BufferData(engo.Gl.ELEMENT_ARRAY_BUFFER, s.indicesRectangles, engo.Gl.STATIC_DRAW)

	// Define things that should be read from the texture buffer
	s.inPosition = engo.Gl.GetAttribLocation(s.program, "in_Position")
	s.inColor = engo.Gl.GetAttribLocation(s.program, "in_Color")

	// Define things that should be set per draw
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

	s.fov = 200
	s.playerOffset = engo.Point{X: 49, Y: 242}

	return nil
}

func (s *viewShader) Pre() {
	engo.Gl.Enable(engo.Gl.BLEND)
	engo.Gl.BlendFunc(engo.Gl.SRC_ALPHA, engo.Gl.ONE_MINUS_SRC_ALPHA)

	// Bind shader and buffer, enable attributes
	engo.Gl.UseProgram(s.program)
	engo.Gl.EnableVertexAttribArray(s.inPosition)
	engo.Gl.EnableVertexAttribArray(s.inColor)

	if engo.ScaleOnResize() {
		s.projectionMatrix[0] = 1 / (engo.GameWidth() / 2)
		s.projectionMatrix[4] = 1 / (-engo.GameHeight() / 2)
	} else {
		s.projectionMatrix[0] = 1 / (engo.CanvasWidth() / (2 * engo.CanvasScale()))
		s.projectionMatrix[4] = 1 / (-engo.CanvasHeight() / (2 * engo.CanvasScale()))
	}

	s.viewMatrix[6] = -1 / s.projectionMatrix[0]
	s.viewMatrix[7] = 1 / s.projectionMatrix[4]

	engo.Gl.UniformMatrix3fv(s.matrixProjection, false, s.projectionMatrix)
	engo.Gl.UniformMatrix3fv(s.matrixView, false, s.viewMatrix)
}

func (s *viewShader) updateBuffer(ren *common.RenderComponent, space *common.SpaceComponent) {
	if len(ren.BufferContent) == 0 {
		ren.BufferContent = make([]float32, s.computeBufferSize(ren.Drawable)) // because we add at most this many elements to it
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

func (s *viewShader) computeBufferSize(draw common.Drawable) int {
	switch draw.(type) {
	case Wall:
		return 20
	default:
		return 0
	}
}

func (s *viewShader) generateBufferContent(ren *common.RenderComponent, space *common.SpaceComponent, buffer []float32) bool {
	var changed bool

	tint := colorToFloat32(ren.Color)
	tint1 := colorToFloat32(color.RGBA{255, 0, 0, 255})
	tint2 := colorToFloat32(color.White)
	tint3 := colorToFloat32(color.Black)
	w := engo.GameWidth()
	h := engo.GameHeight()

	switch d := ren.Drawable.(type) {
	case Wall:
		sin, cos := math.Sincos((s.player.Rotation) * math.Pi / 180)
		p1 := d.Line.P1
		p2 := d.Line.P2
		p1X := (p1.X - (s.player.Position.X - s.playerOffset.X))
		p1Y := (p1.Y + (s.player.Position.Y - s.playerOffset.Y))
		p2X := (p2.X - (s.player.Position.X - s.playerOffset.X))
		p2Y := (p2.Y + (s.player.Position.Y - s.playerOffset.Y))
		x0 := (p1X*cos - p1Y*sin)
		y0 := (p1Y*cos + p1X*sin)
		z0 := -1 * s.player.Height
		x1 := (p2X*cos - p2Y*sin)
		y1 := (p2Y*cos + p2X*sin)
		z1 := -1 * s.player.Height
		x2 := x0
		y2 := y0
		z2 := -1*s.player.Height + d.H
		x3 := x1
		y3 := y1
		z3 := -1*s.player.Height + d.H

		//clipping behind player
		if y0 < 1 && y1 < 1 {
			x0, y0, z0 = 0, 1, 0
			x1, y1, z1 = 0, 1, 0
			x2, y2, z2 = 0, 1, 0
			x3, y3, z3 = 0, 1, 0
		} else if y0 < 1 {
			x0, y0, z0 = clipBehindPlayer(x0, y0, z0, x1, y1, z1)
			x2, y2, z2 = clipBehindPlayer(x2, y2, z2, x3, y3, z3)
		} else if y1 < 1 {
			x1, y1, z1 = clipBehindPlayer(x1, y1, z1, x0, y0, z0)
			x3, y3, z3 = clipBehindPlayer(x3, y3, z3, x2, y2, z2)
		}

		//convert to screen coordinates
		wx0 := ((x0 * s.fov / y0) + w/2)
		wy0 := ((z0 * s.fov / y0) + h/2)
		wx1 := ((x1 * s.fov / y1) + w/2)
		wy1 := ((z1 * s.fov / y1) + h/2)
		wx2 := ((x2 * s.fov / y2) + w/2)
		wy2 := ((z2 * s.fov / y2) + h/2)
		wx3 := ((x3 * s.fov / y3) + w/2)
		wy3 := ((z3 * s.fov / y3) + h/2)

		//clipping to screen
		if wx0 < 10 {
			wx0 = 10
		}
		if wx1 < 10 {
			wx1 = 10
		}
		if wx2 < 10 {
			wx2 = 10
		}
		if wx3 < 10 {
			wx3 = 10
		}
		if wy0 < 10 {
			wy0 = 10
		}
		if wy1 < 10 {
			wy1 = 10
		}
		if wy2 < 10 {
			wy2 = 10
		}
		if wy3 < 10 {
			wy3 = 10
		}
		if wx0 > w {
			wx0 = w
		}
		if wx1 > w {
			wx1 = w
		}
		if wx2 > w {
			wx2 = w
		}
		if wx3 > w {
			wx3 = w
		}
		if wy0 > h {
			wy0 = h
		}
		if wy1 > h {
			wy1 = h
		}
		if wy2 > h {
			wy2 = h
		}
		if wy3 > h {
			wy3 = h
		}

		setBufferValue(buffer, 0, wx0, &changed)
		setBufferValue(buffer, 1, wy0, &changed)
		setBufferValue(buffer, 2, tint, &changed)

		setBufferValue(buffer, 3, wx1, &changed)
		setBufferValue(buffer, 4, wy1, &changed)
		setBufferValue(buffer, 5, tint1, &changed)

		setBufferValue(buffer, 6, wx2, &changed)
		setBufferValue(buffer, 7, wy2, &changed)
		setBufferValue(buffer, 8, tint2, &changed)

		setBufferValue(buffer, 9, wx2, &changed)
		setBufferValue(buffer, 10, wy2, &changed)
		setBufferValue(buffer, 11, tint2, &changed)

		setBufferValue(buffer, 12, wx1, &changed)
		setBufferValue(buffer, 13, wy1, &changed)
		setBufferValue(buffer, 14, tint1, &changed)

		setBufferValue(buffer, 15, wx3, &changed)
		setBufferValue(buffer, 16, wy3, &changed)
		setBufferValue(buffer, 17, tint3, &changed)
	default:
		unsupportedType(ren.Drawable)
	}

	return changed
}

func (s *viewShader) PrepareCulling() {}

func (s *viewShader) ShouldDraw(ren *common.RenderComponent, space *common.SpaceComponent) bool {
	if s.player == nil {
		return false
	}
	if s.lastBuffer != ren.Buffer || ren.Buffer == nil {
		s.updateBuffer(ren, space)

		engo.Gl.BindBuffer(engo.Gl.ARRAY_BUFFER, ren.Buffer)
		engo.Gl.VertexAttribPointer(s.inPosition, 2, engo.Gl.FLOAT, false, 12, 0)
		engo.Gl.VertexAttribPointer(s.inColor, 4, engo.Gl.UNSIGNED_BYTE, true, 12, 8)

		s.lastBuffer = ren.Buffer
	}

	return true
}

func (s *viewShader) Draw(ren *common.RenderComponent, space *common.SpaceComponent) {
	s.modelMatrix[0] = ren.Scale.X * engo.GetGlobalScale().X
	s.modelMatrix[1] = 0
	s.modelMatrix[3] = 0
	s.modelMatrix[4] = ren.Scale.Y * engo.GetGlobalScale().Y
	engo.Gl.UniformMatrix3fv(s.matrixModel, false, s.modelMatrix)

	switch ren.Drawable.(type) {
	case Wall:
		engo.Gl.DrawArrays(engo.Gl.TRIANGLES, 0, 6)
	default:
		unsupportedType(ren.Drawable)
	}
}

func (s *viewShader) Post() {
	s.lastBuffer = nil

	// Cleanup
	engo.Gl.DisableVertexAttribArray(s.inPosition)
	engo.Gl.DisableVertexAttribArray(s.inColor)

	engo.Gl.BindBuffer(engo.Gl.ARRAY_BUFFER, nil)
	engo.Gl.BindBuffer(engo.Gl.ELEMENT_ARRAY_BUFFER, nil)

	engo.Gl.Disable(engo.Gl.BLEND)
}

func (s *viewShader) SetCamera(*common.CameraSystem) {}

func (s *viewShader) AddPlayer(space *common.SpaceComponent) {
	s.player = space
}
