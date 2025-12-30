package main

import (
	"fmt"
	"image/color"

	rl "github.com/gen2brain/raylib-go/raylib"
)

//	ball
//	paddle
//	bricks
//
// Create the game entities/objects
// draw the game entities
// update the objects/entities
// destroy them when no longer in use

func main() {
	game := Game{
		paused:       false,
		gameOver:     false,
		windowWidth:  800,
		windowHeight: 600,
		player:       Player{centerX: int32(400), centerY: int32(300), radius: 15},
		ground: Ground{
			posX:   0,
			posY:   550,
			height: 20,
			width:  800,
			color:  rl.Color{},
		},
		brick: Enemy{
			posX:   0,
			posY:   0,
			width:  40,
			height: 20,
		},
		board: Score{
			current: 0,
		},
	}

	//Create game entities
	NewGame(&game)

	n := int(game.windowWidth / game.brick.width)

	rl.InitWindow(game.windowWidth, game.windowHeight, "Hoorah! A window")

	defer rl.CloseWindow()
	rl.SetTargetFPS(60)

	for !rl.WindowShouldClose() {

		//draw game entities
		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)
		game.board.draw()
		game.player.draw()
		game.ground.draw()

		for i := 0; i <= n; i++ {
			//in the draw function we need to increment the x and y axes
			rl.DrawRectangle(game.brick.posX+int32(i), game.brick.posY, game.brick.width, game.brick.height, rl.Blue)

			//game.brick.draw()
		}
		rl.EndDrawing()
		//update entities
		game.player.update(0.1)
		game.update()

		//destroy the entities.
	}
}

type Game struct {
	paused       bool
	gameOver     bool
	windowWidth  int32
	windowHeight int32
	player       Player
	ground       Ground
	brick        Enemy
	board        Score
}
type Score struct {
	current int32
}

func (s *Score) draw() {
	rl.DrawText("SCORE:", 20, 20, 20, rl.Maroon)
	scoreText := fmt.Sprintf("%d", s.current) // Convert int32 to string
	rl.DrawText(scoreText, 100, 20, 20, rl.White)
}

type Enemy struct {
	posX, posY    int32
	width, height int32
}

func (b *Enemy) draw() {
	//draw multiple bricks
	//call this function multiple times while spacing the bricks by x amount in both axes
	rl.DrawRectangle(b.posX, b.posY, b.width, b.height, rl.Blue)
}

// NewGame init game
func NewGame(g *Game) {
	g.init()
}

func (g *Game) init() {
	g.player.velocityY = 100
	g.player.gravity = 1200
	g.player.velocityX = 0
	g.player.radius = 5
	g.player.centerX = 400
	g.player.centerY = 300
	g.board.current = 0
}

// Player draw the Circle
type Player struct {
	velocityY  float32
	velocityX  float32
	centerX    int32
	centerY    int32
	radius     float32
	gravity    int32
	isGrounded bool
	col        color.RGBA
}

func (g *Game) update() {
	playerBottom := float32(g.player.centerY) + g.player.radius

	// Check if player is colliding with ground
	if playerBottom >= float32(g.ground.posY) &&
		float32(g.player.centerX) >= float32(g.ground.posX) &&
		float32(g.player.centerX) <= float32(g.ground.posX)+float32(g.ground.width) {

		// snap player onto ground
		g.player.centerY = int32(float32(g.ground.posY) - g.player.radius)
		g.player.velocityY = 0
		g.player.isGrounded = true
	} else {
		g.player.isGrounded = false
	}

	//movement
	if g.player.isGrounded && rl.IsKeyPressed(rl.KeySpace) {
		g.player.velocityY = -500

		g.player.isGrounded = false
	}
	if g.player.isGrounded && rl.IsKeyPressed(rl.KeyRight) {
		g.player.velocityX = +10
		g.player.centerX = +1
	}
}

func (b *Player) update(dt float32) {

	b.velocityY += float32(b.gravity) * dt //apply velocity every frame
	b.centerY += int32(b.velocityY * dt)   //convert the result to int32
	b.centerX += int32(b.velocityX * dt)
}
func (b *Player) draw() {
	rl.DrawCircle(int32(b.centerX), int32(b.centerY), b.radius, rl.Maroon)
}

//func NewBall(centerX int32, centerY int32, rad float32, col rl.Color) *Ball {
//	return &Ball{centerX: centerX, centerY: centerY, radius: rad, col: rl.Red}
//}

type Ground struct {
	posX, posY int32
	height     int32
	width      int32
	color      rl.Color
}

func (p *Ground) draw() {
	rl.DrawRectangle(p.posX, p.posY, p.width, p.height, rl.White)
}
func (p *Ground) update() {

}

// func NewPaddle(height int32, col rl.Color, width int32) *Ground {
// return &Ground{height: height, width: width, color: col}
// }
