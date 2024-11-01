package main

import (
	"log"
	"main/backend"
)

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	// Set the directory where your Svelte build files are located
	backend.Server()

}
