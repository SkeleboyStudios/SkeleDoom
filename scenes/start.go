package scenes

import (
	"image/color"

	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"
	"github.com/EngoEngine/gl"
	"github.com/SkeleboyStudios/SkeleDoom/shaders"
	"github.com/SkeleboyStudios/SkeleDoom/systems"
)

const StartSceneTypeString = "Start Scene"

type StartScene struct{}

func (s *StartScene) Type() string { return StartSceneTypeString }

func (s *StartScene) Preload() {
	engo.Files.Load("ui/statsborder.png")
	engo.Files.Load("ui/bomb.png")
	engo.Files.Load("ui/hands.png")
	common.AddShader(shaders.ViewShader)
	common.AddShader(shaders.MinimapShader)
	engo.Input.RegisterButton("up", engo.KeyW, engo.KeyArrowUp)
	engo.Input.RegisterButton("down", engo.KeyS, engo.KeyArrowDown)
	engo.Input.RegisterButton("left", engo.KeyA, engo.KeyArrowLeft)
	engo.Input.RegisterButton("right", engo.KeyD, engo.KeyArrowRight)
	engo.Input.RegisterButton("sprint", engo.KeyLeftShift, engo.KeyRightShift)
	engo.Input.RegisterButton("crouch", engo.KeyLeftControl, engo.KeyRightControl)
	engo.Input.RegisterButton("jump", engo.KeySpace)
	engo.Input.RegisterAxis("hori", engo.NewAxisMouse(engo.AxisMouseHori))
}

func (s *StartScene) Setup(u engo.Updater) {
	w, _ := u.(*ecs.World)

	common.SetBackground(color.RGBA{0x55, 0x55, 0x55, 0xFF})

	var renderable *common.Renderable
	var notrenderable *common.NotRenderable
	w.AddSystemInterface(&common.RenderSystem{}, renderable, notrenderable)

	var animatable *common.Animationable
	var notanimatable *common.NotAnimationable
	w.AddSystemInterface(&common.AnimationSystem{}, animatable, notanimatable)

	var collisionable *common.Collisionable
	var notcollisionable *common.NotCollisionable
	w.AddSystemInterface(&common.CollisionSystem{Solids: systems.CollisionGroupPlaya | systems.CollisionGroupWall}, collisionable, notcollisionable)

	var playermapable *systems.PlayerMapAble
	var wallmapable *systems.WallMapAble
	var notmapable *systems.NotMapAble
	w.AddSystemInterface(&systems.MapSystem{}, []any{playermapable, wallmapable}, notmapable)

	var playerviewable *systems.ViewPlayerAble
	var wallviewable *systems.ViewWallAble
	var notviewable *systems.NotViewAble
	w.AddSystemInterface(&systems.ViewSystem{}, []any{playerviewable, wallviewable}, notviewable)

	var playeritemable *systems.ViewPlayerAble
	var itemable *systems.ItemAble
	w.AddSystemInterface(&systems.ItemSystem{}, []any{playeritemable, itemable}, nil)

	var controlable *systems.ControlAble
	w.AddSystemInterface(&systems.ControlSystem{}, controlable, nil)

	var lavaplayerable *systems.LavaPlayerAble
	var lavazonable *systems.LavaZoneAble
	w.AddSystemInterface(&systems.LavaSystem{}, []any{lavaplayerable, lavazonable}, nil)

	p := player{BasicEntity: ecs.NewBasic()}
	p.Speed = 150
	p.RotSpeed = 25
	p.Height = 20
	w.AddEntity(&p)

	// Generate a single brick texture shared by all walls.
	// CreateBrickTexture must be called after the GL context is ready (i.e. here in Setup).
	brickTex := shaders.CreateBrickTexture(128, 128)

	addWall := func(p1, p2 engo.Point, tex *gl.Texture) {
		e := wall{BasicEntity: ecs.NewBasic()}
		e.Wall = engo.Line{P1: p1, P2: p2}
		e.Tex = tex
		w.AddEntity(&e)
	}

	addWall(engo.Point{X: -25, Y: 0}, engo.Point{X: 100, Y: 0}, brickTex)
	addWall(engo.Point{X: 15, Y: 15}, engo.Point{X: 200, Y: 250}, brickTex)
	addWall(engo.Point{X: 150, Y: 50}, engo.Point{X: 250, Y: -25}, brickTex)
	addWall(engo.Point{X: 150, Y: 50}, engo.Point{X: 150, Y: -25}, brickTex)

	// addZone places a lava damage zone.  X/Y/W/H are in wall world-space
	// (same coordinate system as wall endpoints above).
	addZone := func(x, y, zw, zh float32, c color.RGBA, dps float32) {
		e := lavaZone{BasicEntity: ecs.NewBasic()}
		e.SpaceComponent.Position.X = x
		e.SpaceComponent.Position.Y = y
		e.SpaceComponent.Width = zw
		e.SpaceComponent.Height = zh
		e.LavaZoneComponent.Color = c
		e.LavaZoneComponent.DPS = dps
		w.AddEntity(&e)
	}

	// Orange lava pool along the first wall (low DPS — just a hazard to dodge).
	addZone(0, 0, 75, 30,
		color.RGBA{0xFF, 0x66, 0x00, 0xCC}, 8)

	// Deep-red lava pit near the corner walls (higher DPS — punishing).
	addZone(155, -20, 45, 45,
		color.RGBA{0xCC, 0x11, 0x00, 0xCC}, 20)
	// Generate the potion texture (must be after GL context is ready).
	potionTex := shaders.CreatePotionTexture(64)

	// addItem places a pickupable potion at the given wall-space position.
	addItem := func(pos engo.Point, effect systems.ItemEffect) {
		e := item{BasicEntity: ecs.NewBasic()}
		e.Position = pos
		e.Tex = potionTex
		e.W = 20
		e.H = 30
		e.Radius = 20
		e.Effect = effect
		w.AddEntity(&e)
	}

	// Demo potions – replace the effects with whatever gameplay logic you need.
	addItem(engo.Point{X: 40, Y: 30}, func() {
		p.Speed += 50 // speed boost
	})
	addItem(engo.Point{X: 120, Y: -10}, func() {
		p.RotSpeed += 10 // turn-speed boost
	})
}
