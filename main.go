package main

import (
	"github.com/EngoEngine/engo"

	"github.com/SkeleboyStudios/SkeleDoom/scenes"
)

func main() {
	engo.Run(engo.RunOptions{
		Title:         "Skeleboy Studios",
		Width:         640,
		Height:        360,
		ScaleOnResize: true,
	}, &scenes.StartScene{})
}
