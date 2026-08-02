package main

import "github.com/Milchstrassse/Ecampus-go/internal/app/ecampus/commentapp"

func main() {
	if err := commentapp.Run(); err != nil {
		panic(err)
	}
}
