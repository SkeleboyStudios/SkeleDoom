package systems

import (
	"image/color"

	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"
	"github.com/SkeleboyStudios/SkeleDoom/shaders"
)

type NotMapComponent struct{}

func (c *NotMapComponent) GetNotMapComponent() *NotMapComponent { return c }

type NotMapFace interface {
	GetNotMapComponent() *NotMapComponent
}

type NotMapAble interface {
	NotMapFace
}

type PlayerMapComponent struct{}

func (c *PlayerMapComponent) GetPlayerMapComponent() *PlayerMapComponent {
	return c
}

type PlayerMapFace interface {
	GetPlayerMapComponent() *PlayerMapComponent
}

type PlayerMapAble interface {
	common.BasicFace

	common.SpaceFace
	PlayerMapFace
}

type mapPlayerEntity struct {
	*ecs.BasicEntity

	*common.RenderComponent
	*common.SpaceComponent
	*common.CollisionComponent
	*PlayerMapComponent
	*NotMapComponent
}

type WallMapComponent struct {
	Wall engo.Line
}

func (c *WallMapComponent) GetWallMapComponent() *WallMapComponent {
	return c
}

type WallMapFace interface {
	GetWallMapComponent() *WallMapComponent
}

type WallMapAble interface {
	common.BasicFace

	WallMapFace
}

type mapWallEntity struct {
	*ecs.BasicEntity

	*common.RenderComponent
	*common.SpaceComponent
	*common.CollisionComponent
	*WallMapComponent
	*NotMapComponent
}

type sprite struct {
	ecs.BasicEntity

	common.RenderComponent
	common.SpaceComponent
}

type MapSystem struct {
	w *ecs.World

	player      mapPlayerEntity
	boundingbox sprite
	walls       []mapWallEntity
}

func (s *MapSystem) New(w *ecs.World) {
	s.w = w

	s.boundingbox = sprite{BasicEntity: ecs.NewBasic()}
	s.boundingbox.SpaceComponent = common.SpaceComponent{
		Position: engo.Point{X: 10, Y: 213},
	}
	borderTex, _ := common.LoadedSprite("ui/statsborder.png")
	s.boundingbox.RenderComponent = common.RenderComponent{
		Drawable: borderTex,
	}
	s.boundingbox.SetShader(common.HUDShader)
	s.boundingbox.SetZIndex(3)
	w.AddEntity(&s.boundingbox)
}

func (s *MapSystem) AddByInterface(i ecs.Identifier) {
	if o, ok := i.(PlayerMapAble); ok {
		s.player.BasicEntity = o.GetBasicEntity()
		s.player.SpaceComponent = o.GetSpaceComponent()
		s.player.Position.X += s.boundingbox.Position.X + 121.5
		s.player.Position.Y += s.boundingbox.Position.Y + 75
		s.player.Width = 5
		s.player.Height = 10
		s.player.RenderComponent = &common.RenderComponent{
			Drawable:    common.Triangle{},
			Color:       color.RGBA{0xFF, 0x00, 0x00, 0xFF},
			StartZIndex: 5,
		}
		s.player.SetShader(shaders.MapShader)
		s.player.CollisionComponent = &common.CollisionComponent{Main: CollisionGroupPlaya}
		s.w.AddEntity(&s.player)
	}
	if o, ok := i.(WallMapAble); ok {
		wa := mapWallEntity{BasicEntity: o.GetBasicEntity()}
		wall := o.GetWallMapComponent().Wall
		wall.P1.X += s.boundingbox.Position.X + 29
		wall.P2.X += s.boundingbox.Position.X + 29
		wall.P1.Y += s.boundingbox.Position.Y + 40
		wall.P2.Y += s.boundingbox.Position.Y + 40
		wa.SpaceComponent = &common.SpaceComponent{
			Position: wall.P1,
			Width:    5,
			Height:   wall.Magnitude(),
			Rotation: 180 + wall.AngleDeg(),
		}
		lines := []engo.Line{}
		lines = append(lines, engo.Line{
			P1: engo.Point{X: 0, Y: 0},
			P2: engo.Point{X: 5, Y: 0},
		}, engo.Line{
			P1: engo.Point{X: 5, Y: 0},
			P2: engo.Point{X: 5, Y: wall.Magnitude()},
		}, engo.Line{
			P1: engo.Point{X: 5, Y: wall.Magnitude()},
			P2: engo.Point{X: 0, Y: wall.Magnitude()},
		}, engo.Line{
			P1: engo.Point{X: 0, Y: wall.Magnitude()},
			P2: engo.Point{X: 0, Y: 0},
		})
		wa.AddShape(common.Shape{Lines: lines})
		wa.RenderComponent = &common.RenderComponent{
			Drawable:    common.Rectangle{},
			Color:       color.RGBA{0x54, 0xCD, 0xF0, 0xFF},
			StartZIndex: 5,
		}
		wa.SetShader(shaders.MapShader)
		wa.CollisionComponent = &common.CollisionComponent{Group: CollisionGroupPlaya}
		s.w.AddEntity(&wa)
		s.walls = append(s.walls, wa)
	}
}

func (s *MapSystem) Remove(basic ecs.BasicEntity) {}

func (s *MapSystem) Update(dt float32) {
	width, height := s.player.SpaceComponent.Width, s.player.SpaceComponent.Height

	pos := s.player.SpaceComponent.Position
	trackToX := pos.X + width/2 + 188.5
	trackToY := pos.Y + height/2 - 108

	engo.Mailbox.Dispatch(common.CameraMessage{Axis: common.XAxis, Value: trackToX, Incremental: false})
	engo.Mailbox.Dispatch(common.CameraMessage{Axis: common.YAxis, Value: trackToY, Incremental: false})
}
