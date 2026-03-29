package systems

import (
	"image/color"

	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"
	"github.com/EngoEngine/engo/math"

	"github.com/SkeleboyStudios/SkeleDoom/shaders"
)

type ViewPlayerComponent struct{}

func (c *ViewPlayerComponent) GetViewPlayerComponent() *ViewPlayerComponent { return c }

type NotViewComponent struct{}

func (n *NotViewComponent) GetNotViewComponent() *NotViewComponent { return n }

type NotViewFace interface {
	GetNotViewComponent() *NotViewComponent
}

type NotViewAble interface {
	NotViewFace
}

type ViewPlayerFace interface {
	GetViewPlayerComponent() *ViewPlayerComponent
}

type ViewPlayerAble interface {
	common.BasicFace
	common.SpaceFace

	ViewPlayerFace
}

type viewPlayerEntity struct {
	*ecs.BasicEntity

	hands struct {
		ecs.BasicEntity
		common.RenderComponent
		common.SpaceComponent
		//common.AnimationComponent
	}

	*common.SpaceComponent

	*ViewPlayerComponent
	*NotViewComponent
}

type ViewWallComponent struct{}

func (c *ViewWallComponent) GetViewWallComponent() *ViewWallComponent { return c }

type ViewWallFace interface {
	GetViewWallComponent() *ViewWallComponent
}

type ViewWallAble interface {
	common.BasicFace
	common.SpaceFace

	ViewWallFace
	WallMapFace
}

type viewWallEntity struct {
	*ecs.BasicEntity

	wall struct {
		ecs.BasicEntity
		*common.RenderComponent
		common.SpaceComponent
	}

	*ViewWallComponent
	*WallMapComponent
	*NotMapComponent
	*NotViewComponent
}

type ViewSystem struct {
	w          *ecs.World
	player     viewPlayerEntity
	walls      []viewWallEntity
	numLines   int
	lineLength float32
}

func (s *ViewSystem) New(w *ecs.World) {
	s.w = w
	s.numLines = 60
	s.lineLength = 1000
}

func (s *ViewSystem) AddByInterface(i ecs.Identifier) {
	if o, ok := i.(ViewPlayerAble); ok {
		s.player.BasicEntity = o.GetBasicEntity()
		s.player.ViewPlayerComponent = o.GetViewPlayerComponent()
		tex, _ := common.LoadedSprite("ui/hands.png")
		s.player.hands.BasicEntity = ecs.NewBasic()
		s.player.hands.RenderComponent = common.RenderComponent{Drawable: tex}
		s.player.hands.Scale = engo.Point{X: 2, Y: 2}
		s.player.hands.SetShader(common.HUDShader)
		s.player.hands.Hidden = true
		s.player.SpaceComponent = o.GetSpaceComponent()
		shaders.ViewShader.AddPlayer(o.GetSpaceComponent())
		s.w.AddEntity(&s.player.hands)
	}
	if o, ok := i.(ViewWallAble); ok {
		wa := o.GetWallMapComponent().Wall
		wall := viewWallEntity{BasicEntity: o.GetBasicEntity()}
		wall.wall.BasicEntity = ecs.NewBasic()
		wall.wall.SpaceComponent = common.SpaceComponent{Position: wa.P1, Width: wa.Magnitude(), Height: 60}
		wall.wall.RenderComponent = &common.RenderComponent{Drawable: shaders.Wall{Line: wa, H: 60}, Color: color.RGBA{0x00, 0x00, 0xff, 0xff}}
		wall.wall.SetShader(shaders.ViewShader)
		wall.WallMapComponent = o.GetWallMapComponent()
		s.w.AddEntity(&wall.wall)
		s.walls = append(s.walls, wall)
	}
}

func (s *ViewSystem) Remove(basic ecs.BasicEntity) {}

func (s *ViewSystem) Update(dt float32) {
	if s.player.SpaceComponent == nil {
		return
	}

	const near float32 = 1.0
	const fovAngleDeg float32 = 90.0
	tanHalfFov := math.Tan((fovAngleDeg * math.Pi / 180) * 0.5)

	playerPos := s.player.SpaceComponent.Position
	playerRot := s.player.SpaceComponent.Rotation
	playerOffset := engo.Point{X: 49, Y: 242}

	sin, cos := math.Sincos(playerRot * math.Pi / 180)

	for i := range s.walls {
		e := &s.walls[i]
		wa := e.WallMapComponent.Wall

		// Translate wall endpoints into player-relative coordinates
		p1X := wa.P1.X - (playerPos.X - playerOffset.X)
		p1Y := -wa.P1.Y + (playerPos.Y - playerOffset.Y)
		p2X := wa.P2.X - (playerPos.X - playerOffset.X)
		p2Y := -wa.P2.Y + (playerPos.Y - playerOffset.Y)

		// Rotate into camera space (y = depth, x = right-handed: positive x = screen right)
		x0 := p1Y*sin - p1X*cos
		y0 := p1Y*cos + p1X*sin
		x1 := p2Y*sin - p2X*cos
		y1 := p2Y*cos + p2X*sin

		// Hide if fully behind near plane
		if y0 < near && y1 < near {
			e.wall.Hidden = true
			continue
		}

		// Frustum-side visibility clipping test in camera space:
		// visible region satisfies -y*tanHalfFov <= x <= y*tanHalfFov for y > 0
		left0 := x0 + y0*tanHalfFov
		left1 := x1 + y1*tanHalfFov
		right0 := y0*tanHalfFov - x0
		right1 := y1*tanHalfFov - x1

		if (left0 < 0 && left1 < 0) || (right0 < 0 && right1 < 0) {
			e.wall.Hidden = true
			continue
		}

		e.wall.Hidden = false

		// Painter-style ordering: farther walls first, nearer walls last
		depth := (y0 + y1) * 0.5
		e.wall.SetZIndex(-depth)
	}
}
