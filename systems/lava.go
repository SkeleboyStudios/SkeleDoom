package systems

import (
	"image/color"
	"math"

	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"
	"github.com/SkeleboyStudios/SkeleDoom/shaders"
)

// ─── Component ───────────────────────────────────────────────────────────────

// LavaZoneComponent defines a rectangular damage zone in wall world-space
// (the same coordinate space as wall endpoints in the scene file).
// Position and size are stored in the entity's SpaceComponent.
type LavaZoneComponent struct {
	// Color is shown on the minimap and used for the damage-flash vignette.
	Color color.RGBA
	// DPS is the damage per second dealt to a grounded player inside the zone.
	DPS float32
}

func (c *LavaZoneComponent) GetLavaZoneComponent() *LavaZoneComponent { return c }

// LavaZoneFace is satisfied by anything that embeds *LavaZoneComponent.
type LavaZoneFace interface {
	GetLavaZoneComponent() *LavaZoneComponent
}

// LavaZoneAble is the interface AddByInterface uses to detect zone entities.
// Requiring SpaceFace and CollisionFace (in addition to LavaZoneFace) means
// the lavaZone scene entity automatically satisfies common.Collisionable, so
// engo's CollisionSystem registers it without any extra wiring.
type LavaZoneAble interface {
	common.BasicFace
	common.SpaceFace
	common.CollisionFace
	LavaZoneFace
}

// ─── Player interface ─────────────────────────────────────────────────────────

// LavaPlayerAble is satisfied by the player entity, which carries a
// ControlComponent holding isJumping (grounded check) and Health (damage sink).
type LavaPlayerAble interface {
	common.BasicFace
	ControlFace
}

// ─── Internal entity types ───────────────────────────────────────────────────

type lavaPlayerEntity struct {
	*ecs.BasicEntity
	*ControlComponent
}

type lavaZoneEntity struct {
	*ecs.BasicEntity
	*common.SpaceComponent
	*LavaZoneComponent
	*common.CollisionComponent        // read Collides each frame
	mapRect                    sprite // minimap rectangle owned by LavaSystem
}

// ─── System ──────────────────────────────────────────────────────────────────

const (
	// vignetteMaxAlpha is the peak opacity (0-255) of the damage flash.
	vignetteMaxAlpha uint8 = 0x55 // ≈ 33 % — visible but not overwhelming

	// vignettePulseHz is how many full oscillations per second when in lava.
	vignettePulseHz float64 = 2.0

	// vignetteFadeRate is how quickly the vignette fades out once the player
	// leaves the zone, in alpha units per second.
	vignetteFadeRate float32 = 300
)

// LavaSystem applies per-zone DPS to the player while they are grounded inside
// a zone, renders each zone as a coloured minimap rectangle, and shows a
// pulsing full-screen vignette while the player is taking damage.
//
// Collision detection is delegated entirely to engo's CollisionSystem:
//   - Each lava zone entity carries Main: CollisionGroupLava.
//   - The player's CollisionComponent carries Group: CollisionGroupLava
//     (set by MapSystem).
//   - CollisionSystem sets zone.CollisionComponent.Collides to
//     CollisionGroupLava every frame the two shapes overlap, and resets it to
//     zero when they don't.  LavaSystem reads that flag; no manual AABB math
//     is required.
//   - CollisionGroupLava is absent from CollisionSystem.Solids, so the player
//     is never pushed back out of a lava zone.
type LavaSystem struct {
	w         *ecs.World
	player    lavaPlayerEntity
	hasPlayer bool
	zones     []lavaZoneEntity

	// vignette is a full-screen HUD sprite used for the damage flash.
	vignette      sprite
	vignetteAlpha uint8      // tracked separately; color.Color is an interface
	flashTimer    float32    // time accumulator driving the sine pulse
	lastColor     color.RGBA // colour of the most-recently active zone
}

func (s *LavaSystem) New(w *ecs.World) {
	s.w = w

	// Full-screen overlay — starts fully transparent.
	s.vignette = sprite{BasicEntity: ecs.NewBasic()}
	s.vignette.SpaceComponent = common.SpaceComponent{
		Position: engo.Point{X: 0, Y: 0},
		Width:    640,
		Height:   360,
	}
	s.vignette.RenderComponent = common.RenderComponent{
		Drawable:    common.Rectangle{},
		Color:       color.RGBA{0, 0, 0, 0},
		StartZIndex: 50, // above walls and minimap, below any critical HUD
	}
	s.vignette.SetShader(common.LegacyHUDShader)
	w.AddEntity(&s.vignette)
}

