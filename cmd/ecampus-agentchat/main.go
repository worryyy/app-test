package main

import "github.com/Milchstrassse/Ecampus-go/internal/app/ecampus/agentchatapp"

func main() {
	if err := agentchatapp.Run(); err != nil {
		panic(err)
	}
}
