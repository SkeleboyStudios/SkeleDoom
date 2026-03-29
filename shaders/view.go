package shaders

import (
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
	inTexCoord int
	inColor    int

	matrixProjection *gl.UniformLocation
	matrixView       *gl.UniformLocation
	matrixModel      *gl.UniformLocation
	texSampler       *gl.UniformLocation
	useTextureLoc    *gl.UniformLocation

	projectionMatrix []float32
	viewMatrix       []float32
	modelMatrix      []float32

	lastBuffer *gl.Buffer

	player       *common.SpaceComponent
	playerOffset engo.Point
	fovAngleDeg  float32
	tanHalfFov   float32
}

func (s *viewShader) Setup(w *ecs.World) error {
	var err error
	s.program, err = common.LoadShader(`
attribute vec2 in_Position;
attribute vec3 in_TexCoord;
attribute vec4 in_Color;

uniform mat3 matrixProjection;
uniform mat3 matrixView;
uniform mat3 matrixModel;

varying vec3 var_TexCoord;
varying vec4 var_Color;

void main() {
  var_TexCoord = in_TexCoord;
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

uniform sampler2D tex0;
uniform float useTexture;

varying vec3 var_TexCoord;
varying vec4 var_Color;

void main (void) {
  if (useTexture > 0.5) {
    vec2 uv = var_TexCoord.xy / var_TexCoord.z;
    vec4 texColor = texture2D(tex0, uv);
    gl_FragColor = texColor * var_Color;
  } else {
    gl_FragColor = var_Color;
  }
}`)

	if err != nil {
		return err
	}

	// Create and populate indices buffer
	s.indicesRectangles = []uint16{0, 1, 2, 0, 2, 3}
	s.indicesRectanglesVBO = engo.Gl.CreateBuffer()
	engo.Gl.BindBuffer(engo.Gl.ELEMENT_ARRAY_BUFFER, s.indicesRectanglesVBO)
	engo.Gl.BufferData(engo.Gl.ELEMENT_ARRAY_BUFFER, s.indicesRectangles, engo.Gl.STATIC_DRAW)

	// Attribute locations
	s.inPosition = engo.Gl.GetAttribLocation(s.program, "in_Position")
	s.inTexCoord = engo.Gl.GetAttribLocation(s.program, "in_TexCoord")
	s.inColor = engo.Gl.GetAttribLocation(s.program, "in_Color")

	// Uniform locations
	s.matrixProjection = engo.Gl.GetUniformLocation(s.program, "matrixProjection")
	s.matrixView = engo.Gl.GetUniformLocation(s.program, "matrixView")
	s.matrixModel = engo.Gl.GetUniformLocation(s.program, "matrixModel")
	s.texSampler = engo.Gl.GetUniformLocation(s.program, "tex0")
	s.useTextureLoc = engo.Gl.GetUniformLocation(s.program, "useTexture")

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

	s.fovAngleDeg = 90
	s.tanHalfFov = math.Tan((s.fovAngleDeg * math.Pi / 180) * 0.5)
	s.playerOffset = engo.Point{X: 49, Y: 242}

	return nil
}

