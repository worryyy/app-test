package main

import "github.com/Milchstrassse/Ecampus-go/internal/app/ecampus/reservationapp"

func main() {
	if err := reservationapp.Run(); err != nil {
		panic(err)
	}
}
