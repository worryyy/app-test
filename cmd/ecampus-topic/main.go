package main

import "github.com/Milchstrassse/Ecampus-go/internal/app/ecampus/topicapp"

func main() {
	if err := topicapp.Run(); err != nil {
		panic(err)
	}
}
