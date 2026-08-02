package main

import "github.com/Milchstrassse/Ecampus-go/internal/app/ecampus/schoolapp"

func main() {
	if err := schoolapp.Run(); err != nil {
		panic(err)
	}
}
