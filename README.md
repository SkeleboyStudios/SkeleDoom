# SkeleDoom
A ray-casted doom-like game for the skeleboy studios website

## Controls

- **WASD / Arrow Keys**: Move forward, backward, strafe left/right
- **Mouse**: Look around (horizontal mouse movement)
- **Left Mouse Button**: Shoot projectiles
- **Left Shift**: Sprint (consumes stamina)
- **Left Control**: Crouch (reduces movement speed and lowers view)
- **Space**: Jump

## Features

- First-person raycasting 3D view with textured walls
- **Animated weapon sprites** (8-frame sprite sheet support)
- **Shooting mechanics** with smooth firing animation
- Projectile billboards that follow the player's view
- Health and stamina bars (HUD)
- Sprint and crouch mechanics with stamina system
- Jump physics
- Item pickups (potions)
- Lava damage zones
- Minimap showing player position, walls, items, and projectiles

## Weapon Sprites

The game uses sprite sheets for weapon animations. Place your weapon sprite sheet at `ui/pistol.png`.

**Sprite sheet format**:
- 8 frames in a horizontal strip
- Frame 0: Idle
- Frames 1-2: Firing animation
- Frame 3: Cooldown
- Frames 4-7: Reload (reserved for future use)

Sprite assets from [OpenGameArt - Lowres FPS Gun Sprites](https://opengameart.org/content/lowres-fps-gun-sprites)
