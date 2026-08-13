package systems

import (
	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/gl"
)

// ShootingSystem monitors the player's shooting state and spawns projectiles.
// It requires both the player (via ControlAble) and the ProjectileSystem to
// coordinate shooting mechanics.
type ShootingSystem struct {
	w                *ecs.World
	player           *ControlComponent
	projectileSystem *ProjectileSystem
	projectileTex    *gl.Texture
}

func (s *ShootingSystem) New(w *ecs.World) {
	s.w = w
}

// SetProjectileSystem links this system to the ProjectileSystem so it can
// spawn projectiles. Call this after both systems are added to the world.
func (s *ShootingSystem) SetProjectileSystem(ps *ProjectileSystem) {
	s.projectileSystem = ps
}

// SetProjectileTexture sets the texture used for spawned projectiles.
func (s *ShootingSystem) SetProjectileTexture(tex *gl.Texture) {
	s.projectileTex = tex
}

func (s *ShootingSystem) AddByInterface(i ecs.Identifier) {
	// Track the player to monitor shooting state
	if o, ok := i.(ControlAble); ok {
		s.player = o.GetControlComponent()
	}
}

func (s *ShootingSystem) Remove(basic ecs.BasicEntity) {
	// Nothing to remove
}

func (s *ShootingSystem) Update(dt float32) {
	if s.player == nil || s.projectileSystem == nil {
		return
	}

	// Check if player shot this frame
	if s.player.IsShooting {
		s.projectileSystem.SpawnProjectile(s.projectileTex)
	}
}
