package service

import (
	"github.com/gin-gonic/gin"

	"github.com/breeew/brew-api/app/core"
	"github.com/breeew/brew-api/app/core/srv"
	v1 "github.com/breeew/brew-api/app/logic/v1"
	"github.com/breeew/brew-api/app/response"
	"github.com/breeew/brew-api/cmd/service/handler"
	"github.com/breeew/brew-api/cmd/service/middleware"
)

func serve(core *core.Core) {
	engine := gin.New()

	core.Plugins.RegisterHTTPEngine(engine)

	httpSrv := &handler.HttpSrv{
		Core:   core,
		Engine: engine,
	}
	setupHttpRouter(httpSrv)

	engine.Run(core.Cfg().Addr)
}

func getUserLimitBuilder(core *core.Core) func(key string) gin.HandlerFunc {
	return func(key string) gin.HandlerFunc {
		return middleware.UseLimit(core, key, func(c *gin.Context) string {
			token, _ := v1.InjectTokenClaim(c)
			return key + ":" + token.User
		})
	}
}

func getSpaceLimitBuilder(core *core.Core) func(key string) gin.HandlerFunc {
	return func(key string) gin.HandlerFunc {
		return middleware.UseLimit(core, key, func(c *gin.Context) string {
			spaceid, _ := c.Params.Get("spaceid")
			return key + ":" + spaceid
		})
	}
}

func setupHttpRouter(s *handler.HttpSrv) {
	userLimit := getUserLimitBuilder(s.Core)
	spaceLimit := getSpaceLimitBuilder(s.Core)
	// auth
	s.Engine.Use(middleware.I18n(), response.NewResponse())
	s.Engine.Use(middleware.Cors)
	apiV1 := s.Engine.Group("/api/v1")
	{
		apiV1.GET("/connect", middleware.AuthorizationFromQuery(s.Core), handler.Websocket(s.Core))
		apiV1.POST("/login/token", middleware.Authorization(s.Core), s.AccessLogin)
		authed := apiV1.Group("")
		authed.Use(middleware.Authorization(s.Core))
		user := authed.Group("/user")
		{
			user.PUT("/profile", userLimit("profile"), s.UpdateUserProfile)
		}

		space := authed.Group("/space")
		{
			space.GET("/list", s.ListUserSpaces)
			space.DELETE("/:spaceid/leave", middleware.VerifySpaceIDPermission(s.Core, srv.PermissionView), s.LeaveSpace)

			space.POST("", userLimit("modify_space"), s.CreateUserSpace)

			space.Use(middleware.VerifySpaceIDPermission(s.Core, srv.PermissionAdmin))
			space.DELETE("/:spaceid", s.DeleteUserSpace)
			space.PUT("/:spaceid", userLimit("modify_space"), s.UpdateSpace)
			space.PUT("/:spaceid/user/role", userLimit("modify_space"), s.SetUserSpaceRole)
			space.GET("/:spaceid/users", s.ListSpaceUsers)

			object := space.Group("/:spaceid/object")
			{
				object.POST("/upload/key", userLimit("upload"), s.GenUploadKey)
			}
		}

		knowledge := authed.Group("/:spaceid/knowledge")
		{
			viewScope := knowledge.Group("")
			{
				viewScope.Use(middleware.VerifySpaceIDPermission(s.Core, srv.PermissionView))
				viewScope.GET("", s.GetKnowledge)
				viewScope.GET("/list", spaceLimit("knowledge_list"), s.ListKnowledge)
				viewScope.POST("/query", spaceLimit("query"), s.Query)
			}

			editScope := knowledge.Group("")
			{
				editScope.Use(middleware.VerifySpaceIDPermission(s.Core, srv.PermissionEdit), spaceLimit("knowledge_modify"))
				editScope.POST("", s.CreateKnowledge)
				editScope.PUT("", s.UpdateKnowledge)
				editScope.DELETE("", s.DeleteKnowledge)
			}
		}

		resource := authed.Group("/:spaceid/resource")
		{
			resource.Use(middleware.VerifySpaceIDPermission(s.Core, srv.PermissionView))
			resource.GET("", s.GetResource)
			resource.GET("/list", s.ListResource)

			resource.Use(spaceLimit("resource"))
			resource.POST("", s.CreateResource)
			resource.PUT("", s.UpdateResource)
			resource.DELETE("/:resourceid", s.DeleteResource)
		}

		chat := authed.Group("/:spaceid/chat")
		{
			chat.Use(middleware.VerifySpaceIDPermission(s.Core, srv.PermissionView))
			chat.POST("", s.CreateChatSession)
			chat.DELETE("/:session", s.DeleteChatSession)
			chat.GET("/list", s.ListChatSession)
			chat.POST("/:session/message/id", s.GenMessageID)
			chat.PUT("/:session/named", spaceLimit("named_session"), s.RenameChatSession)
			chat.GET("/:session/message/:messageid/ext", s.GetChatMessageExt)

			history := chat.Group("/:session/history")
			{
				history.GET("/list", s.GetChatSessionHistory)
			}

			message := chat.Group("/:session/message")
			{
				message.Use(spaceLimit("create_message"))
				message.POST("", s.CreateChatMessage)
			}
		}

		tools := authed.Group("/tools")
		{
			tools.Use(userLimit("tools"))
			tools.GET("/reader", s.ToolsReader)
			tools.POST("/describe/image", s.DescribeImage)
		}
	}
}
