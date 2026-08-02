package main

import "github.com/Milchstrassse/Ecampus-go/internal/app/ecampus/academicapp"

func main() {
	if err := academicapp.Run(); err != nil {
		panic(err)
	}
}
