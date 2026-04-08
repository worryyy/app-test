package main

import (
	"github.com/Milchstrassse/Ecampus-go/internal/app/ecampuscrm"
)

func main() {
	if err := ecampuscrm.Run(); err != nil {
		panic(err)
	}
}
