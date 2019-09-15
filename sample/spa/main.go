package main

import (
	"github.com/xdatk/pisces"
	"github.com/xdatk/pisces/middleware"
	"net/http"
)

func hello(p pisces.Context) error {
	return p.JSON(http.StatusOK, "Hello, World!")
}

func main() {
	p := pisces.New()
	p.Use(middleware.Gzip())
	p.Use(middleware.RequestID())
	p.Use(middleware.Logger())
	p.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Root:  "static",
		HTML5: true,
	}))

	g := p.Group("/api")
	g.GET("/hello", hello)

	_ = p.Start(":1099")
}
