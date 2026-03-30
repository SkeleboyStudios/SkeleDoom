package scenes

import (
	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo/common"
	"github.com/SkeleboyStudios/SkeleDoom/systems"
)

type sprite struct {
	ecs.BasicEntity

	common.RenderComponent
	common.SpaceComponent
}

type wall struct {
	ecs.BasicEntity

	common.SpaceComponent
	systems.WallMapComponent
	systems.ViewWallComponent
}

type player struct {
	ecs.BasicEntity

	common.SpaceComponent
	systems.PlayerMapComponent
	systems.ControlComponent
	systems.ViewPlayerComponent
}

type lavaZone struct {
	ecs.BasicEntity

	common.SpaceComponent
	common.CollisionComponent
	systems.LavaZoneComponent
}
// item is a pickupable world object. It is excluded from the MapSystem and
// ViewSystem (which handle walls) via NotMapComponent and NotViewComponent;
// the ItemSystem creates and owns the 3D billboard and minimap dot instead.
type item struct {
	ecs.BasicEntity

	common.SpaceComponent
	systems.ItemComponent
	systems.NotMapComponent
	systems.NotViewComponent
}

// Tex, W, H, Effect, and Radius on item are promoted from ItemComponent and
// can be set directly:
//
//	e := item{BasicEntity: ecs.NewBasic()}
//	e.Position = engo.Point{X: 50, Y: 20}
//	e.Tex    = myTex
//	e.W, e.H = 20, 30
//	e.Radius = 20
//	e.Effect = func() { /* ... */ }
