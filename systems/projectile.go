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

const (
	// Projectile physics constants
	projectileSpeed    float32 = 400  // world units per second
	projectileLifetime float32 = 3.0  // seconds before despawn
	projectileSize     float32 = 8.0  // billboard width/height
	projectileRadius   float32 = 10.0 // collision detection radius
)

// ProjectileComponent holds all data for a projectile entity.
type ProjectileComponent struct {
	// Velocity is the current movement vector in world space (units/sec).
	Velocity engo.Point
	// Lifetime is the remaining time before despawn (seconds).
	Lifetime float32
	// Tex is the texture shown on the 3D billboard. Nil renders a solid colour.
	Tex *gl.Texture
}

func (c *ProjectileComponent) GetProjectileComponent() *ProjectileComponent { return c }

// ProjectileFace is the minimal interface for the ProjectileComponent accessor.
type ProjectileFace interface {
	GetProjectileComponent() *ProjectileComponent
}

// ProjectileAble is implemented by any entity that can be managed by ProjectileSystem.
type ProjectileAble interface {
	common.BasicFace
	common.SpaceFace
	ProjectileFace
}

// NotProjectileComponent marks entities that should not be processed by ProjectileSystem.
type NotProjectileComponent struct{}

func (n *NotProjectileComponent) GetNotProjectileComponent() *NotProjectileComponent { return n }

// projectileEntity is the system's internal representation of one projectile.
type projectileEntity struct {
	*ecs.BasicEntity
	*common.SpaceComponent
	*ProjectileComponent

	// billboard is the 3D view entity; uses shaders.ViewShader.
	billboard struct {
		ecs.BasicEntity
		common.RenderComponent
		common.SpaceComponent
	}

	// mapDot is the minimap pixel; uses shaders.MinimapShader.
	mapDot struct {
		ecs.BasicEntity
		common.RenderComponent
		common.SpaceComponent
	}
}

// ProjectileSystem manages projectile entities. It tracks the player to know
// their position for spawning new projectiles and handles projectile physics,
// collision, and despawning.
type ProjectileSystem struct {
	w           *ecs.World
	player      *common.SpaceComponent
	projectiles []*projectileEntity
}

func (s *ProjectileSystem) New(w *ecs.World) {
	s.w = w
}

func (s *ProjectileSystem) AddByInterface(i ecs.Identifier) {
	// Accept the player entity so we can track their position/rotation.
	if o, ok := i.(ViewPlayerAble); ok {
		s.player = o.GetSpaceComponent()
		return
	}

	// Accept projectile entities.
	o, ok := i.(ProjectileAble)
	if !ok {
		return
	}

	pc := o.GetProjectileComponent()
	sp := o.GetSpaceComponent()

	proj := &projectileEntity{
		BasicEntity:         o.GetBasicEntity(),
		SpaceComponent:      sp,
		ProjectileComponent: pc,
	}

	// ── 3D billboard ─────────────────────────────────────────────────────
	proj.billboard.BasicEntity = ecs.NewBasic()
	proj.billboard.SpaceComponent = common.SpaceComponent{
		Position: sp.Position,
		Width:    projectileSize,
		Height:   projectileSize,
	}
	proj.billboard.RenderComponent = common.RenderComponent{
		Drawable: shaders.Billboard{
			Pos: sp.Position,
			W:   projectileSize,
			H:   projectileSize,
			Tex: pc.Tex,
		},
		Color: color.RGBA{0xff, 0xff, 0xff, 0xff},
	}
	if pc.Tex == nil {
		// No texture: render as a solid red sphere.
		proj.billboard.RenderComponent.Color = color.RGBA{0xff, 0x33, 0x00, 0xff}
	}
	proj.billboard.SetShader(shaders.ViewShader)
	s.w.AddEntity(&proj.billboard)

	// ── Minimap dot ───────────────────────────────────────────────────────
	const dotSize float32 = 3
	proj.mapDot.BasicEntity = ecs.NewBasic()
	proj.mapDot.SpaceComponent = common.SpaceComponent{
		Position: engo.Point{
			X: sp.Position.X + MapWallOffsetX - dotSize/2,
			Y: sp.Position.Y + MapWallOffsetY - dotSize/2,
		},
		Width:  dotSize,
		Height: dotSize,
	}
	proj.mapDot.RenderComponent = common.RenderComponent{
		Drawable:    common.Rectangle{},
		Color:       color.RGBA{0xff, 0x00, 0x00, 0xff}, // bright red
		StartZIndex: 4,
	}
	proj.mapDot.SetShader(shaders.MinimapShader)
	s.w.AddEntity(&proj.mapDot)

	s.projectiles = append(s.projectiles, proj)
}

