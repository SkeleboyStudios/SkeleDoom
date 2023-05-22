package systems

import (
	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"
	"github.com/EngoEngine/engo/math"
)

type ControlComponent struct {
	Speed    float32
	RotSpeed float32

	velocity engo.Point
}

func (c *ControlComponent) GetControlComponent() *ControlComponent { return c }

type ControlFace interface {
	GetControlComponent() *ControlComponent
}

type ControlAble interface {
	common.BasicFace

	ControlFace
	common.SpaceFace
}

type controlEntity struct {
	*ecs.BasicEntity

	*ControlComponent
	*common.SpaceComponent
}

type ControlSystem struct {
	entities []controlEntity
}

func (s *ControlSystem) New(w *ecs.World) {}

func (s *ControlSystem) Add(basic *ecs.BasicEntity, control *ControlComponent, space *common.SpaceComponent) {
	s.entities = append(s.entities, controlEntity{basic, control, space})
}

func (s *ControlSystem) AddByInterface(i ecs.Identifier) {
	o, ok := i.(ControlAble)
	if !ok {
		return
	}
	s.Add(o.GetBasicEntity(), o.GetControlComponent(), o.GetSpaceComponent())
}

func (s *ControlSystem) Remove(basic ecs.BasicEntity) {
	d := -1
	for i, e := range s.entities {
		if e.BasicEntity.ID() == basic.ID() {
			d = i
			break
		}
	}
	if d >= 0 {
		s.entities = append(s.entities[:d], s.entities[d+1:]...)
	}
}

func (s *ControlSystem) Update(dt float32) {
	for i, entity := range s.entities {
		if v, changed := s.getSpeed(); changed {
			v, _ = v.Normalize()
			v.MultiplyScalar(dt * entity.Speed)
			s.entities[i].velocity = v
		}
		ctr := entity.GetSpaceComponent().Center()
		entity.Rotation += engo.Input.Axis("hori").Value() * entity.RotSpeed * dt
		entity.SetCenter(ctr)
		sin, cos := math.Sincos(entity.Rotation * math.Pi / 180)
		entpt := engo.Point{
			X: entity.velocity.X*cos - entity.velocity.Y*sin,
			Y: entity.velocity.Y*cos + entity.velocity.X*sin,
		}
		entity.Position.Add(entpt)
	}
}

func (s *ControlSystem) getSpeed() (p engo.Point, changed bool) {
	if engo.Input.Button("up").JustPressed() {
		p.Y = -1
	} else if engo.Input.Button("down").JustPressed() {
		p.Y = 1
	}
	if engo.Input.Button("left").JustPressed() {
		p.X = -1
	} else if engo.Input.Button("right").JustPressed() {
		p.X = 1
	}

	if engo.Input.Button("up").JustReleased() || engo.Input.Button("down").JustReleased() {
		p.Y = 0
		changed = true
		if engo.Input.Button("up").Down() {
			p.Y = -1
		} else if engo.Input.Button("down").Down() {
			p.Y = 1
		} else if engo.Input.Button("left").Down() {
			p.X = -1
		} else if engo.Input.Button("right").Down() {
			p.X = 1
		}
	}
	if engo.Input.Button("left").JustReleased() || engo.Input.Button("right").JustReleased() {
		p.X = 0
		changed = true
		if engo.Input.Button("left").Down() {
			p.X = -1
		} else if engo.Input.Button("right").Down() {
			p.X = 1
		} else if engo.Input.Button("up").Down() {
			p.Y = -1
		} else if engo.Input.Button("down").Down() {
			p.Y = 1
		}
	}
	changed = changed || p.X != 0 || p.Y != 0
	return
}
