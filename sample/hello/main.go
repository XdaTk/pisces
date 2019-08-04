package hello

import (
	"net/http"
	"pisces"
	"pisces/middleware"
)

func hello(p pisces.Context) error {
	return p.JSON(http.StatusOK, "Hello, World!")
}

func main() {
	p := pisces.New()
	p.Use(middleware.Gzip())
	p.Use(middleware.Logger())
	p.GET("/", hello)
	_ = p.Start(":1099")
}
