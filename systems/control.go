package systems

import (
	"image/color"

	"github.com/EngoEngine/ecs"
	"github.com/EngoEngine/engo"
	"github.com/EngoEngine/engo/common"
	"github.com/EngoEngine/engo/math"
)

const (
	// Health bar HUD position and dimensions (screen coordinates).
	healthBarX float32 = 10
	healthBarY float32 = 185
	healthBarW float32 = 300
	healthBarH float32 = 10

	// Stamina bar HUD position and dimensions (screen coordinates).
	staminaBarX float32 = 10
	staminaBarY float32 = 200
	staminaBarW float32 = 300
	staminaBarH float32 = 10

	// Stamina rates in units per second.
	staminaDrainRate float32 = 20 // drained per second while sprinting
	staminaRegenRate float32 = 8  // regenerated per second while not sprinting
	staminaResumeAt  float32 = 50 // stamina level at which exhaustion clears

	// Speed multipliers applied on top of the entity's base Speed.
	sprintMultiplier float32 = 2.0
	crouchSpeedMul   float32 = 0.5

	// Crouch adds this fraction of NormalHeight to raise the Height value,
	// which lowers the camera view (higher Height = floor rises = view goes down).
	crouchHeightMul float32 = 0.5

	// Jump physics.  The jump arc works by briefly *decreasing* Height (which
	// raises the view), then gravity restores Height back to baseHeight.
	// Values are kept small so Height never goes negative (NormalHeight ≈ 10).
	jumpInitVel float32 = 60  // initial speed at which Height decreases (units/sec)
	gravity     float32 = 250 // rate at which Height is restored (units/sec²)
)

// ControlComponent holds movement parameters and runtime state for a
// player-controlled entity.
type ControlComponent struct {
	// Speed is the base translation speed in world-units per second.
	Speed float32
	// RotSpeed is the rotation speed in degrees per second (mouse axis).
	RotSpeed float32

	// Health is the player's hit-points in the range [0, 100].
	// It is initialised to 100 by ControlSystem.Add when the value is zero.
	Health float32

	// Stamina is the sprint resource in the range [0, 100].
	// It is initialised to 100 by ControlSystem.Add when the value is zero.
	Stamina float32

	// NormalHeight is captured from SpaceComponent.Height on the very first
	// Update tick.  Crouch and jump are expressed relative to this value.
	NormalHeight float32

	// unexported runtime state
	exhausted    bool    // true when stamina hit 0; cleared when Stamina >= staminaResumeAt
	isJumping    bool    // true while the player is airborne
	jumpVelocity float32 // current velocity magnitude; positive decreases Height (view goes up)

	velocity engo.Point // current horizontal movement vector (world-units/frame)
}

func (c *ControlComponent) GetControlComponent() *ControlComponent { return c }

// ControlFace is satisfied by any component that embeds *ControlComponent.
type ControlFace interface {
	GetControlComponent() *ControlComponent
}

// ControlAble is the interface required for AddByInterface.
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

// hudBar is a minimal HUD entity used internally by ControlSystem for the
// stamina bar background and foreground rectangles.
type hudBar struct {
	ecs.BasicEntity
	common.RenderComponent
	common.SpaceComponent
}

// ControlSystem handles keyboard-driven movement, rotation, sprinting,
// crouching, jumping, and renders a health and stamina bar as a HUD overlay.
type ControlSystem struct {
	entities []controlEntity
	w        *ecs.World

	healthBarBg  hudBar // dark background, always full width
	healthBarFg  hudBar // coloured foreground, width scales with health
	staminaBarBg hudBar // dark background, always full width
	staminaBarFg hudBar // coloured foreground, width scales with stamina
}

