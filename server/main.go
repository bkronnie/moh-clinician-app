package main

import (
	"log"

	"github.com/moh/clinician/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
