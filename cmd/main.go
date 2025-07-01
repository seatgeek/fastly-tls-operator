package main

import (
	"log"
	"time"
)

func main() {
	log.Println("Starting hello logger...")

	for {
		log.Println("hello!")
		time.Sleep(5 * time.Second)
	}
}
