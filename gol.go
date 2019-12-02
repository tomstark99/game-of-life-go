package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func sendWorld(p golParams, world [][]byte, d distributorChans, turns int){
	d.io.command <- ioOutput
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight) + "-turns-" + strconv.Itoa(turns)}, "x")

	for y := range world{
		for x := range world[y]{
			d.io.outputVal <- world[y][x]
		}
	}
}

func printAlive(p golParams, world [][]byte){
	alive := 0
	for y := 0; y < p.imageHeight; y++{
		for x := 0; x < p.imageWidth; x++{
			if world[y][x] == 0xFF {
				alive++
			}
		}
	}
	fmt.Println("alive cells: ", alive)
}

func isAlive(width, x, y int, world [][]byte) bool {
	x += width
	x %= width
	if world[y][x] == 0 {
		return false
	} else {
		return true
	}
}

func worker(workerHeight, width int, in <-chan byte, out chan<- byte){
	world := make([][]byte, width)
	for i := range world{
		world[i] = make([]byte, width)
	}
	for {
		for y := 0; y < workerHeight; y++ {
			for x := 0; x < width; x++ {
				world[y][x] = <-in
			}
		}
		for y := 1; y < workerHeight-1; y++ {
			for x := 0; x < width; x++ {
				alive := 0
				for i := -1; i <= 1; i++ {
					for j := -1; j <= 1; j++ {
						if (j != 0 || i != 0) && isAlive(width, x+i, y+j, world) {
							alive++
						}
					}
				}
				if alive == 3 || (isAlive(width, x, y, world) && alive == 2) {
					out <- 0xFF
				} else {
					out <- 0
				}
			}
		}
	}
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p golParams, d distributorChans, alive chan []cell) {

	// Create the 2D slice to store the world.
	world := make([][]byte, p.imageHeight)
	for i := range world {
		world[i] = make([]byte, p.imageWidth)
	}

	// Request the io goroutine to read in the image with the given filename.
	d.io.command <- ioInput
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight)}, "x")

	// The io goroutine sends the requested image byte by byte, in rows.
	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			val := <-d.io.inputVal
			if val != 0 {
				fmt.Println("Alive cell at", x, y)
				world[y][x] = val
			}
		}
	}

	ticker := time.NewTicker(2 * time.Second)

	workerHeight := p.imageHeight / p.threads
	in := make([]chan byte, p.threads)
	out := make([]chan byte, p.threads)

	for i := 0; i < p.threads; i++{
		in[i] = make(chan byte, p.imageHeight)
		out[i] = make(chan byte, p.imageHeight)
	}

	for i := 0; i < p.threads; i++{
		go worker(workerHeight+2, p.imageWidth, in[i], out[i])
	}

	// Calculate the new state of Game of Life after the given number of turns.
	loop1:for turns := 0; turns < p.turns; turns++ {
		select {
		case keyValue := <-d.key:
			char := string(keyValue)
			if char == "s" {
				fmt.Println("s pressed")
				go sendWorld(p, world, d, turns)
			}
			if char == "q" {
				fmt.Println("q pressed")
				break loop1
			}
			if char == "p" {
				fmt.Println("p pressed, pausing at turn " + strconv.Itoa(turns))
				fmt.Println("press p to continue")
				loop2:for{
					select {
					case keyValue := <-d.key:
						char := string(keyValue)
						if char == "p" {
							fmt.Println("continuing")
							break loop2
						}
					default:
					}
				}
			}
		case <-ticker.C:
			go printAlive(p, world)
		default:
			for i := 0; i < p.threads; i++{
				for y := 0; y < (workerHeight+2); y++{
					threadHeight := y+(i*workerHeight)-1
					if threadHeight < 0 {
						threadHeight += p.imageHeight
					}
					threadHeight %= p.imageHeight
					for x := 0; x < p.imageWidth; x++{
						in[i] <- world[threadHeight][x]
					}
				}
			}
			for i := 0; i < p.threads; i++{
				for y := 0; y < workerHeight; y++{
					for x := 0; x < p.imageWidth; x++{
						world[y+(i*workerHeight)][x] = <- out[i]
					}
				}
			}
		}
	}

	sendWorld(p, world, d, p.turns)

	// Create an empty slice to store coordinates of cells that are still alive after p.turns are done.
	var finalAlive []cell
	// Go through the world and append the cells that are still alive.
	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			if world[y][x] != 0 {
				finalAlive = append(finalAlive, cell{x: x, y: y})
			}
		}
	}

	// Make sure that the Io has finished any output before exiting.
	d.io.command <- ioCheckIdle
	<-d.io.idle

	// Return the coordinates of cells that are still alive.
	alive <- finalAlive
}
