package main

import "github.com/Milchstrassse/Ecampus-go/internal/app/ecampus/fileapp"

func main() {
	if err := fileapp.Run(); err != nil {
		panic(err)
	}
}
