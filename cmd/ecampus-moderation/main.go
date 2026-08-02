package main

import "github.com/Milchstrassse/Ecampus-go/internal/app/ecampus/moderationapp"

func main() {
	if err := moderationapp.Run(); err != nil {
		panic(err)
	}
}
