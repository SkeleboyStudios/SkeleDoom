package systems

import (
	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"
)

type sprite struct {
	ecs.BasicEntity

	common.RenderComponent
	common.SpaceComponent
}

type wall struct {
	texture, icon sprite
	repeat        common.TextureRepeating

	lines []engo.Line
}

type player struct {
	ecs.BasicEntity

	common.SpaceComponent
	PlayerMapComponent
	ControlComponent
}
