package main

import (
	"log"

	"clinician/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
