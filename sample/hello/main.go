package hello

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
	p.Use(middleware.Logger())
	p.GET("/", hello)
	_ = p.Start(":1099")
}
