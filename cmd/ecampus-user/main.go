package main

import "github.com/Milchstrassse/Ecampus-go/internal/app/ecampus/userapp"

func main() {
	if err := userapp.Run(); err != nil {
		panic(err)
	}
}
