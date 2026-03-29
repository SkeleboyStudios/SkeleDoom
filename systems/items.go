package systems

import (
	"image/color"

	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"
	"github.com/EngoEngine/engo/math"
	"github.com/EngoEngine/gl"
	"github.com/SkeleboyStudios/SkeleDoom/shaders"
)

// ItemEffect is called when the player picks up an item. It receives no
// arguments; the caller closes over whatever state it needs to modify.
type ItemEffect func()

// ItemComponent holds all data needed to render and interact with a pickupable
// item. Embed it in your entity struct and implement ItemAble.
type ItemComponent struct {
	// Tex is the texture shown on the 3D billboard. Nil renders a solid colour.
	Tex *gl.Texture
	// W and H are the billboard's world-unit dimensions.
	W, H float32
	// Effect is called once when the player enters pickup radius. May be nil.
	Effect ItemEffect
	// Radius is the pickup detection distance in world units.
	Radius float32
}

func (c *ItemComponent) GetItemComponent() *ItemComponent { return c }

// ItemFace is the minimal interface for the ItemComponent accessor.
type ItemFace interface {
	GetItemComponent() *ItemComponent
}

// ItemAble is implemented by any entity that can be managed by ItemSystem.
type ItemAble interface {
	common.BasicFace
	common.SpaceFace
	ItemFace
}

// itemEntity is the system's internal representation of one item. It owns two
// sub-entities: a 3D billboard registered with ViewShader, and a minimap dot
// registered with MinimapShader.
type itemEntity struct {
	*ecs.BasicEntity
	*common.SpaceComponent
	*ItemComponent

	// billboard is the 3D view entity; uses shaders.ViewShader.
	billboard struct {
		ecs.BasicEntity
		common.RenderComponent
		common.SpaceComponent
	}

	// mapDot is the minimap square; uses shaders.MinimapShader.
	mapDot struct {
		ecs.BasicEntity
		common.RenderComponent
		common.SpaceComponent
	}

	pickedUp bool
}

// ItemSystem manages pickupable items. It must receive both ViewPlayerAble
// entities (to track the player) and ItemAble entities (to manage items).
//
// Register it in the scene with something like:
//
//	var playerviewable *systems.ViewPlayerAble
//	var itemable       *systems.ItemAble
//	w.AddSystemInterface(&systems.ItemSystem{}, []any{playerviewable, itemable}, nil)
type ItemSystem struct {
	w      *ecs.World
	player *common.SpaceComponent
	items  []*itemEntity
}

func (s *ItemSystem) New(w *ecs.World) {
	s.w = w
}

func (s *ItemSystem) AddByInterface(i ecs.Identifier) {
	// Accept the player entity so we have its position/rotation every frame.
	if o, ok := i.(ViewPlayerAble); ok {
		s.player = o.GetSpaceComponent()
		return
	}

	// Accept item entities.
	o, ok := i.(ItemAble)
	if !ok {
		return
	}

	ic := o.GetItemComponent()
	sp := o.GetSpaceComponent()

	item := &itemEntity{
		BasicEntity:    o.GetBasicEntity(),
		SpaceComponent: sp,
		ItemComponent:  ic,
	}

	// ── 3D billboard ─────────────────────────────────────────────────────
	item.billboard.BasicEntity = ecs.NewBasic()
	item.billboard.SpaceComponent = common.SpaceComponent{
		Position: sp.Position,
		Width:    ic.W,
		Height:   ic.H,
	}
	item.billboard.RenderComponent = common.RenderComponent{
		Drawable: shaders.Billboard{
			Pos: sp.Position,
			W:   ic.W,
			H:   ic.H,
			Tex: ic.Tex,
		},
		Color: color.RGBA{0xff, 0xff, 0xff, 0xff},
	}
	if ic.Tex == nil {
		// No texture: render as a solid yellow quad so it's still visible.
		item.billboard.RenderComponent.Color = color.RGBA{0xff, 0xee, 0x00, 0xff}
	}
	item.billboard.SetShader(shaders.ViewShader)
	s.w.AddEntity(&item.billboard)

	// ── Minimap dot ───────────────────────────────────────────────────────
	const dotSize float32 = 6
	item.mapDot.BasicEntity = ecs.NewBasic()
	item.mapDot.SpaceComponent = common.SpaceComponent{
		Position: engo.Point{
			X: sp.Position.X + MapWallOffsetX - dotSize/2,
			Y: sp.Position.Y + MapWallOffsetY - dotSize/2,
		},
		Width:  dotSize,
		Height: dotSize,
	}
	item.mapDot.RenderComponent = common.RenderComponent{
		Drawable:    common.Rectangle{},
		Color:       color.RGBA{0x00, 0xff, 0x00, 0xff}, // bright green
		StartZIndex: 4,
	}
	item.mapDot.SetShader(shaders.MinimapShader)
	s.w.AddEntity(&item.mapDot)

	s.items = append(s.items, item)
}

func (s *ItemSystem) Remove(basic ecs.BasicEntity) {
	for i, item := range s.items {
		if item.BasicEntity.ID() == basic.ID() {
			s.items = append(s.items[:i], s.items[i+1:]...)
			return
		}
	}
}

func (s *ItemSystem) Update(dt float32) {
	if s.player == nil {
		return
	}

	const near float32 = 1.0

	po := shaders.PlayerOffset
	// Effective player position in wall-space (same coordinate frame as item
	// positions and wall endpoints).
	playerX := s.player.Position.X - po.X
	playerY := s.player.Position.Y - po.Y

	sin, cos := math.Sincos(s.player.Rotation * math.Pi / 180)

	for _, item := range s.items {
		if item.pickedUp {
			continue
		}

		// ── Proximity pickup ─────────────────────────────────────────────
		dx := playerX - item.SpaceComponent.Position.X
		dy := playerY - item.SpaceComponent.Position.Y
		if math.Sqrt(dx*dx+dy*dy) <= item.Radius {
			item.pickedUp = true
			item.billboard.Hidden = true
			item.mapDot.Hidden = true
			if item.Effect != nil {
				item.Effect()
			}
			continue
		}

		// ── Depth z-sorting for the billboard ────────────────────────────
		// Transform item into camera space to get depth, then set z-index so
		// items sort correctly relative to walls (which use the same scheme).
		relX := item.SpaceComponent.Position.X - playerX
		relY := -item.SpaceComponent.Position.Y + playerY
		camY := relY*cos + relX*sin // camera-space depth

		if camY < near {
			// Behind the player; hide the billboard (shader would cull it
			// anyway, but setting Hidden avoids a wasted ShouldDraw call).
			item.billboard.Hidden = true
			continue
		}

		item.billboard.Hidden = false

		// Clamp to near so depth is never negative (same fix as ViewSystem).
		dy2 := camY
		if dy2 < near {
			dy2 = near
		}
		// Offset of 50 matches ViewSystem so items and walls interleave correctly.
		item.billboard.SetZIndex(-(dy2 + 50))
	}
}
