package handler

import (
	"net/http"
	"thaiqr-go/internal/pkg/ping"

	"github.com/teera123/gin"
)

type route struct {
	Name        string
	Description string
	Method      string
	Pattern     string
	Endpoint    gin.HandlerFunc
	AuthenLevel int
}

// Routes holds configurations related to API of this project
type Routes struct {
	v1 []route
	v2 []route
}

func (r Routes) InitRoute() http.Handler {
	r.v1 = []route{
		{
			Name:        "basic ping",
			Description: "ping/pong message",
			Method:      http.MethodGet,
			Pattern:     "/hello",
			Endpoint:    ping.Endpoint,
			AuthenLevel: 1,
		},
	}
	ro := gin.New()
	gin.SetMode(gin.ReleaseMode)
	ro.Use(gin.Recovery())
	v2 := ro.Group("/v1")
	for _, e := range r.v1 {
		v2.Handle(e.Method, e.Pattern)
	}
	return ro

}
