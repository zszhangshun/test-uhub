package server

import (
	"net/http"
	"test/api"
	uerr "test/pkg/error"

	"github.com/gin-gonic/gin"

	"github.com/go-playground/validator/v10"
	"github.com/golang/glog"
	"gorm.io/gorm"
)

var (
	basePath = "/v1/uhub/uniq"
	validate = validator.New()
)

type RequestCommonParams struct {
	Action    string `form:"action" binding:"required"`
	ChannelID string `form:"uniq_cloud_channel_id"`
}

type Server struct {
	Engine *gin.Engine
}

func NewServer(h *api.Handle) *Server {
	if !glog.V(5) {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	s := &Server{
		Engine: r,
	}
	s.Engine.Use(ValidateParamsCheck())
	s.globalRouter(h.Store.DBClient())
	s.setRoute(h)
	return s
}
func (s *Server) globalRouter(db *gorm.DB) {
	s.Engine.Any(basePath+"/health", func(ctx *gin.Context) {
		dbclient, err := db.DB()
		if err != nil {
			ctx.JSON(http.StatusServiceUnavailable, err.Error())
			return
		}
		if err = dbclient.Ping(); err != nil {
			ctx.JSON(http.StatusServiceUnavailable, err.Error())
			return
		}
		ctx.JSON(http.StatusOK, "check is ok")
		return

	})

}
func (s *Server) setRoute(h *api.Handle) {
	s.Engine.POST("/channel/flush", h.FlushVaule())
	s.Engine.Static("/static", "./static")
	s.Engine.GET("/", h.IndexHtml)
	s.Engine.LoadHTMLGlob("static/*.tmpl")
	s.Engine.GET("/channel", h.ChannelTotal)
	s.Engine.POST("/channel/update/:id", h.UpdateChannelinfo())
	s.Engine.POST("/channel/create", h.CreateNewChannel)
	s.Engine.POST("/channel/delete/:id", h.DeleteChannel)
}

func ValidateParamsCheck() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var params RequestCommonParams

		if err := validate.Struct(params); err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, uerr.New(err.Error()))
			return
		}

		ctx.Set("validatedParams", params)
		ctx.Next()
	}

}
