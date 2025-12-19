package main

import (
	"log"

	apiengine "github.com/newodahs/accessapi/internal/engine"
)

func main() {
	apiEng := apiengine.NewAPIEngine("", "", true)
	if apiEng == nil {
		log.Fatal("could not create api engine")
	}

	if err := apiEng.Run("0.0.0.0", 8080); err != nil {
		log.Fatalf("error while running api engine %s", err)
	}
}
