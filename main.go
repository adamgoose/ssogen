package main

import (
	"log"

	"github.com/adamgoose/ssogen/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Println(err)
	}
}
