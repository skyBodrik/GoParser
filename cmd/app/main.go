package main

import (
	"github.com/joho/godotenv"
	"goParser/internal/app"
	"log"
)

func init() {
	// loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func main() {
	app.Run()
}
