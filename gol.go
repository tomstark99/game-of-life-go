package main

import (
	"fmt"
	"strconv"
	"strings"
)

func sendWorld(p golParams, world [][]byte, d distributorChans){
	d.io.command <- ioOutput
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight) + "-" + strconv.Itoa(p.turns)}, "x")

	for y := range world{
		for x := range world[y]{
			d.io.outputVal <- world[y][x]
		}
	}
}

func isAlive(p golParams, x, y int, world[][]byte) bool{
	x += p.imageWidth
	x %= p.imageWidth
	y += p.imageHeight
	y %= p.imageHeight
	if world[y][x] == 0 {
		return false
	} else {
		return true
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

	temp := make([][]byte, p.imageHeight)
	for i := range world {
		temp[i] = make([]byte, p.imageWidth)
		copy(temp[i], world[i])
	}

	// Calculate the new state of Game of Life after the given number of turns.
	for turns := 0; turns < p.turns; turns++ {
		for y := 0; y < p.imageHeight; y++ {
			for x := 0; x < p.imageWidth; x++ {
				alive := 0
				for i := -1; i <= 1; i++ {
					for j := -1; j <= 1; j++ {
						if (j != 0 || i != 0) && isAlive(p, x+i, y+j, temp){
							alive++
						}
					}
				}
				if alive == 3 || (isAlive(p, x, y, temp) && alive == 2){
					world[y][x] = 1
				}else{
					world[y][x] = 0
				}
			}
		}
		for i := range world{
			copy(temp[i], world[i])
		}
	}

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

	sendWorld(p, world, d)

	// Make sure that the Io has finished any output before exiting.
	d.io.command <- ioCheckIdle
	<-d.io.idle

	// Return the coordinates of cells that are still alive.
	alive <- finalAlive
}
