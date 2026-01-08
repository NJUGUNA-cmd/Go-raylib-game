package main

import (
	"fmt"
	"image/color"
	"math"
	"os"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ============================================================================
// WAVEFORM ANALYSIS & BEAT DETECTION
// ============================================================================

type AudioAnalyzer struct {
	samples          []float32 // amplitude samples over time
	beats            []Beat
	sampleRate       int32
	samplesPerWindow int
}

type Beat struct {
	TimeSeconds float32
	TimeStamp   float32 // Alias for compatibility
	Strength    float32
	Index       int
}

// NewAudioAnalyzer creates an analyzer and extracts beat data from audio file
func NewAudioAnalyzer(filepath string) *AudioAnalyzer {
	// Load audio file
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
		samplesPerWindow: int(wave.SampleRate) / 100, // 10ms windows
	}

	// Extract amplitude envelope
	analyzer.extractAmplitudeEnvelope(wave)

	// Detect beats from amplitude
	analyzer.detectBeats()

	rl.UnloadWave(wave)

	fmt.Printf("Analysis complete: %d samples, %d beats detected\n",
		len(analyzer.samples), len(analyzer.beats))

	return analyzer
}

// extractAmplitudeEnvelope samples the audio amplitude every 10ms
func (aa *AudioAnalyzer) extractAmplitudeEnvelope(wave rl.Wave) {
	data := rl.LoadWaveSamples(wave)
	defer rl.UnloadWaveSamples(data)

	// CRITICAL FIX: FrameCount is the number of frames, not samples
	// Each frame has multiple channels, so total samples = frames * channels
	totalFrames := int(wave.FrameCount)
	channels := int(wave.Channels)
	windowSizeInFrames := aa.samplesPerWindow

	fmt.Printf("Processing: %d frames, %d channels, window=%d frames\n",
		totalFrames, channels, windowSizeInFrames)

	// Process audio in windows (working with frames, not samples)
	for frameIdx := 0; frameIdx < totalFrames; frameIdx += windowSizeInFrames {
		endFrame := frameIdx + windowSizeInFrames
		if endFrame > totalFrames {
			endFrame = totalFrames
		}

		// Calculate RMS (Root Mean Square) for this window
		sumSquares := float32(0)
		count := 0

		// For each frame in the window
		for f := frameIdx; f < endFrame; f++ {
			// For each channel in the frame
			for ch := 0; ch < channels; ch++ {
				sampleIdx := f*channels + ch

				// Bounds check
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

	fmt.Printf("Generated %d amplitude samples\n", len(aa.samples))
}

// detectBeats finds peaks in amplitude (onset detection)
func (aa *AudioAnalyzer) detectBeats() {
	if len(aa.samples) < 3 {
		return
	}

	// Calculate dynamic threshold
	avgAmplitude := float32(0)
	for _, amp := range aa.samples {
		avgAmplitude += amp
	}
	avgAmplitude /= float32(len(aa.samples))

	threshold := avgAmplitude * 1.5 // 1.5x average
	minBeatGap := 10                // minimum 10 windows (~100ms) between beats

	beatIndex := 0
	lastBeatIdx := -minBeatGap

	// Find local maxima that exceed threshold
	for i := 1; i < len(aa.samples)-1; i++ {
		current := aa.samples[i]
		prev := aa.samples[i-1]
		next := aa.samples[i+1]

		// Is this a peak?
		isPeak := current > prev && current > next

		// Is it strong enough and far enough from last beat?
		if isPeak && current > threshold && (i-lastBeatIdx) >= minBeatGap {
			timeSeconds := float32(i) * 0.01 // 10ms per sample

			aa.beats = append(aa.beats, Beat{
				TimeSeconds: timeSeconds,
				TimeStamp:   timeSeconds,            // Set both fields
				Strength:    current / avgAmplitude, // normalized strength
				Index:       beatIndex,
			})

			lastBeatIdx = i
			beatIndex++
		}
	}

	// If very few beats detected, lower threshold and try again
	if len(aa.beats) < 10 {
		fmt.Println("Low beat count, retrying with lower threshold...")
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
					TimeStamp:   timeSeconds, // Set both fields
					Strength:    current / avgAmplitude,
					Index:       beatIndex,
				})

				lastBeatIdx = i
				beatIndex++
			}
		}
	}
}

