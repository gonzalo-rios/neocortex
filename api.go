package neocortex

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type API struct {
	e          *gin.Engine
	Port       string
	repository Repository
	prefix     string
	analytics  *Analytics
}

func newCortexAPI(repo Repository, analytics *Analytics, prefix, port string) *API {
	return &API{
		e:          gin.Default(),
		Port:       port,
		prefix:     prefix,
		repository: repo,
		analytics:  analytics,
	}
}

func (api *API) registerEndpoints(engine *Engine) {
	corsConf := cors.DefaultConfig()
	corsConf.AddAllowHeaders("Authorization")

	corsConf.AllowAllOrigins = true

	c := cors.New(corsConf)

	api.e.Use(c)

	authJWTMiddleware := getJWTAuth(engine, engine.secret)

	api.e.POST("/login", authJWTMiddleware.LoginHandler)

	api.e.GET("/token_refresh", authJWTMiddleware.RefreshHandler)

	api.e.Use(authJWTMiddleware.MiddlewareFunc())

	r := api.e.Group(api.prefix)
	api.registerDialogsAPI(r)
	api.registerViewsAPI(r)
	api.registerActionsAPI(r)
	api.registerCollectionsAPI(r)
	api.registerSummaryAPI(r)
	api.registerChatsAPI(r)
	api.registerDownloadsAPI(r)
}

func (api *API) Launch(engine *Engine) error {
	api.registerEndpoints(engine)
	return api.e.Run(api.Port)
}
