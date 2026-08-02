package main

import "github.com/Milchstrassse/Ecampus-go/internal/app/ecampus/chatapp"

func main() {
	if err := chatapp.Run(); err != nil {
		panic(err)
	}
}
