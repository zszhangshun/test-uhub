package server

import (
	"net/http"
	handler "test/handler"

	"github.com/gin-gonic/gin"

	"github.com/golang/glog"
	"gorm.io/gorm"
)

var (
	v1Router = "uhub/v1/"
)

type RequestCommonParams struct {
	Action    string `form:"action" binding:"required"`
	ChannelID string `form:"uniq_cloud_channel_id"`
}

type Server struct {
	Engine *gin.Engine
}

func NewServer(h *handler.Handle) *Server {
	if !glog.V(5) {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	s := &Server{
		Engine: r,
	}
	s.checkHealthRouter(h.Store.DBClient())
	s.loadTemplate()
	s.routerGroup(h)
	return s
}
func (s *Server) routerGroup(h *handler.Handle) {
	v1 := s.Engine.Group(v1Router, h.Authentication)
	v1.POST("/channel/flush", h.FlushVaule())
	//v1.GET("/", h.IndexHtml)
	v1.GET("/", h.ChannelTotal)
	v1.POST("/channel/update/:id", h.ValidateParamsCheck, h.UpdateChannelinfo())
	v1.POST("/channel/create/:id", h.ValidateParamsCheck, h.CreateNewChannel)
	v1.POST("/channel/delete/:id", h.DeleteChannel)

}

// loadTemplate loads static web resources from the ./static directory
// and associates them with the gin.Engine so that they may be served
// when the server is run.
func (s *Server) loadTemplate() {
	s.Engine.LoadHTMLGlob(("./static/*.tmpl"))
	s.Engine.Static("/static", "static")
	defer func() {
		if r := recover(); r != nil {
			msg := "loadTemplate panic recovered: %s"
			glog.Error(msg, r)
			panic(msg)
		}
	}()
}

func (s *Server) checkHealthRouter(db *gorm.DB) {
	s.Engine.Any(v1Router+"/db/health", func(ctx *gin.Context) {
		dbclient, err := db.DB()
		if err != nil {
			ctx.AbortWithError(http.StatusServiceUnavailable, err)
			return
		}
		if err = dbclient.Ping(); err != nil {
			ctx.AbortWithError(http.StatusServiceUnavailable, err)
			return
		}
		ctx.JSON(http.StatusOK, " db is ok")
		return

	})

}
