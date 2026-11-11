package api

import "github.com/gin-gonic/gin"

func (s *Server) registerEventRoutes(api *gin.RouterGroup)     {}
func (s *Server) registerInventoryRoutes(api *gin.RouterGroup) {}
func (s *Server) registerBorrowingRoutes(api *gin.RouterGroup) {}
func (s *Server) registerReturnRoutes(api *gin.RouterGroup)    {}
