package systems

import (
	"image/color"

	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"

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
	// increment := 80.0 / float32(s.numLines)
	// cur := float32(-40.0)
	//
	//	for _, e := range s.walls {
	//		e.wall.Hidden = true
	//	}
	//
	//	for i := 0; i < s.numLines; i++ {
	//		l := engo.Line{P1: s.player.SpaceComponent.Position}
	//		sin, cos := math.Sincos((s.player.SpaceComponent.Rotation + cur) * math.Pi / 180)
	//		l.P2.X = s.player.SpaceComponent.Position.X + sin*s.lineLength
	//		l.P2.Y = s.player.SpaceComponent.Position.Y - cos*s.lineLength
	//		for _, e := range s.walls {
	//			if _, ok := engo.LineIntersection(l, e.Wall); ok {
	//				e.wall.Hidden = false
	//				//e.wall.SetZIndex(p.PointDistance(l.P1))
	//			}
	//		}
	//		cur += increment
	//	}
}