func (s *ControlSystem) New(w *ecs.World) {
	s.w = w

	// ── Health bar background (dark, full width) ──────────────────────────
	s.healthBarBg = hudBar{BasicEntity: ecs.NewBasic()}
	s.healthBarBg.SpaceComponent = common.SpaceComponent{
		Position: engo.Point{X: healthBarX, Y: healthBarY},
		Width:    healthBarW,
		Height:   healthBarH,
	}
	s.healthBarBg.RenderComponent = common.RenderComponent{
		Drawable:    common.Rectangle{},
		Color:       color.RGBA{0x30, 0x30, 0x30, 0xCC},
		StartZIndex: 10,
	}
	s.healthBarBg.SetShader(common.LegacyHUDShader)
	w.AddEntity(&s.healthBarBg)

	// ── Health bar foreground (coloured, variable width) ──────────────────
	s.healthBarFg = hudBar{BasicEntity: ecs.NewBasic()}
	s.healthBarFg.SpaceComponent = common.SpaceComponent{
		Position: engo.Point{X: healthBarX, Y: healthBarY},
		Width:    healthBarW,
		Height:   healthBarH,
	}
	s.healthBarFg.RenderComponent = common.RenderComponent{
		Drawable:    common.Rectangle{},
		Color:       color.RGBA{0x22, 0xFF, 0x44, 0xFF},
		StartZIndex: 11,
	}
	s.healthBarFg.SetShader(common.LegacyHUDShader)
	w.AddEntity(&s.healthBarFg)

	// ── Stamina bar background (dark, full width) ────────────────────────
	s.staminaBarBg = hudBar{BasicEntity: ecs.NewBasic()}
	s.staminaBarBg.SpaceComponent = common.SpaceComponent{
		Position: engo.Point{X: staminaBarX, Y: staminaBarY},
		Width:    staminaBarW,
		Height:   staminaBarH,
	}
	s.staminaBarBg.RenderComponent = common.RenderComponent{
		Drawable:    common.Rectangle{},
		Color:       color.RGBA{0x30, 0x30, 0x30, 0xCC},
		StartZIndex: 10,
	}
	s.staminaBarBg.SetShader(common.LegacyHUDShader)
	w.AddEntity(&s.staminaBarBg)

	// ── Foreground bar (coloured, variable width) ─────────────────────────
	s.staminaBarFg = hudBar{BasicEntity: ecs.NewBasic()}
	s.staminaBarFg.SpaceComponent = common.SpaceComponent{
		Position: engo.Point{X: staminaBarX, Y: staminaBarY},
		Width:    staminaBarW,
		Height:   staminaBarH,
	}
	s.staminaBarFg.RenderComponent = common.RenderComponent{
		Drawable:    common.Rectangle{},
		Color:       color.RGBA{0x00, 0xFF, 0x44, 0xFF},
		StartZIndex: 11,
	}
	s.staminaBarFg.SetShader(common.LegacyHUDShader)
	w.AddEntity(&s.staminaBarFg)
}

func (s *ControlSystem) Add(
	basic *ecs.BasicEntity,
	control *ControlComponent,
	space *common.SpaceComponent,
) {
	// Default health and stamina to full when the caller hasn't pre-set them.
	if control.Health == 0 {
		control.Health = 100
	}
	if control.Stamina == 0 {
		control.Stamina = 100
	}
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
	del := -1
	for i, e := range s.entities {
		if e.BasicEntity.ID() == basic.ID() {
			del = i
			break
		}
	}
	if del >= 0 {
		s.entities = append(s.entities[:del], s.entities[del+1:]...)
	}
}

