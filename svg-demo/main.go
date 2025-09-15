package main

import (
	"fmt"
	"os"

	svg "github.com/ajstarks/svgo"
)

func main() {
	width := 500
	height := 500
	canvas := svg.New(os.Stdout)
	canvas.Start(width, height)
	canvas.Circle(width/2, height/2, 100, "fill:none;stroke:black;stroke-width:3")
	canvas.Text(width/2, height/2+5, "SVG Demo", "text-anchor:middle;font-size:20;fill:black")
	canvas.End()
	fmt.Println("SVG generated")
}
