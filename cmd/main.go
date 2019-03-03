package main

import (
	"flag"
	"fmt"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/tab1293/websocketfs"
)

var fs *websocketfs.FileSystem

func bindFS() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("fs", fs)
			return next(c)
		}
	}
}

func main() {
	port := flag.Int("port", 8015, "http port")
	flag.Parse()

	fs = websocketfs.NewFileSystem()

	e := echo.New()
	e.Use(middleware.CORS())
	e.Use(bindFS())

	e.GET("/fileAnnounce", websocketfs.FileAnnounceHandler)
	e.GET("/readResponse/*", websocketfs.ReadResponseHandler)

	addr := fmt.Sprintf(":%d", *port)
	e.Start(addr)
}
