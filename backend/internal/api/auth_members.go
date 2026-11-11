package api

import "github.com/gin-gonic/gin"

func (s *Server) registerAuthRoutes(api *gin.RouterGroup)             {}
func (s *Server) registerMemberRoutes(api *gin.RouterGroup)           {}
func (s *Server) registerAdminRoutes(api *gin.RouterGroup)            {}
func (s *Server) registerMemberManagementRoutes(api *gin.RouterGroup) {}