func (s *ProjectileSystem) Remove(basic ecs.BasicEntity) {
	for i, proj := range s.projectiles {
		if proj.BasicEntity.ID() == basic.ID() {
			// Remove billboard and map dot from the world
			for _, sys := range s.w.Systems() {
				sys.Remove(proj.billboard.BasicEntity)
				sys.Remove(proj.mapDot.BasicEntity)
			}
			s.projectiles = append(s.projectiles[:i], s.projectiles[i+1:]...)
			return
		}
	}
}

func (s *ProjectileSystem) Update(dt float32) {
	if s.player == nil {
		return
	}

	const near float32 = 1.0
	po := shaders.PlayerOffset
	playerX := s.player.Position.X - po.X
	playerY := s.player.Position.Y - po.Y

	sin, cos := math.Sincos(s.player.Rotation * math.Pi / 180)

	// Process each projectile
	for i := len(s.projectiles) - 1; i >= 0; i-- {
		proj := s.projectiles[i]

		// ── Update lifetime ──────────────────────────────────────────────
		proj.Lifetime -= dt
		if proj.Lifetime <= 0 {
			// Despawn projectile
			s.Remove(*proj.BasicEntity)
			continue
		}

		// ── Move projectile ──────────────────────────────────────────────
		proj.SpaceComponent.Position.X += proj.Velocity.X * dt
		proj.SpaceComponent.Position.Y += proj.Velocity.Y * dt

		// Update billboard and map dot positions
		proj.billboard.SpaceComponent.Position = proj.SpaceComponent.Position
		proj.billboard.Drawable = shaders.Billboard{
			Pos: proj.SpaceComponent.Position,
			W:   projectileSize,
			H:   projectileSize,
			Tex: proj.Tex,
		}
		proj.mapDot.SpaceComponent.Position = engo.Point{
			X: proj.SpaceComponent.Position.X + MapWallOffsetX - proj.mapDot.Width/2,
			Y: proj.SpaceComponent.Position.Y + MapWallOffsetY - proj.mapDot.Height/2,
		}

		// ── Depth z-sorting for the billboard ────────────────────────────
		// Transform projectile into camera space to get depth.
		relX := proj.SpaceComponent.Position.X - playerX
		relY := -proj.SpaceComponent.Position.Y + playerY
		camY := relY*cos + relX*sin // camera-space depth

		if camY < near {
			// Behind the player; hide the billboard.
			proj.billboard.Hidden = true
			continue
		}

		proj.billboard.Hidden = false

		// Clamp to near so depth is never negative.
		dy2 := camY
		if dy2 < near {
			dy2 = near
		}
		// Offset of 50 matches ViewSystem and ItemSystem.
		proj.billboard.SetZIndex(-(dy2 + 50))
	}
}

// SpawnProjectile creates a new projectile at the player's position,
// fired in the direction the player is facing.
func (s *ProjectileSystem) SpawnProjectile(tex *gl.Texture) {
	if s.player == nil {
		return
	}

	po := shaders.PlayerOffset
	// Spawn slightly in front of the player to avoid self-collision
	spawnOffset := float32(15.0)

	sin, cos := math.Sincos(s.player.Rotation * math.Pi / 180)

	// Calculate spawn position (player position + offset in facing direction)
	spawnX := (s.player.Position.X - po.X) + spawnOffset*sin
	spawnY := (s.player.Position.Y - po.Y) - spawnOffset*cos

	// Calculate velocity vector in facing direction
	velX := projectileSpeed * sin
	velY := -projectileSpeed * cos

	// Create projectile entity
	e := &projectileEntity{
		BasicEntity: new(ecs.BasicEntity),
		SpaceComponent: &common.SpaceComponent{
			Position: engo.Point{X: spawnX, Y: spawnY},
			Width:    projectileSize,
			Height:   projectileSize,
		},
		ProjectileComponent: &ProjectileComponent{
			Velocity: engo.Point{X: velX, Y: velY},
			Lifetime: projectileLifetime,
			Tex:      tex,
		},
	}
	*e.BasicEntity = ecs.NewBasic()

	// Add it to the system (which will create billboard and map dot)
	s.AddByInterface(e)
}
