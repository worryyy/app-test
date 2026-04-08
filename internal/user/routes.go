package user

import "github.com/gin-gonic/gin"


func RegisterProtectedRoutes(api *gin.RouterGroup, handler *Handler) {
	registerProfileRoutes(api, handler)
	registerFollowRoutes(api, handler)
	registerIdentityRoutes(api, handler)
	registerWXRoutes(api, handler)
}

func RegisterPublicRoutes(engine *gin.Engine, handler *Handler) {
	public := engine.Group("/api/user")
	public.POST("/login", handler.Login)
	public.POST("/refresh", handler.RefreshToken)
	public.PUT("/pre_authentication", handler.PreAuth)
}

func registerProfileRoutes(api *gin.RouterGroup, handler *Handler) {
	profile := api.Group("/user")
	profile.GET("", handler.GetCurrent)
	profile.PUT("", handler.Edit)
	profile.GET("/nickname/random", handler.RandomNickname)
	profile.GET("/profile", handler.GetUserProfile)
}

func registerFollowRoutes(api *gin.RouterGroup, handler *Handler) {
	follow := api.Group("/user")
	follow.POST("/follow", handler.Follow)
	follow.DELETE("/follow", handler.Unfollow)
	follow.GET("/stats", handler.GetUserStats)
}

func registerIdentityRoutes(api *gin.RouterGroup, handler *Handler) {
	identity := api.Group("/user/identity")
	identity.POST("/anonymous", handler.CreateAnonymous)
	identity.PUT("/anonymous/nickname", handler.UpdateAnonymousNickname)
	identity.GET("/list", handler.ListIdentity)
	identity.POST("/switch", handler.SwitchIdentity)
}

func registerWXRoutes(api *gin.RouterGroup, handler *Handler) {
	api.POST("/wx/unlimited/wxa_code", handler.UnlimitedWXACode)
}
