package main

import (
	"log"

	"github.com/jwil007/roamctl/roam"
)

func main() {
	err := roam.Autoroam()
	if err != nil {
		log.Fatalf("roam.Autoroam: %v", err)
	}
}
