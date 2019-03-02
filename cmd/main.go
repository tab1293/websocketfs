package main

import (
	"flag"
	"fmt"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/tab1293/websocketfs"
)

var NUM_PARTS int

func main() {
	port := flag.Int("port", 8015, "http port")
	NUM_PARTS = flag.Int("numparts", 1, "http port")
	flag.Parse()

	e := echo.New()
	e.Use(middleware.CORS())

	e.GET("/ws", websocketfs.EchoWebsocketHandler)

	addr := fmt.Sprintf(":%d", *port)
	e.Start(addr)
}
