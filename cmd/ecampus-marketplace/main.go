package main

import "github.com/Milchstrassse/Ecampus-go/internal/app/ecampus/marketplaceapp"

func main() {
	if err := marketplaceapp.Run(); err != nil {
		panic(err)
	}
}
