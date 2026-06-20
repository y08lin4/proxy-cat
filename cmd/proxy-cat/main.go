package main

import (
	"fmt"

	"github.com/y08lin4/proxy-cat/internal/app"
)

func main() {
	application := app.New()

	// TODO(wails): wire application into wails.Run once the Wails dependency and
	// generated frontend assets are added to go.mod and the project tree.
	fmt.Printf("Proxy-Cat backend binding shell initialized: %+v\n", application.GetAppStatus())
}
