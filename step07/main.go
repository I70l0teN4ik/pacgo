package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"time"
)

// Player is the player character \o/
type Player struct {
	row int
	col int
}

var player Player

// Ghost is the enemy that chases the player :O
type Ghost struct {
	row int
	col int
}

var ghosts []*Ghost

// Config holds the emoji configuration
type Config struct {
	Player   string `json:"player"`
	Ghost    string `json:"ghost"`
	Wall     string `json:"wall"`
	Dot      string `json:"dot"`
	Pill     string `json:"pill"`
	Death    string `json:"death"`
	Space    string `json:"space"`
	UseEmoji bool   `json:"use_emoji"`
}

var cfg Config

func loadConfig() error {
	f, err := os.Open("config.json")
	if err != nil {
		return err
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		return err
	}

	return nil
}

func loadMaze() error {
	f, err := os.Open("maze01.txt")
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		maze = append(maze, line)
	}

	for row, line := range maze {
		for col, char := range line {
			switch char {
			case 'P':
				player = Player{row, col}
			case 'G':
				ghosts = append(ghosts, &Ghost{row, col})
			case '.':
				numDots++
			}
		}
	}

	return nil
}

var maze []string
var score int
var numDots int
var lives = 1

func clearScreen() {
	fmt.Print("\x1b[2J")
	moveCursor(0, 0)
}

func moveCursor(row, col int) {
	if cfg.UseEmoji {
		fmt.Printf("\x1b[%d;%df", row+1, col*2+1)
	} else {
		fmt.Printf("\x1b[%d;%df", row+1, col+1)
	}
}

func printScreen() {
	clearScreen()
	for _, line := range maze {
		for _, chr := range line {
			switch chr {
			case '#':
				fmt.Print("\x1b[44m" + cfg.Wall + "\x1b[0m")
			case '.':
				fmt.Print(cfg.Dot)
			case 'X':
				fmt.Print(cfg.Pill)
			default:
				fmt.Print(cfg.Space)
			}
		}
		fmt.Println()
	}

	moveCursor(player.row, player.col)
	fmt.Print(cfg.Player)

	for _, g := range ghosts {
		moveCursor(g.row, g.col)
		fmt.Print(cfg.Ghost)
	}

	moveCursor(len(maze)+1, 0)
	fmt.Println("Score:", score, "\tLives:", lives)
}

func readInput() (string, error) {
	buffer := make([]byte, 100)

	cnt, err := os.Stdin.Read(buffer)
	if err != nil {
		return "", err
	}

	if cnt == 1 && buffer[0] == 0x1b {
		return "ESC", nil
	} else if cnt >= 3 {
		if buffer[0] == 0x1b && buffer[1] == '[' {
			switch buffer[2] {
			case 'A':
				return "UP", nil
			case 'B':
				return "DOWN", nil
			case 'C':
				return "RIGHT", nil
			case 'D':
				return "LEFT", nil
			}
		}
	}

	return "", nil
}

func makeMove(oldRow, oldCol int, dir string) (newRow, newCol int) {
	newRow, newCol = oldRow, oldCol

	switch dir {
	case "UP":
		newRow = newRow - 1
		if newRow < 0 {
			newRow = len(maze) - 1
		}
	case "DOWN":
		newRow = newRow + 1
		if newRow == len(maze) {
			newRow = 0
		}
	case "RIGHT":
		newCol = newCol + 1
		if newCol == len(maze[0]) {
			newCol = 0
		}
	case "LEFT":
		newCol = newCol - 1
		if newCol < 0 {
			newCol = len(maze[0]) - 1
		}
	}

	if maze[newRow][newCol] == '#' {
		newRow = oldRow
		newCol = oldCol
	}

	return
}

func movePlayer(dir string) {
	player.row, player.col = makeMove(player.row, player.col, dir)
	switch maze[player.row][player.col] {
	case '.':
		numDots--
		score++
		// Remove dot from the maze
		maze[player.row] = maze[player.row][0:player.col] + " " + maze[player.row][player.col+1:]
	case 'X':
		score += 10
		maze[player.row] = maze[player.row][0:player.col] + " " + maze[player.row][player.col+1:]
	}
}

func drawDirection() string {
	dir := rand.Intn(4)
	move := map[int]string{
		0: "UP",
		1: "DOWN",
		2: "RIGHT",
		3: "LEFT",
	}
	return move[dir]
}

func moveGhosts() {
	for _, g := range ghosts {
		dir := drawDirection()
		g.row, g.col = makeMove(g.row, g.col, dir)
	}
}

func init() {
	cbTerm := exec.Command("stty", "cbreak", "-echo")
	cbTerm.Stdin = os.Stdin

	err := cbTerm.Run()
	if err != nil {
		log.Fatalln("Unable to activate cbreak mode terminal:", err)
	}
}

func cleanup() {
	cookedTerm := exec.Command("stty", "-cbreak", "echo")
	cookedTerm.Stdin = os.Stdin

	err := cookedTerm.Run()
	if err != nil {
		log.Fatalln("Unable to activate cooked mode terminal:", err)
	}
}

func main() {
	// initialize game
	defer cleanup()

	// load resources
	err := loadMaze()
	if err != nil {
		log.Println("Error loading maze:", err)
		return
	}

	err = loadConfig()
	if err != nil {
		log.Println("Error loading configuration:", err)
		return
	}

	// process input (async)
	input := make(chan string)
	go func(ch chan<- string) {
		for {
			input, err := readInput()
			if err != nil {
				log.Print("Error reading input:", err)
				ch <- "ESC"
			}
			ch <- input
		}
	}(input)

	// game loop
	for {
		// process movement
		select {
		case inp := <-input:
			if inp == "ESC" {
				lives = 0
			}
			movePlayer(inp)
		default:
		}

		moveGhosts()

		// process collisions
		for _, g := range ghosts {
			if player.row == g.row && player.col == g.col {
				lives = 0
			}
		}

		// update screen
		printScreen()

		// check game over
		if numDots == 0 || lives == 0 {
			if lives == 0 {
				moveCursor(player.row, player.col)
				fmt.Print(cfg.Death)
				moveCursor(len(maze)+2, 0)
			}
			break
		}

		// repeat
		time.Sleep(200 * time.Millisecond)
	}
}
