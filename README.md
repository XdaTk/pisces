`pisces` 超轻量级`Go REST Framework`,基于 `labstack` 的 [echo](https://github.com/labstack/echo) ，专门为微服务定制。

## 特性支持

- 移除模板渲染相关功能，仅保留对 `json`，`form`，`file` 的基础支持
- 调整源码接口，方便二次开发
- 移除大量 `middleware` ，尽保留最基础内容
- 维持最小依赖，目前仅依赖 `logrus`

## 示例

```go
package main

import (
	"net/http"
	"github.com/xdatk/pisces"
	"github.com/xdatk/pisces/middleware"
)

func main() {
	p := pisces.New()
	p.Use(middleware.Logger())
	p.GET("/", hello)
	p.Logger.Fatal(p.Start(":1323"))
}

func hello(p pisces.Context) error {
	return p.JSON(http.StatusOK, "Hello, World!")
}

```

## License

[MIT](https://github.com/labstack/echo/blob/master/LICENSE)
