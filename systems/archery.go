package systems

import (
	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo/common"
	"github.com/EngoEngine/gl"
)

type Ammunition struct {
	ProjectileTex *gl.Texture
	ReloadTime    float32
	TimeBtwnShots float32
	Loaded, Cap   int
}

// ArcheryComponent is the ability to fire projectiles.
type ArcheryComponent struct {
	IsShooting   bool
	IsReloading  bool
	ShotCooldown float32
	Ammo         Ammunition
}

func (c *ArcheryComponent) GetArcheryComponent() *ArcheryComponent {
	return c
}

type ArcheryFace interface {
	GetArcheryComponent() *ArcheryComponent
}

type ArcheryAble interface {
	common.BasicFace
	ArcheryFace
	common.AnimationFace
}

type archeryEntity struct {
	*ecs.BasicEntity
	*ArcheryComponent
	*common.AnimationComponent
}

// ArcherySystem is the system that handles the firing of projectiles. Any
// entity with an ArcheryComponent can fire a projectile. This system
// handles the spawning of the projectile and associated animations.
// Projectiles are then handled by the Projectile system.
type ArcherySystem struct {
	entities         []archeryEntity
	projectileSystem *ProjectileSystem
}

// SetProjectileSystem links this system to the ProjectileSystem so it can
// spawn projectiles. Call this after both systems are added to the world.
func (s *ArcherySystem) SetProjectileSystem(ps *ProjectileSystem) {
	s.projectileSystem = ps
}

func (s *ArcherySystem) AddByInterface(i ecs.Identifier) {
	if o, ok := i.(ArcheryAble); ok {
		s.Add(o.GetBasicEntity(), o.GetArcheryComponent(), o.GetAnimationComponent())
	}
}

func (s *ArcherySystem) Add(basic *ecs.BasicEntity, arch *ArcheryComponent, anim *common.AnimationComponent) {
	s.entities = append(s.entities, archeryEntity{basic, arch, anim})
}

func (s *ArcherySystem) Remove(basic ecs.BasicEntity) {
	d := -1
	for i, entity := range s.entities {
		if entity.ID() == basic.ID() {
			d = i
		}
	}
	if d >= 0 {
		s.entities = append(s.entities[:d], s.entities[d+1:]...)
	}
}

func (s *ArcherySystem) Update(dt float32) {
	for _, entity := range s.entities {
		entity.ShotCooldown -= dt
		if entity.ShotCooldown <= 0 {
			if entity.IsShooting && entity.Ammo.Loaded > 0 {
				s.projectileSystem.SpawnProjectile(entity.Ammo.ProjectileTex)
				entity.Ammo.Loaded -= 1
				entity.SelectAnimationByName("shoot")
				entity.ShotCooldown = entity.Ammo.TimeBtwnShots
			}
		}
		if entity.ShotCooldown <= 0 {
			if entity.IsReloading {
				entity.Ammo.Loaded = entity.Ammo.Cap
				entity.SelectAnimationByName("reload")
				entity.ShotCooldown = entity.Ammo.ReloadTime
			}
		}
		if entity.ShotCooldown < -2e20 {
			entity.ShotCooldown = 0
		}
	}
}
