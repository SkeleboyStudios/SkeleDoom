package scenes

import (
	"image/color"

	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"
	"github.com/SkeleboyStudios/SkeleDoom/shaders"
	"github.com/SkeleboyStudios/SkeleDoom/systems"
)

const StartSceneTypeString = "Start Scene"

type StartScene struct{}

func (s *StartScene) Type() string { return StartSceneTypeString }

func (s *StartScene) Preload() {
	common.AddShader(shaders.MapShader)
	engo.Input.RegisterButton("up", engo.KeyW, engo.KeyArrowUp)
	engo.Input.RegisterButton("down", engo.KeyS, engo.KeyArrowDown)
	engo.Input.RegisterButton("left", engo.KeyA, engo.KeyArrowLeft)
	engo.Input.RegisterButton("right", engo.KeyD, engo.KeyArrowRight)
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

	var playermapable *systems.PlayerMapAble
	var wallmapable *systems.WallMapAble
	var notmapable *systems.NotMapAble
	w.AddSystemInterface(&systems.MapSystem{}, []interface{}{playermapable, wallmapable}, notmapable)

	var controlable *systems.ControlAble
	w.AddSystemInterface(&systems.ControlSystem{}, controlable, nil)

	p := player{BasicEntity: ecs.NewBasic()}
	p.Speed = 150
	p.RotSpeed = 25
	w.AddEntity(&p)

	wall1 := wall{BasicEntity: ecs.NewBasic()}
	wall1.Wall = engo.Line{
		P1: engo.Point{X: -25, Y: 0},
		P2: engo.Point{X: 100, Y: 0},
	}
	w.AddEntity(&wall1)

	wall2 := wall{BasicEntity: ecs.NewBasic()}
	wall2.Wall = engo.Line{
		P1: engo.Point{X: 15, Y: 15},
		P2: engo.Point{X: 200, Y: 250},
	}
	w.AddEntity(&wall2)

	wall3 := wall{BasicEntity: ecs.NewBasic()}
	wall3.Wall = engo.Line{
		P1: engo.Point{X: 150, Y: 50},
		P2: engo.Point{X: 250, Y: -25},
	}
	w.AddEntity(&wall3)

	wall4 := wall{BasicEntity: ecs.NewBasic()}
	wall4.Wall = engo.Line{
		P1: engo.Point{X: 150, Y: 50},
		P2: engo.Point{X: 150, Y: -25},
	}
	w.AddEntity(&wall4)
}