func (s *viewShader) Pre() {
	engo.Gl.Enable(engo.Gl.BLEND)
	engo.Gl.BlendFunc(engo.Gl.SRC_ALPHA, engo.Gl.ONE_MINUS_SRC_ALPHA)

	// Bind shader and buffer, enable attributes
	engo.Gl.UseProgram(s.program)
	engo.Gl.EnableVertexAttribArray(s.inPosition)
	engo.Gl.EnableVertexAttribArray(s.inTexCoord)
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

func (s *viewShader) computeBufferSize(draw common.Drawable) int {
	switch draw.(type) {
	case Wall:
		// 6 vertices × 6 floats (x, y, u/w, v/w, 1/w, color) = 36
		return 36
	default:
		return 0
	}
}

// clipU interpolates the u texture coordinate at the clipping boundary.
// ua is the u at the vertex being clipped away, ub is the u at the vertex
// being kept. fa and fb are the signed distances from each vertex to the
// clip plane (fa < 0 means clipped away).
func clipU(ua, ub, fa, fb float32) float32 {
	t := fa / (fa - fb)
	return ua + (ub-ua)*t
}

func (s *viewShader) generateBufferContent(ren *common.RenderComponent, space *common.SpaceComponent, buffer []float32) bool {
	var changed bool

	tint := colorToFloat32(ren.Color)
	w := engo.GameWidth()
	h := engo.GameHeight()

	switch d := ren.Drawable.(type) {
	case Wall:
		sin, cos := math.Sincos((s.player.Rotation) * math.Pi / 180)
		p1 := d.Line.P1
		p2 := d.Line.P2
		p1X := (p1.X - (s.player.Position.X - s.playerOffset.X))
		p1Y := (-p1.Y + (s.player.Position.Y - s.playerOffset.Y))
		p2X := (p2.X - (s.player.Position.X - s.playerOffset.X))
		p2Y := (-p2.Y + (s.player.Position.Y - s.playerOffset.Y))
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

		const near float32 = 1.0

		// UV coordinates along the wall: u=0 at p1, u=1 at p2
		u0, u1 := float32(0), float32(1)

		// Clip against near plane in camera space
		if y0 < near && y1 < near {
			return false
		} else if y0 < near {
			nearT := y0 / (y0 - y1)
			u0 = u0 + nearT*(u1-u0)
			x0, y0, z0 = clipBehindPlayer(x0, y0, z0, x1, y1, z1)
			x2, y2, z2 = clipBehindPlayer(x2, y2, z2, x3, y3, z3)
		} else if y1 < near {
			nearT := y1 / (y1 - y0)
			u1 = u1 + nearT*(u0-u1)
			x1, y1, z1 = clipBehindPlayer(x1, y1, z1, x0, y0, z0)
			x3, y3, z3 = clipBehindPlayer(x3, y3, z3, x2, y2, z2)
		}

		clipSide := func(xa, ya, za, xb, yb, zb, sign float32) (float32, float32, float32) {
			fa := sign*xa + ya*s.tanHalfFov
			fb := sign*xb + yb*s.tanHalfFov
			t := fa / (fa - fb)
			return xa + (xb-xa)*t, ya + (yb-ya)*t, za + (zb-za)*t
		}

		// Left plane (sign=+1): x + y*tanHalfFov >= 0
		left0 := x0 + y0*s.tanHalfFov
		left1 := x1 + y1*s.tanHalfFov
		if left0 < 0 && left1 < 0 {
			return false
		}
		if left0 < 0 {
			u0 = clipU(u0, u1, left0, left1)
			x0, y0, z0 = clipSide(x0, y0, z0, x1, y1, z1, 1)
			x2, y2, z2 = clipSide(x2, y2, z2, x3, y3, z3, 1)
		} else if left1 < 0 {
			u1 = clipU(u1, u0, left1, left0)
			x1, y1, z1 = clipSide(x1, y1, z1, x0, y0, z0, 1)
			x3, y3, z3 = clipSide(x3, y3, z3, x2, y2, z2, 1)
		}

		// Right plane (sign=-1): y*tanHalfFov - x >= 0
		right0 := y0*s.tanHalfFov - x0
		right1 := y1*s.tanHalfFov - x1
		if right0 < 0 && right1 < 0 {
			return false
		}
		if right0 < 0 {
			u0 = clipU(u0, u1, right0, right1)
			x0, y0, z0 = clipSide(x0, y0, z0, x1, y1, z1, -1)
			x2, y2, z2 = clipSide(x2, y2, z2, x3, y3, z3, -1)
		} else if right1 < 0 {
			u1 = clipU(u1, u0, right1, right0)
			x1, y1, z1 = clipSide(x1, y1, z1, x0, y0, z0, -1)
			x3, y3, z3 = clipSide(x3, y3, z3, x2, y2, z2, -1)
		}

		// Safety clamp to near plane
		if y0 < near {
			y0 = near
			y2 = near
		}
		if y1 < near {
			y1 = near
			y3 = near
		}

		// Convert to screen coordinates
		focalX := (w * 0.5) / s.tanHalfFov
		focalY := focalX

		wx0 := (x0*focalX/y0 + w/2)
		wy0 := (z0*focalY/y0 + h/2)
		wx1 := (x1*focalX/y1 + w/2)
		wy1 := (z1*focalY/y1 + h/2)
		wx2 := (x2*focalX/y2 + w/2)
		wy2 := (z2*focalY/y2 + h/2)
		wx3 := (x3*focalX/y3 + w/2)
		wy3 := (z3*focalY/y3 + h/2)

		// Reject fully off-screen quads
		if (wx0 < 0 && wx1 < 0 && wx2 < 0 && wx3 < 0) ||
			(wx0 > w && wx1 > w && wx2 > w && wx3 > w) ||
			(wy0 < 0 && wy1 < 0 && wy2 < 0 && wy3 < 0) ||
			(wy0 > h && wy1 > h && wy2 > h && wy3 > h) {
			return false
		}

		// Perspective-correct UV components: store u/w, v/w, 1/w
		// w (depth) at p1 side = y0, at p2 side = y1
		ow0 := float32(1) / y0 // 1/w at p1
		ow1 := float32(1) / y1 // 1/w at p2

		// Buffer layout per vertex: [x, y, u/w, v/w, 1/w, color] (6 floats, 24 bytes stride)
		// Triangle 1: v0(bottom-left p1), v1(bottom-right p2), v2(top-left p1)
		// Triangle 2: v3(top-left p1),    v4(bottom-right p2), v5(top-right p2)

		// v0: bottom-left (p1, v=1 bottom)
		setBufferValue(buffer, 0, wx0, &changed)
		setBufferValue(buffer, 1, wy0, &changed)
		setBufferValue(buffer, 2, u0*ow0, &changed)
		setBufferValue(buffer, 3, 1*ow0, &changed) // v=1 (bottom of texture)
		setBufferValue(buffer, 4, ow0, &changed)
		setBufferValue(buffer, 5, tint, &changed)

		// v1: bottom-right (p2, v=1 bottom)
		setBufferValue(buffer, 6, wx1, &changed)
		setBufferValue(buffer, 7, wy1, &changed)
		setBufferValue(buffer, 8, u1*ow1, &changed)
		setBufferValue(buffer, 9, 1*ow1, &changed) // v=1 (bottom of texture)
		setBufferValue(buffer, 10, ow1, &changed)
		setBufferValue(buffer, 11, tint, &changed)

		// v2: top-left (p1, v=0 top)
		setBufferValue(buffer, 12, wx2, &changed)
		setBufferValue(buffer, 13, wy2, &changed)
		setBufferValue(buffer, 14, u0*ow0, &changed)
		setBufferValue(buffer, 15, 0, &changed) // v=0 (top of texture), 0/w = 0
		setBufferValue(buffer, 16, ow0, &changed)
		setBufferValue(buffer, 17, tint, &changed)

		// v3: top-left again
		setBufferValue(buffer, 18, wx2, &changed)
		setBufferValue(buffer, 19, wy2, &changed)
		setBufferValue(buffer, 20, u0*ow0, &changed)
		setBufferValue(buffer, 21, 0, &changed)
		setBufferValue(buffer, 22, ow0, &changed)
		setBufferValue(buffer, 23, tint, &changed)

		// v4: bottom-right again
		setBufferValue(buffer, 24, wx1, &changed)
		setBufferValue(buffer, 25, wy1, &changed)
		setBufferValue(buffer, 26, u1*ow1, &changed)
		setBufferValue(buffer, 27, 1*ow1, &changed)
		setBufferValue(buffer, 28, ow1, &changed)
		setBufferValue(buffer, 29, tint, &changed)

		// v5: top-right (p2, v=0 top)
		setBufferValue(buffer, 30, wx3, &changed)
		setBufferValue(buffer, 31, wy3, &changed)
		setBufferValue(buffer, 32, u1*ow1, &changed)
		setBufferValue(buffer, 33, 0, &changed)
		setBufferValue(buffer, 34, ow1, &changed)
		setBufferValue(buffer, 35, tint, &changed)

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

		// Stride is now 24 bytes (6 floats × 4 bytes):
		//   offset  0: position (2 × float32)
		//   offset  8: texcoord (3 × float32)  ← u/w, v/w, 1/w
		//   offset 20: color    (4 × uint8)
		engo.Gl.VertexAttribPointer(s.inPosition, 2, engo.Gl.FLOAT, false, 24, 0)
		engo.Gl.VertexAttribPointer(s.inTexCoord, 3, engo.Gl.FLOAT, false, 24, 8)
		engo.Gl.VertexAttribPointer(s.inColor, 4, engo.Gl.UNSIGNED_BYTE, true, 24, 20)

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

	switch d := ren.Drawable.(type) {
	case Wall:
		if d.Tex != nil {
			engo.Gl.ActiveTexture(engo.Gl.TEXTURE0)
			engo.Gl.BindTexture(engo.Gl.TEXTURE_2D, d.Tex)
			engo.Gl.Uniform1i(s.texSampler, 0)
			engo.Gl.Uniform1f(s.useTextureLoc, 1.0)
		} else {
			engo.Gl.Uniform1f(s.useTextureLoc, 0.0)
		}
		engo.Gl.DrawArrays(engo.Gl.TRIANGLES, 0, 6)
	default:
		unsupportedType(ren.Drawable)
	}
}

func (s *viewShader) Post() {
	s.lastBuffer = nil

	engo.Gl.DisableVertexAttribArray(s.inPosition)
	engo.Gl.DisableVertexAttribArray(s.inTexCoord)
	engo.Gl.DisableVertexAttribArray(s.inColor)

	engo.Gl.BindBuffer(engo.Gl.ARRAY_BUFFER, nil)
	engo.Gl.BindBuffer(engo.Gl.ELEMENT_ARRAY_BUFFER, nil)

	engo.Gl.Disable(engo.Gl.BLEND)
}

func (s *viewShader) SetCamera(*common.CameraSystem) {}

func (s *viewShader) AddPlayer(space *common.SpaceComponent) {
	s.player = space
}