func (s *LavaSystem) AddByInterface(i ecs.Identifier) {
	// ── Player ────────────────────────────────────────────────────────────
	if o, ok := i.(LavaPlayerAble); ok {
		s.player = lavaPlayerEntity{
			BasicEntity:      o.GetBasicEntity(),
			ControlComponent: o.GetControlComponent(),
		}
		s.hasPlayer = true
	}

	// ── Lava zone ────────────────────────────────────────────────────────
	if o, ok := i.(LavaZoneAble); ok {
		z := o.GetLavaZoneComponent()

		// The caller stores the zone's position/size directly in the entity's
		// SpaceComponent (wall world-space).  We save the raw values, then
		// translate the SpaceComponent into player/collision-space so that
		// engo's CollisionSystem sees matching coordinates (same shift MapSystem
		// applies to the player's SpaceComponent).
		space := o.GetSpaceComponent()
		origX := space.Position.X
		origY := space.Position.Y
		origW := space.Width
		origH := space.Height
		space.Position.X = origX + MapWallOffsetX
		space.Position.Y = origY + MapWallOffsetY

		// Tag this entity as a lava initiator.  CollisionGroupLava is NOT in
		// CollisionSystem.Solids, so collisions are detected (Collides updated)
		// but the player is never pushed back.
		collision := o.GetCollisionComponent()
		collision.Main = CollisionGroupLava
		collision.Group = CollisionGroupPlaya

		zone := lavaZoneEntity{
			BasicEntity:        o.GetBasicEntity(),
			SpaceComponent:     space,
			LavaZoneComponent:  z,
			CollisionComponent: collision,
		}

		// Minimap rectangle — visual only, positioned in wall-offset space so
		// it lines up with the wall geometry drawn by MapSystem.
		zone.mapRect = sprite{BasicEntity: ecs.NewBasic()}
		zone.mapRect.SpaceComponent = common.SpaceComponent{
			Position: engo.Point{
				X: origX + MapWallOffsetX,
				Y: origY + MapWallOffsetY,
			},
			Width:  origW,
			Height: origH,
		}
		// Draw below walls (z=6) and the player dot (z=5) so it doesn't cover them.
		zone.mapRect.RenderComponent = common.RenderComponent{
			Drawable:    common.Rectangle{},
			Color:       z.Color,
			StartZIndex: 3,
		}
		zone.mapRect.SetShader(shaders.MinimapShader)
		s.w.AddEntity(&zone.mapRect)

		s.zones = append(s.zones, zone)
	}
}

func (s *LavaSystem) Remove(basic ecs.BasicEntity) {
	for i, z := range s.zones {
		if z.BasicEntity.ID() == basic.ID() {
			s.zones = append(s.zones[:i], s.zones[i+1:]...)
			return
		}
	}
}

func (s *LavaSystem) Update(dt float32) {
	if !s.hasPlayer {
		return
	}

	// The player must be grounded (not jumping) for lava to deal damage.
	grounded := !s.player.isJumping

	// Ask the CollisionSystem's results: each zone's Collides field is set to
	// CollisionGroupLava by the CollisionSystem when it overlaps the player
	// (player has Group: CollisionGroupLava set by MapSystem), and reset to
	// zero when they don't.  No position math needed here.
	var totalDPS float32
	var activeColor color.RGBA
	inAnyZone := false

	for _, zone := range s.zones {
		if zone.CollisionComponent.Collides != 0 {
			inAnyZone = true
			activeColor = zone.LavaZoneComponent.Color
			if grounded {
				totalDPS += zone.LavaZoneComponent.DPS
			}
		}
	}

	// Apply accumulated damage.
	if totalDPS > 0 {
		s.player.Health -= totalDPS * dt
		if s.player.Health < 0 {
			s.player.Health = 0
		}
	}

	// ── Vignette update ──────────────────────────────────────────────────
	if inAnyZone {
		s.lastColor = activeColor
		s.flashTimer += dt

		// Sine wave oscillates between 0 and vignetteMaxAlpha.
		sine := math.Sin(float64(s.flashTimer) * vignettePulseHz * 2 * math.Pi)
		s.vignetteAlpha = uint8(float64(vignetteMaxAlpha) * (0.5 + 0.5*sine))

		s.vignette.Color = color.RGBA{
			R: activeColor.R,
			G: activeColor.G,
			B: activeColor.B,
			A: s.vignetteAlpha,
		}
	} else {
		// Fade out smoothly once the player leaves (or jumps over) the zone.
		s.flashTimer = 0
		if s.vignetteAlpha > 0 {
			fade := vignetteFadeRate * dt
			if fade >= float32(s.vignetteAlpha) {
				s.vignetteAlpha = 0
				s.vignette.Color = color.RGBA{}
			} else {
				s.vignetteAlpha -= uint8(fade)
				s.vignette.Color = color.RGBA{
					R: s.lastColor.R,
					G: s.lastColor.G,
					B: s.lastColor.B,
					A: s.vignetteAlpha,
				}
			}
		}
	}
}