// GetAmplitudeAtTime returns the amplitude at a specific time
func (aa *AudioAnalyzer) GetAmplitudeAtTime(timeSeconds float32) float32 {
	idx := int(timeSeconds / 0.01)
	if idx < 0 || idx >= len(aa.samples) {
		return 0
	}
	return aa.samples[idx]
}

// ============================================================================
// AUDIO PLAYBACK SYSTEM
// ============================================================================

type AudioSystem struct {
	music    rl.Music
	analyzer *AudioAnalyzer
}

func NewAudioSystem(filepath string) *AudioSystem {
	rl.InitAudioDevice()

	// Analyze the audio file first
	analyzer := NewAudioAnalyzer(filepath)

	// Load for playback
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
// TERRAIN SYSTEM
// ============================================================================

type TerrainPoint struct {
	X         float32
	Y         float32
	TimeStamp float32
	IsBeat    bool
	BeatIndex int
	Amplitude float32
}

type Terrain struct {
	points      []TerrainPoint
	scrollSpeed float32
	baseY       float32
	maxHeight   float32
	minHeight   float32
	analyzer    *AudioAnalyzer
}

func NewTerrain(audioSys *AudioSystem, windowHeight int32) *Terrain {
	terrain := &Terrain{
		points:      []TerrainPoint{},
		scrollSpeed: 200.0,
		baseY:       float32(windowHeight) - 150,
		maxHeight:   150.0,
		minHeight:   20.0,
		analyzer:    audioSys.analyzer,
	}

	// Generate terrain from actual waveform + detected beats
	terrain.GenerateFromWaveform()

	return terrain
}

func (t *Terrain) GenerateFromWaveform() {
	t.points = []TerrainPoint{}

	// Create terrain points from amplitude samples
	for i, amplitude := range t.analyzer.samples {
		timeSeconds := float32(i) * 0.01
		xPos := timeSeconds * t.scrollSpeed

		// Map amplitude to height (with some scaling)
		heightRange := t.maxHeight - t.minHeight
		height := t.minHeight + (amplitude * heightRange * 300) // 300 = scale factor

		// Clamp height
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
			IsBeat:    false,
			BeatIndex: -1,
			Amplitude: amplitude,
		}

		t.points = append(t.points, point)
	}

	// Mark beat points
	for _, beat := range t.analyzer.beats {
		// Find closest terrain point to this beat
		beatIdx := int(beat.TimeSeconds / 0.01)
		if beatIdx >= 0 && beatIdx < len(t.points) {
			t.points[beatIdx].IsBeat = true
			t.points[beatIdx].BeatIndex = beat.Index
		}
	}
}

