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
}

type player struct {
	ecs.BasicEntity

	common.SpaceComponent
	systems.PlayerMapComponent
	systems.ControlComponent
}
