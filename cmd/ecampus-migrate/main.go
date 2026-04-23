package main

import "github.com/Milchstrassse/Ecampus-go/internal/app/ecampus"

func main() {
	if err := ecampus.RunMigrations(); err != nil {
		panic(err)
	}
}