func (t *Terrain) Draw(currentTime float32, windowWidth int32) {
	if len(t.points) < 2 {
		return
	}

	offset := currentTime * t.scrollSpeed
	playerX := float32(windowWidth) / 3

	// Pulsing animation for beats
	pulse := float32(math.Sin(float64(rl.GetTime() * 5)))
	pulseIntensity := (pulse + 1) / 2 // 0 to 1

	// First pass: Draw base waveform
	for i := 0; i < len(t.points)-1; i++ {
		p1 := t.points[i]
		p2 := t.points[i+1]

		x1 := playerX + (p1.X - offset)
		y1 := p1.Y
		x2 := playerX + (p2.X - offset)
		y2 := p2.Y

		// Only draw if on screen
		if x2 >= -50 && x1 <= float32(windowWidth)+50 {
			// Base terrain line with intensity based on amplitude
			intensity := uint8(p1.Amplitude * 255)
			if intensity < 100 {
				intensity = 100
			}
			lineColor := rl.Color{R: intensity, G: intensity, B: intensity, A: 255}

			rl.DrawLineEx(
				rl.Vector2{X: x1, Y: y1},
				rl.Vector2{X: x2, Y: y2},
				2.0,
				lineColor,
			)
		}
	}

	// Second pass: Draw beat glow segments (outer glow layer)
	for i := 0; i < len(t.points)-1; i++ {
		p1 := t.points[i]

		if !p1.IsBeat {
			continue
		}

		x1 := playerX + (p1.X - offset)

		// Only draw if on screen
		if x1 < -50 || x1 > float32(windowWidth)+50 {
			continue
		}

		// Draw glowing waveform segment around beat
		glowRadius := 15 // points on each side to glow

		for j := -glowRadius; j <= glowRadius; j++ {
			idx := i + j
			if idx < 0 || idx >= len(t.points)-1 {
				continue
			}

			p := t.points[idx]
			pNext := t.points[idx+1]

			x := playerX + (p.X - offset)
			y := p.Y
			xNext := playerX + (pNext.X - offset)
			yNext := pNext.Y

			// Calculate fade based on distance from beat center
			distFromBeat := float32(math.Abs(float64(j))) / float32(glowRadius)
			fade := 1.0 - distFromBeat

			// Three-layer glow effect

			// Outer glow (orange)
			outerAlpha := uint8(fade * 200 * (0.7 + pulseIntensity*0.3))
			outerColor := rl.Color{R: 255, G: 140, B: 0, A: outerAlpha}
			rl.DrawLineEx(
				rl.Vector2{X: x, Y: y},
				rl.Vector2{X: xNext, Y: yNext},
				6.0,
				outerColor,
			)

			// Inner bright layer (yellow)
			if fade > 0.3 {
				innerAlpha := uint8(fade * 255 * (0.6 + pulseIntensity*0.4))
				innerColor := rl.Color{R: 255, G: 220, B: 0, A: innerAlpha}
				rl.DrawLineEx(
					rl.Vector2{X: x, Y: y},
					rl.Vector2{X: xNext, Y: yNext},
					4.0,
					innerColor,
				)
			}

			// Core white highlight (center of beat)
			if fade > 0.7 {
				coreAlpha := uint8(fade * 255 * (0.5 + pulseIntensity*0.5))
				coreColor := rl.Color{R: 255, G: 255, B: 255, A: coreAlpha}
				rl.DrawLineEx(
					rl.Vector2{X: x, Y: y},
					rl.Vector2{X: xNext, Y: yNext},
					2.5,
					coreColor,
				)
			}
		}
	}

	// Third pass: Draw small beat markers at exact beat points
	for i := 0; i < len(t.points)-1; i++ {
		p1 := t.points[i]

		if !p1.IsBeat {
			continue
		}

		x1 := playerX + (p1.X - offset)

		// Only draw if on screen
		if x1 < -50 || x1 > float32(windowWidth)+50 {
			continue
		}

		// Small pulsing marker at exact beat location
		markerPulse := 2.0 + pulseIntensity*2.0
		rl.DrawCircle(int32(x1), int32(p1.Y), markerPulse+2, rl.Orange)
		rl.DrawCircle(int32(x1), int32(p1.Y), markerPulse, rl.Yellow)
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

	// Interpolate between points for smooth collision
	if idx < len(t.points)-1 {
		p1 := t.points[idx]
		p2 := t.points[idx+1]

		t := (currentTime - p1.TimeStamp) / (p2.TimeStamp - p1.TimeStamp)
		return p1.Y + (p2.Y-p1.Y)*t
	}

	return t.points[idx].Y
}

func (t *Terrain) GetNearestBeat(currentTime float32) *Beat {
	var nearest *Beat
	minDist := float32(math.MaxFloat32)

	for i := range t.analyzer.beats {
		beat := &t.analyzer.beats[i]
		dist := float32(math.Abs(float64(beat.TimeSeconds - currentTime)))
		if dist < minDist {
			minDist = dist
			nearest = beat
		}
	}

	return nearest
}

// ============================================================================
// TIMING SYSTEM
// ============================================================================

type TimingResult int

const (
	Miss TimingResult = iota
	Good
	Perfect
)

func (tr TimingResult) String() string {
	switch tr {
	case Perfect:
		return "PERFECT"
	case Good:
		return "GOOD"
	case Miss:
		return "MISS"
	default:
		return "MISS"
	}
}

func JudgeTiming(actionTime, beatTime float32) TimingResult {
	delta := float32(math.Abs(float64(actionTime - beatTime)))

	if delta <= 0.040 {
		return Perfect
	} else if delta <= 0.080 {
		return Good
	}
	return Miss
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
	lastTiming   TimingResult
	timingText   string
	timingTimer  float32
	combo        int32
}

type Score struct {
	current int32
}

func (s *Score) draw(combo int32) {
	rl.DrawText("SCORE:", 20, 20, 20, rl.Maroon)
	scoreText := fmt.Sprintf("%d", s.current)
	rl.DrawText(scoreText, 110, 20, 20, rl.White)

	rl.DrawText("COMBO:", 20, 50, 20, rl.Maroon)
	comboText := fmt.Sprintf("%dx", combo)
	comboColor := rl.White
	if combo > 10 {
		comboColor = rl.Gold
	} else if combo > 5 {
		comboColor = rl.Yellow
	}
	rl.DrawText(comboText, 110, 50, 20, comboColor)
}

type Player struct {
	velocityY    float32
	velocityX    float32
	centerX      int32
	centerY      int32
	radius       float32
	gravity      int32
	isGrounded   bool
	col          color.RGBA
	canJump      bool
	coyoteTime   float32 // grace period after leaving ground
	jumpBuffered bool    // jump input buffering
	bufferTime   float32
}

func (b *Player) update(dt float32) {
	// Apply gravity with terminal velocity
	maxFallSpeed := float32(800)
	b.velocityY += float32(b.gravity) * dt
	if b.velocityY > maxFallSpeed {
		b.velocityY = maxFallSpeed
	}

	b.centerY += int32(b.velocityY * dt)
	b.centerX += int32(b.velocityX * dt)

	// Horizontal momentum damping (air friction)
	if !b.isGrounded {
		b.velocityX *= 0.98 // slight air resistance
	} else {
		b.velocityX *= 0.85 // ground friction
	}

	// Update timers
	if b.coyoteTime > 0 {
		b.coyoteTime -= dt
	}
	if b.bufferTime > 0 {
		b.bufferTime -= dt
	}
}

func (b *Player) draw() {
	rl.DrawCircle(int32(b.centerX), int32(b.centerY), b.radius, rl.Maroon)
	rl.DrawCircle(int32(b.centerX), int32(b.centerY), b.radius/2, rl.White)
}

func NewGame(g *Game) {
	g.init()
}

func (g *Game) init() {
	g.player.velocityY = 0
	g.player.gravity = 1800 // Increased for snappier feel
	g.player.velocityX = 0
	g.player.radius = 10
	g.player.centerX = g.windowWidth / 3
	g.player.centerY = 300
	g.player.canJump = true
	g.player.coyoteTime = 0
	g.player.jumpBuffered = false
	g.player.bufferTime = 0
	g.board.current = 0
	g.combo = 0
}

func (g *Game) update(dt float32) {
	if g.paused {
		return
	}

	currentTime := g.audio.GetCurrentTime()

	terrainHeight := g.terrain.GetHeightAtTime(currentTime)
	playerBottom := float32(g.player.centerY) + g.player.radius

	// Ground collision
	wasGrounded := g.player.isGrounded
	if playerBottom >= terrainHeight {
		g.player.centerY = int32(terrainHeight - g.player.radius)
		g.player.velocityY = 0
		g.player.isGrounded = true
		g.player.canJump = true
		g.player.coyoteTime = 0.15 // 150ms grace period after landing
	} else {
		g.player.isGrounded = false
		// Start coyote time when leaving ground
		if wasGrounded && !g.player.isGrounded {
			g.player.coyoteTime = 0.15
		}
	}

	// Horizontal air control (A/D or Left/Right)
	airControlForce := float32(150)
	maxHorizontalSpeed := float32(300)

	if rl.IsKeyDown(rl.KeyA) || rl.IsKeyDown(rl.KeyLeft) {
		g.player.velocityX -= airControlForce * dt
	}
	if rl.IsKeyDown(rl.KeyD) || rl.IsKeyDown(rl.KeyRight) {
		g.player.velocityX += airControlForce * dt
	}

	// Clamp horizontal speed
	if g.player.velocityX > maxHorizontalSpeed {
		g.player.velocityX = maxHorizontalSpeed
	}
	if g.player.velocityX < -maxHorizontalSpeed {
		g.player.velocityX = -maxHorizontalSpeed
	}

	// Jump input buffering - remember jump input for short time
	if rl.IsKeyPressed(rl.KeySpace) {
		g.player.jumpBuffered = true
		g.player.bufferTime = 0.1 // 100ms buffer window
	}

	// Variable jump height - release space early for shorter jump
	if rl.IsKeyReleased(rl.KeySpace) && g.player.velocityY < 0 {
		g.player.velocityY *= 0.5 // Cut jump short
	}

	// Beat-gated jumping with coyote time and buffering
	canCoyoteJump := g.player.coyoteTime > 0
	wantsToJump := g.player.jumpBuffered && g.player.bufferTime > 0

	if wantsToJump && (g.player.canJump || canCoyoteJump) {
		nearestBeat := g.terrain.GetNearestBeat(currentTime)

		if nearestBeat != nil {
			timing := JudgeTiming(currentTime, nearestBeat.TimeStamp)

			g.lastTiming = timing
			g.timingText = timing.String()
			g.timingTimer = 1.0

			// Different jump strengths and horizontal boosts based on timing
			switch timing {
			case Perfect:
				g.player.velocityY = -600 // Stronger jump
				g.combo++
				g.board.current += 100 * g.combo
				// Perfect timing gives horizontal momentum boost
				if rl.IsKeyDown(rl.KeyD) || rl.IsKeyDown(rl.KeyRight) {
					g.player.velocityX += 100
				} else if rl.IsKeyDown(rl.KeyA) || rl.IsKeyDown(rl.KeyLeft) {
					g.player.velocityX -= 100
				}
			case Good:
				g.player.velocityY = -450
				g.combo++
				g.board.current += 50 * g.combo
				// Good timing gives small boost
				if rl.IsKeyDown(rl.KeyD) || rl.IsKeyDown(rl.KeyRight) {
					g.player.velocityX += 50
				} else if rl.IsKeyDown(rl.KeyA) || rl.IsKeyDown(rl.KeyLeft) {
					g.player.velocityX -= 50
				}
			case Miss:
				g.player.velocityY = -250 // Weak jump
				g.combo = 0
			}

			g.player.isGrounded = false
			g.player.canJump = false
			g.player.coyoteTime = 0
			g.player.jumpBuffered = false
			g.player.bufferTime = 0
		}
	}

	if g.timingTimer > 0 {
		g.timingTimer -= dt
	}
}

func (g *Game) drawTimingFeedback() {
	if g.timingTimer > 0 {
		col := rl.White
		switch g.lastTiming {
		case Perfect:
			col = rl.Gold
		case Good:
			col = rl.Green
		case Miss:
			col = rl.Red
		}

		alpha := uint8(g.timingTimer * 255)
		col.A = alpha

		rl.DrawText(g.timingText, g.windowWidth/2-50, 200, 40, col)
	}
}

// ============================================================================
// MAIN
// ============================================================================

func main() {
	const (
		windowWidth  = 1200
		windowHeight = 600
	)

	rl.InitWindow(windowWidth, windowHeight, "Waveform Rhythm Terrain - Real Beat Detection")
	defer rl.CloseWindow()
	rl.SetTargetFPS(60)

	// CHANGE THIS: Set your audio file path
	audioFile := "song.wav" // <-- PUT YOUR .WAV OR .OGG FILE HERE

	game := Game{
		paused:       false,
		gameOver:     false,
		windowWidth:  windowWidth,
		windowHeight: windowHeight,
		player:       Player{},
		board:        Score{current: 0},
	}

	NewGame(&game)

	// Initialize audio system with REAL beat detection
	fmt.Println("Analyzing audio file...")
	game.audio = NewAudioSystem(audioFile)
	defer game.audio.Close()

	// Generate terrain from actual waveform
	game.terrain = NewTerrain(game.audio, windowHeight)

	fmt.Printf("Ready! %d beats detected\n", len(game.audio.GetBeats()))

	// Start music
	game.audio.Play()

	lastTime := float32(rl.GetTime())

	for !rl.WindowShouldClose() {
		currentTime := float32(rl.GetTime())
		dt := currentTime - lastTime
		lastTime = currentTime

		// Update
		game.audio.Update()
		game.player.update(dt)
		game.update(dt)

		// Draw
		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)

		// Info
		currentSongTime := game.audio.GetCurrentTime()
		rl.DrawText(fmt.Sprintf("Beats: %d", len(game.audio.GetBeats())), 20, 80, 20, rl.Gray)
		rl.DrawText(fmt.Sprintf("Time: %.2fs", currentSongTime), 20, 110, 20, rl.Gray)
		rl.DrawText("Press SPACE on GLOWING BEATS!", windowWidth/2-170, windowHeight-60, 20, rl.Yellow)
		rl.DrawText("A/D or Arrows for air control", windowWidth/2-140, windowHeight-35, 16, rl.Gray)

		// Draw game
		game.terrain.Draw(currentSongTime, windowWidth)
		game.player.draw()
		game.board.draw(game.combo)
		game.drawTimingFeedback()

		rl.EndDrawing()
	}
}
