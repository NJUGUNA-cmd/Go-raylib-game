package main

import (
	"fmt"
	"math"
	"os"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ============================================================================
// WAVEFORM ANALYSIS & BEAT DETECTION
// ============================================================================

type AudioAnalyzer struct {
	samples          []float32
	beats            []Beat
	sampleRate       int32
	samplesPerWindow int
}

type Beat struct {
	TimeSeconds float32
	TimeStamp   float32
	Strength    float32
	Index       int
}

func NewAudioAnalyzer(filepath string) *AudioAnalyzer {
	wave := rl.LoadWave(filepath)
	if wave.FrameCount == 0 {
		fmt.Printf("Error: Could not load audio file: %s\n", filepath)
		os.Exit(1)
	}

	fmt.Printf("Loaded audio: %d frames, %d channels, %d sample rate\n",
		wave.FrameCount, wave.Channels, wave.SampleRate)

	analyzer := &AudioAnalyzer{
		samples:          []float32{},
		beats:            []Beat{},
		sampleRate:       int32(wave.SampleRate),
		samplesPerWindow: int(wave.SampleRate) / 100,
	}

	analyzer.extractAmplitudeEnvelope(wave)
	analyzer.detectBeats()
	rl.UnloadWave(wave)

	fmt.Printf("Analysis complete: %d samples, %d beats detected\n",
		len(analyzer.samples), len(analyzer.beats))

	return analyzer
}

func (aa *AudioAnalyzer) extractAmplitudeEnvelope(wave rl.Wave) {
	data := rl.LoadWaveSamples(wave)
	defer rl.UnloadWaveSamples(data)

	totalFrames := int(wave.FrameCount)
	channels := int(wave.Channels)
	windowSizeInFrames := aa.samplesPerWindow

	for frameIdx := 0; frameIdx < totalFrames; frameIdx += windowSizeInFrames {
		endFrame := frameIdx + windowSizeInFrames
		if endFrame > totalFrames {
			endFrame = totalFrames
		}

		sumSquares := float32(0)
		count := 0

		for f := frameIdx; f < endFrame; f++ {
			for ch := 0; ch < channels; ch++ {
				sampleIdx := f*channels + ch
				if sampleIdx >= len(data) {
					break
				}
				sample := data[sampleIdx]
				sumSquares += sample * sample
				count++
			}
		}

		rms := float32(0)
		if count > 0 {
			rms = float32(math.Sqrt(float64(sumSquares / float32(count))))
		}
		aa.samples = append(aa.samples, rms)
	}
}

func (aa *AudioAnalyzer) detectBeats() {
	if len(aa.samples) < 3 {
		return
	}

	avgAmplitude := float32(0)
	for _, amp := range aa.samples {
		avgAmplitude += amp
	}
	avgAmplitude /= float32(len(aa.samples))

	threshold := avgAmplitude * 1.5
	minBeatGap := 10
	beatIndex := 0
	lastBeatIdx := -minBeatGap

	for i := 1; i < len(aa.samples)-1; i++ {
		current := aa.samples[i]
		prev := aa.samples[i-1]
		next := aa.samples[i+1]

		isPeak := current > prev && current > next

		if isPeak && current > threshold && (i-lastBeatIdx) >= minBeatGap {
			timeSeconds := float32(i) * 0.01

			aa.beats = append(aa.beats, Beat{
				TimeSeconds: timeSeconds,
				TimeStamp:   timeSeconds,
				Strength:    current / avgAmplitude,
				Index:       beatIndex,
			})

			lastBeatIdx = i
			beatIndex++
		}
	}

	if len(aa.beats) < 10 {
		aa.beats = []Beat{}
		threshold = avgAmplitude * 1.2
		lastBeatIdx = -minBeatGap
		beatIndex = 0

		for i := 1; i < len(aa.samples)-1; i++ {
			current := aa.samples[i]
			prev := aa.samples[i-1]
			next := aa.samples[i+1]
			isPeak := current > prev && current > next

			if isPeak && current > threshold && (i-lastBeatIdx) >= minBeatGap {
				timeSeconds := float32(i) * 0.01
				aa.beats = append(aa.beats, Beat{
					TimeSeconds: timeSeconds,
					TimeStamp:   timeSeconds,
					Strength:    current / avgAmplitude,
					Index:       beatIndex,
				})
				lastBeatIdx = i
				beatIndex++
			}
		}
	}
}

func (aa *AudioAnalyzer) GetAmplitudeAtTime(timeSeconds float32) float32 {
	idx := int(timeSeconds / 0.01)
	if idx < 0 || idx >= len(aa.samples) {
		return 0
	}
	return aa.samples[idx]
}

// ============================================================================
// AUDIO SYSTEM
// ============================================================================

type AudioSystem struct {
	music    rl.Music
	analyzer *AudioAnalyzer
}

func NewAudioSystem(filepath string) *AudioSystem {
	rl.InitAudioDevice()
	analyzer := NewAudioAnalyzer(filepath)
	music := rl.LoadMusicStream(filepath)

	return &AudioSystem{
		music:    music,
		analyzer: analyzer,
	}
}

func (as *AudioSystem) Play() {
	rl.PlayMusicStream(as.music)
}

func (as *AudioSystem) Update() {
	rl.UpdateMusicStream(as.music)
}

func (as *AudioSystem) GetCurrentTime() float32 {
	return rl.GetMusicTimePlayed(as.music)
}

func (as *AudioSystem) GetBeats() []Beat {
	return as.analyzer.beats
}

func (as *AudioSystem) Close() {
	rl.UnloadMusicStream(as.music)
	rl.CloseAudioDevice()
}

// ============================================================================
// OBSTACLE SYSTEM
// ============================================================================

type Obstacle struct {
	X         float32
	Height    float32 // How high the red bar extends upward
	TimeStamp float32
	BeatIndex int
	IsIntense bool
	BottomY   float32 // Base position on waveform
}

// ============================================================================
// TERRAIN SYSTEM
// ============================================================================

type TerrainPoint struct {
	X         float32
	Y         float32
	TimeStamp float32
	Amplitude float32
}

type Terrain struct {
	points      []TerrainPoint
	obstacles   []Obstacle
	scrollSpeed float32
	baseY       float32
	maxHeight   float32
	minHeight   float32
	analyzer    *AudioAnalyzer
}

func NewTerrain(audioSys *AudioSystem, windowHeight int32) *Terrain {
	terrain := &Terrain{
		points:      []TerrainPoint{},
		obstacles:   []Obstacle{},
		scrollSpeed: 200.0,
		baseY:       float32(windowHeight) - 150,
		maxHeight:   150.0,
		minHeight:   20.0,
		analyzer:    audioSys.analyzer,
	}

	terrain.GenerateFromWaveform()
	terrain.GenerateObstacles(audioSys.GetBeats())

	return terrain
}

func (t *Terrain) GenerateFromWaveform() {
	t.points = []TerrainPoint{}

	for i, amplitude := range t.analyzer.samples {
		timeSeconds := float32(i) * 0.01
		xPos := timeSeconds * t.scrollSpeed

		heightRange := t.maxHeight - t.minHeight
		height := t.minHeight + (amplitude * heightRange * 300)

		if height > t.maxHeight {
			height = t.maxHeight
		}
		if height < t.minHeight {
			height = t.minHeight
		}

		point := TerrainPoint{
			X:         xPos,
			Y:         t.baseY - height,
			TimeStamp: timeSeconds,
			Amplitude: amplitude,
		}

		t.points = append(t.points, point)
	}
}

// GenerateObstacles creates red bars at beats with intensity pattern
func (t *Terrain) GenerateObstacles(beats []Beat) {
	t.obstacles = []Obstacle{}

	// Intensity pattern: 8 calm, 16 intense, 8 calm (repeating)
	patternLength := 32
	calmDuration := 8
	intenseDuration := 16

	for _, beat := range beats {
		// Determine if this beat should have an obstacle
		positionInPattern := beat.Index % patternLength

		isIntense := false
		if positionInPattern >= calmDuration && positionInPattern < (calmDuration+intenseDuration) {
			isIntense = true
		}

		// During calm sections, only spawn obstacles occasionally (30% chance)
		if !isIntense && (beat.Index%10) < 7 {
			continue // Skip this beat
		}

		// Get terrain height at this beat position
		terrainY := t.GetHeightAtTime(beat.TimeSeconds)

		// Obstacle height varies based on intensity and beat strength
		baseHeight := float32(60)
		if isIntense {
			// Intense sections have taller obstacles
			baseHeight = 60 + (beat.Strength * 40) // 60-100 height
		} else {
			// Calm sections have shorter obstacles
			baseHeight = 40 + (beat.Strength * 20) // 40-60 height
		}

		obstacle := Obstacle{
			X:         beat.TimeSeconds * t.scrollSpeed,
			Height:    baseHeight,
			TimeStamp: beat.TimeSeconds,
			BeatIndex: beat.Index,
			IsIntense: isIntense,
			BottomY:   terrainY,
		}

		t.obstacles = append(t.obstacles, obstacle)
	}

	fmt.Printf("Generated %d obstacles from %d beats\n", len(t.obstacles), len(beats))
}

func (t *Terrain) Draw(currentTime float32, windowWidth int32) {
	if len(t.points) < 2 {
		return
	}

	offset := currentTime * t.scrollSpeed
	playerX := float32(windowWidth) / 3

	// Draw waveform (blue wavy line)
	for i := 0; i < len(t.points)-1; i++ {
		p1 := t.points[i]
		p2 := t.points[i+1]

		x1 := playerX + (p1.X - offset)
		y1 := p1.Y
		x2 := playerX + (p2.X - offset)
		y2 := p2.Y

		if x2 >= -50 && x1 <= float32(windowWidth)+50 {
			// Blue waveform
			rl.DrawLineEx(
				rl.Vector2{X: x1, Y: y1},
				rl.Vector2{X: x2, Y: y2},
				3.0,
				rl.SkyBlue,
			)
		}
	}

	// Draw obstacles (red bars)
	for _, obs := range t.obstacles {
		x := playerX + (obs.X - offset)

		if x < -50 || x > float32(windowWidth)+50 {
			continue
		}

		// Red bar extends upward from terrain
		barWidth := float32(15)
		topY := obs.BottomY - obs.Height

		// Draw obstacle with glow effect
		if obs.IsIntense {
			// Intense obstacles glow more
			rl.DrawRectangle(
				int32(x-barWidth/2-3),
				int32(topY-3),
				int32(barWidth+6),
				int32(obs.Height+6),
				rl.Color{R: 100, G: 0, B: 0, A: 100},
			)
		}

		// Main red bar
		obstacleColor := rl.Red
		if obs.IsIntense {
			obstacleColor = rl.Color{R: 255, G: 50, B: 50, A: 255}
		}

		rl.DrawRectangle(
			int32(x-barWidth/2),
			int32(topY),
			int32(barWidth),
			int32(obs.Height),
			obstacleColor,
		)

		// Highlight top edge
		rl.DrawRectangle(
			int32(x-barWidth/2),
			int32(topY),
			int32(barWidth),
			3,
			rl.Color{R: 255, G: 150, B: 150, A: 255},
		)
	}

	// Draw baseline
	rl.DrawLine(0, int32(t.baseY), windowWidth, int32(t.baseY), rl.DarkGray)
}

func (t *Terrain) GetHeightAtTime(currentTime float32) float32 {
	idx := int(currentTime / 0.01)

	if idx < 0 {
		return t.baseY
	}
	if idx >= len(t.points) {
		return t.baseY
	}

	if idx < len(t.points)-1 {
		p1 := t.points[idx]
		p2 := t.points[idx+1]

		interpolation := (currentTime - p1.TimeStamp) / (p2.TimeStamp - p1.TimeStamp)
		return p1.Y + (p2.Y-p1.Y)*interpolation
	}

	return t.points[idx].Y
}

// CheckObstacleCollision checks if player hits any obstacle
func (t *Terrain) CheckObstacleCollision(currentTime float32, playerX, playerY, playerRadius float32, offset float32, windowWidth int32) bool {
	playerScreenX := float32(windowWidth) / 3

	for _, obs := range t.obstacles {
		obsScreenX := playerScreenX + (obs.X - offset)

		// Check if obstacle is near player horizontally
		if math.Abs(float64(obsScreenX-playerX)) < 20 {
			// Check if player is below the top of the obstacle
			topY := obs.BottomY - obs.Height

			if playerY+playerRadius > topY && playerY-playerRadius < obs.BottomY {
				return true // Collision!
			}
		}
	}

	return false
}

// ============================================================================
// GAME STRUCTURES
// ============================================================================

type Game struct {
	paused       bool
	gameOver     bool
	windowWidth  int32
	windowHeight int32
	player       Player
	audio        *AudioSystem
	terrain      *Terrain
	board        Score
	health       int32
	maxHealth    int32
}

type Score struct {
	current int32
}

func (s *Score) draw(health, maxHealth int32) {
	rl.DrawText("SCORE:", 20, 20, 20, rl.Maroon)
	scoreText := fmt.Sprintf("%d", s.current)
	rl.DrawText(scoreText, 110, 20, 20, rl.White)

	// Health bar
	rl.DrawText("HEALTH:", 20, 50, 20, rl.Maroon)
	barWidth := int32(200)
	barHeight := int32(20)
	healthRatio := float32(health) / float32(maxHealth)

	// Background
	rl.DrawRectangle(120, 50, barWidth, barHeight, rl.DarkGray)

	// Health fill
	healthColor := rl.Green
	if healthRatio < 0.3 {
		healthColor = rl.Red
	} else if healthRatio < 0.6 {
		healthColor = rl.Yellow
	}

	rl.DrawRectangle(120, 50, int32(float32(barWidth)*healthRatio), barHeight, healthColor)

	// Border
	rl.DrawRectangleLines(120, 50, barWidth, barHeight, rl.White)
}

type Player struct {
	velocityY  float32
	velocityX  float32
	centerX    float32
	centerY    float32
	radius     float32
	gravity    int32
	isGrounded bool
	canJump    bool
	surfSpeed  float32 // Horizontal movement speed on waveform
}

func (p *Player) update(dt float32, terrainHeight float32) {
	// Surfing on waveform - player sticks to terrain surface
	if p.isGrounded {
		p.centerY = terrainHeight - p.radius
		p.velocityY = 0
	} else {
		// In air - apply gravity
		p.velocityY += float32(p.gravity) * dt
		p.centerY += p.velocityY * dt
	}

	// Horizontal surfing movement (A/D keys)
	if rl.IsKeyDown(rl.KeyA) || rl.IsKeyDown(rl.KeyLeft) {
		p.centerX -= p.surfSpeed * dt
	}
	if rl.IsKeyDown(rl.KeyD) || rl.IsKeyDown(rl.KeyRight) {
		p.centerX += p.surfSpeed * dt
	}

	// Keep player on screen horizontally
	if p.centerX < p.radius {
		p.centerX = p.radius
	}
	if p.centerX > 450 { // Limit to left portion of screen
		p.centerX = 450
	}

	// Check if should be grounded
	playerBottom := p.centerY + p.radius
	if playerBottom >= terrainHeight {
		p.isGrounded = true
		p.canJump = true
		p.centerY = terrainHeight - p.radius
		p.velocityY = 0
	}
}

func (p *Player) draw() {
	rl.DrawCircle(int32(p.centerX), int32(p.centerY), p.radius, rl.Maroon)
	rl.DrawCircle(int32(p.centerX), int32(p.centerY), p.radius/2, rl.White)

	// Draw surf trail effect when moving
	if p.isGrounded && (rl.IsKeyDown(rl.KeyA) || rl.IsKeyDown(rl.KeyD)) {
		rl.DrawCircle(int32(p.centerX), int32(p.centerY+p.radius), p.radius/3,
			rl.Color{R: 135, G: 206, B: 235, A: 150})
	}
}

func NewGame(g *Game) {
	g.init()
}

func (g *Game) init() {
	g.player.velocityY = 0
	g.player.gravity = 1500
	g.player.velocityX = 0
	g.player.radius = 12
	g.player.centerX = float32(g.windowWidth) / 3
	g.player.centerY = 300
	g.player.canJump = true
	g.player.surfSpeed = 250.0
	g.board.current = 0
	g.health = 3
	g.maxHealth = 3
}

func (g *Game) update(dt float32) {
	if g.paused || g.gameOver {
		return
	}

	currentTime := g.audio.GetCurrentTime()
	offset := currentTime * g.terrain.scrollSpeed

	terrainHeight := g.terrain.GetHeightAtTime(currentTime)

	// Update player
	g.player.update(dt, terrainHeight)

	// Jump input
	if rl.IsKeyPressed(rl.KeySpace) && g.player.canJump {
		g.player.velocityY = -500
		g.player.isGrounded = false
		g.player.canJump = false
		g.board.current += 10
	}

	// Variable jump
	if rl.IsKeyReleased(rl.KeySpace) && g.player.velocityY < 0 {
		g.player.velocityY *= 0.5
	}

	// Check obstacle collision
	if g.terrain.CheckObstacleCollision(
		currentTime,
		g.player.centerX,
		g.player.centerY,
		g.player.radius,
		offset,
		g.windowWidth,
	) {
		g.health--
		if g.health <= 0 {
			g.gameOver = true
		}
		// Brief invincibility after hit (remove nearby obstacles)
		// This prevents multiple hits from same obstacle
	}
}

func (g *Game) drawGameOver() {
	// Semi-transparent overlay
	rl.DrawRectangle(0, 0, g.windowWidth, g.windowHeight,
		rl.Color{R: 0, G: 0, B: 0, A: 180})

	// Game Over text
	gameOverText := "GAME OVER"
	textWidth := rl.MeasureText(gameOverText, 60)
	rl.DrawText(gameOverText,
		(g.windowWidth-textWidth)/2,
		g.windowHeight/2-50,
		60,
		rl.Red)

	// Final score
	scoreText := fmt.Sprintf("Final Score: %d", g.board.current)
	scoreWidth := rl.MeasureText(scoreText, 30)
	rl.DrawText(scoreText,
		(g.windowWidth-scoreWidth)/2,
		g.windowHeight/2+30,
		30,
		rl.White)

	// Restart instruction
	restartText := "Press R to Restart"
	restartWidth := rl.MeasureText(restartText, 20)
	rl.DrawText(restartText,
		(g.windowWidth-restartWidth)/2,
		g.windowHeight/2+80,
		20,
		rl.Gray)
}

// ============================================================================
// MAIN
// ============================================================================

func main() {
	const (
		windowWidth  = 1200
		windowHeight = 600
	)

	rl.InitWindow(windowWidth, windowHeight, "Waveform Surfing Rhythm Game")
	defer rl.CloseWindow()
	rl.SetTargetFPS(60)

	audioFile := "song.wav" // <-- YOUR AUDIO FILE

	game := Game{
		paused:       false,
		gameOver:     false,
		windowWidth:  windowWidth,
		windowHeight: windowHeight,
		player:       Player{},
		board:        Score{current: 0},
	}

	NewGame(&game)

	fmt.Println("Analyzing audio file...")
	game.audio = NewAudioSystem(audioFile)
	defer game.audio.Close()

	game.terrain = NewTerrain(game.audio, windowHeight)

	fmt.Printf("Ready! %d obstacles generated\n", len(game.terrain.obstacles))

	game.audio.Play()

	lastTime := float32(rl.GetTime())

	for !rl.WindowShouldClose() {
		currentTime := float32(rl.GetTime())
		dt := currentTime - lastTime
		lastTime = currentTime

		// Restart on R
		if game.gameOver && rl.IsKeyPressed(rl.KeyR) {
			NewGame(&game)
			game.audio.Play()
		}

		// Update
		game.audio.Update()
		game.update(dt)

		// Draw
		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)

		currentSongTime := game.audio.GetCurrentTime()

		// Draw game
		game.terrain.Draw(currentSongTime, windowWidth)
		game.player.draw()
		game.board.draw(game.health, game.maxHealth)

		// Instructions
		rl.DrawText("SURF: A/D or Arrows", 20, windowHeight-60, 18, rl.SkyBlue)
		rl.DrawText("JUMP: SPACE (to clear red bars!)", 20, windowHeight-35, 18, rl.Red)

		// Debug info
		rl.DrawText(fmt.Sprintf("Obstacles: %d", len(game.terrain.obstacles)),
			windowWidth-200, 20, 16, rl.Gray)
		rl.DrawText(fmt.Sprintf("Time: %.2fs", currentSongTime),
			windowWidth-200, 40, 16, rl.Gray)

		if game.gameOver {
			game.drawGameOver()
		}

		rl.EndDrawing()
	}
}
