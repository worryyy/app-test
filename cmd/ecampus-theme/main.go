package main

import "github.com/Milchstrassse/Ecampus-go/internal/app/ecampus/themeapp"

func main() {
	if err := themeapp.Run(); err != nil {
		panic(err)
	}
}
