package main

import "github.com/Milchstrassse/Ecampus-go/internal/app/ecampus/notificationapp"

func main() {
	if err := notificationapp.Run(); err != nil {
		panic(err)
	}
}
