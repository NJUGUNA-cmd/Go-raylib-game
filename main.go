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
		ball:         Ball{centerX: int32(400), centerY: int32(300), radius: 15},
		pad: Paddle{
			posX:   300,
			posY:   550,
			height: 20,
			width:  100,
			color:  rl.Color{},
		},
		brick: Brick{
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
		game.ball.draw()
		game.pad.draw()

		for i := 0; i <= n; i++ {
			//in the draw function we need to increment the x and y axes
			rl.DrawRectangle(game.brick.posX+int32(i), game.brick.posY, game.brick.width, game.brick.height, rl.Blue)

			//game.brick.draw()
		}
		rl.EndDrawing()
		//update entities
		game.update()
		game.ball.update(0.1)
		//destroy the entities.
	}
}

type Game struct {
	paused       bool
	gameOver     bool
	windowWidth  int32
	windowHeight int32
	ball         Ball
	pad          Paddle
	brick        Brick
	board        Score
}
type Score struct {
	current int32
}

func (s *Score) draw() {
	rl.DrawText("SCORE:", 20, 20, 30, rl.Maroon)
	scoreText := fmt.Sprintf("%d", s.current) // Convert int32 to string
	rl.DrawText(scoreText, 20, 60, 20, rl.White)
}

type Brick struct {
	posX, posY    int32
	width, height int32
}

func (b *Brick) draw() {
	//draw multiple bricks
	//call this function multiple times while spacing the bricks by x amount in both axes

	rl.DrawRectangle(b.posX, b.posY, b.width, b.height, rl.Blue)
}

// NewGame init game
func NewGame(g *Game) {
	g.init()
	return
}

func (g *Game) init() {
	g.ball.velocityY = 100
	g.ball.velocityX = 60
	g.ball.radius = 5
	g.ball.centerX = 400
	g.ball.centerY = 300
	g.board.current = 0
}

// Ball draw the Circle
type Ball struct {
	velocityY float32
	velocityX float32
	centerX   int32
	centerY   int32
	radius    float32
	col       color.RGBA
}

func (g *Game) update() {

	//here I am trying to find the edge of the ball.
	//Collision for the bottom wall/screen
	if g.ball.centerY-int32(g.ball.radius) >= g.windowHeight {
		//reverse the ball velocity
		g.ball.velocityY *= -1
	}
	//collision for the top wall
	if g.ball.centerY-int32(g.ball.radius) <= 0 {
		g.ball.velocityY *= -1
	}

	//right wall
	if g.ball.centerX-int32(g.ball.radius) >= g.windowWidth {
		g.ball.velocityX *= -1
	}
	if g.ball.centerX-int32(g.ball.radius) <= 0 {
		g.ball.velocityX *= -1
	}

	ballCenter := rl.Vector2{X: float32(g.ball.centerX), Y: float32(g.ball.centerY)}
	paddleRec := rl.Rectangle{
		X:      float32(g.pad.posX),
		Y:      float32(g.pad.posY),
		Width:  float32(g.pad.width),
		Height: float32(g.pad.height),
	}

	if rl.CheckCollisionCircleRec(ballCenter, g.ball.radius, paddleRec) {
		// Collision detected! Reverse ball direction
		g.ball.velocityY *= -1
		// Calculate where on the paddle the ball hit (0 = left edge, 1 = right edge)
		paddleCenter := g.pad.posX + g.pad.width/2
		hitPosition := float32(g.ball.centerX-paddleCenter) / float32(g.pad.width/2)

		// Clamp to [-1, 1] range
		if hitPosition < -1 {
			hitPosition = -1
		}
		if hitPosition > 1 {
			hitPosition = 1
		}

		// Set horizontal velocity based on hit position
		// Multiply by a factor to control the maximum angle (e.g., 200)
		g.ball.velocityX = hitPosition * 200
		g.board.current += 1
	}

	//movement
	if rl.IsKeyDown(rl.KeyLeft) {
		g.pad.posX -= 10
	}
	if rl.IsKeyDown(rl.KeyRight) {
		g.pad.posX += 10
	}

}

func (b *Ball) update(dt float32) {
	b.centerY += int32(b.velocityY * dt) //convert the result to int32
	b.centerX += int32(b.velocityX * dt)
	fmt.Printf("dt is %f and %f velocity.\n", dt, b.velocityY)
}
func (b *Ball) draw() {
	rl.DrawCircle(int32(b.centerX), int32(b.centerY), b.radius, rl.Maroon)
	fmt.Printf("ball.radius = %f\n", b.radius)
}

//func NewBall(centerX int32, centerY int32, rad float32, col rl.Color) *Ball {
//	return &Ball{centerX: centerX, centerY: centerY, radius: rad, col: rl.Red}
//}

type Paddle struct {
	posX, posY int32
	height     int32
	width      int32
	color      rl.Color
}

func (p *Paddle) draw() {
	rl.DrawRectangle(p.posX, p.posY, p.width, p.height, rl.White)
}
func (p *Paddle) update() {

}

func NewPaddle(height int32, col rl.Color, width int32) *Paddle {
	return &Paddle{height: height, width: width, color: col}
}