func (s *ControlSystem) Update(dt float32) {
	for _, entity := range s.entities {
		// Capture the resting eye-height once (after all system initialisers
		// have had a chance to set SpaceComponent.Height).
		if entity.NormalHeight == 0 {
			entity.NormalHeight = entity.SpaceComponent.Height
		}

		// ── Sprint / Stamina ──────────────────────────────────────────────
		wantSprint := engo.Input.Button("sprint").Down()
		canSprint := !entity.exhausted && entity.Stamina > 0
		sprinting := wantSprint && canSprint

		if sprinting {
			entity.Stamina -= staminaDrainRate * dt
			if entity.Stamina <= 0 {
				entity.Stamina = 0
				entity.exhausted = true
			}
		} else {
			entity.Stamina += staminaRegenRate * dt
			if entity.Stamina > 100 {
				entity.Stamina = 100
			}
			// Exhaustion only clears once stamina is sufficiently recovered.
			if entity.exhausted && entity.Stamina >= staminaResumeAt {
				entity.exhausted = false
			}
		}

		// ── Crouch ───────────────────────────────────────────────────────
		crouching := engo.Input.Button("crouch").Down()

		// ── Jump ─────────────────────────────────────────────────────────
		if engo.Input.Button("jump").JustPressed() && !entity.isJumping {
			entity.isJumping = true
			entity.jumpVelocity = jumpInitVel
		}

		// Base eye-height depends on whether we are crouching.
		// Crouching *increases* Height, which makes the floor rise on screen
		// (projection: screen_y = -Height * focal/depth + h/2), lowering the view.
		baseHeight := entity.NormalHeight
		if crouching {
			baseHeight = entity.NormalHeight * (1 + crouchHeightMul)
		}

		if entity.isJumping {
			// Decreasing Height raises the view (floor descends on screen).
			// gravity gradually restores Height back toward baseHeight.
			entity.jumpVelocity -= gravity * dt
			entity.SpaceComponent.Height -= entity.jumpVelocity * dt

			// Land when Height has risen back to (or above) baseHeight.
			if entity.SpaceComponent.Height >= baseHeight {
				entity.SpaceComponent.Height = baseHeight
				entity.isJumping = false
				entity.jumpVelocity = 0
			}
		} else {
			// Keep height locked to baseHeight when grounded.
			entity.SpaceComponent.Height = baseHeight
		}

		// ── Horizontal movement ───────────────────────────────────────────
		effectiveSpeed := entity.Speed
		if sprinting {
			effectiveSpeed *= sprintMultiplier
		}
		if crouching {
			effectiveSpeed *= crouchSpeedMul
		}

		// Recompute direction every frame from currently-held keys so that
		// speed changes (sprint/crouch toggle) take effect immediately.
		dir := s.getDirection()
		if dir.X != 0 || dir.Y != 0 {
			dir, _ = dir.Normalize()
		}
		dir.MultiplyScalar(dt * effectiveSpeed)
		entity.velocity = dir

		// Apply rotation (mouse x-axis) around the entity centre so the
		// pivot stays fixed under the cursor.
		ctr := entity.GetSpaceComponent().Center()
		entity.Rotation += math.Clamp(
			engo.Input.Axis("hori").Value()*entity.RotSpeed*dt,
			-5, 5,
		)
		entity.SetCenter(ctr)

		// Rotate the flat movement vector into world space and translate.
		sin, cos := math.Sincos(entity.Rotation * math.Pi / 180)
		delta := engo.Point{
			X: entity.velocity.X*cos - entity.velocity.Y*sin,
			Y: entity.velocity.Y*cos + entity.velocity.X*sin,
		}
		entity.Position.Add(delta)
	}

	// ── Health and Stamina HUD update ────────────────────────────────────
	if len(s.entities) == 0 {
		return
	}
	e := s.entities[0]

	// Health bar
	healthFrac := e.Health / 100
	if healthFrac < 0 {
		healthFrac = 0
	}
	if healthFrac > 1 {
		healthFrac = 1
	}
	s.healthBarFg.Width = healthBarW * healthFrac
	switch {
	case e.Health < 25:
		s.healthBarFg.Color = color.RGBA{0xFF, 0x22, 0x22, 0xFF} // red: critical
	case e.Health < 60:
		s.healthBarFg.Color = color.RGBA{0xFF, 0xCC, 0x00, 0xFF} // yellow: low
	default:
		s.healthBarFg.Color = color.RGBA{0x22, 0xFF, 0x44, 0xFF} // green: healthy
	}

	frac := e.Stamina / 100
	if frac < 0 {
		frac = 0
	}
	s.staminaBarFg.Width = staminaBarW * frac

	switch {
	case e.exhausted:
		// Red: sprinting locked out until stamina recovers past 50.
		s.staminaBarFg.Color = color.RGBA{0xFF, 0x33, 0x33, 0xFF}
	case e.Stamina < 30:
		// Yellow: getting low.
		s.staminaBarFg.Color = color.RGBA{0xFF, 0xCC, 0x00, 0xFF}
	default:
		// Green: healthy.
		s.staminaBarFg.Color = color.RGBA{0x00, 0xFF, 0x44, 0xFF}
	}
}

// getDirection returns the unit movement direction for whichever WASD / arrow
// keys are currently held.  Diagonal inputs are normalised by the caller.
func (s *ControlSystem) getDirection() engo.Point {
	var p engo.Point
	if engo.Input.Button("up").Down() {
		p.Y = -1
	} else if engo.Input.Button("down").Down() {
		p.Y = 1
	}
	if engo.Input.Button("left").Down() {
		p.X = -1
	} else if engo.Input.Button("right").Down() {
		p.X = 1
	}
	return p
}
