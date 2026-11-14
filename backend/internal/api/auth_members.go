package api

import (
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) registerAuthRoutes(api *gin.RouterGroup) {
	auth := api.Group("/auth")
	auth.POST("/member-login", s.RateLimit("member-login", 20, 5*time.Minute), s.memberLogin)
	auth.POST("/admin-login", s.RateLimit("admin-login", 20, 5*time.Minute), s.adminLogin)
	auth.POST("/admin-verify-otp", s.RateLimit("admin-verify-otp", 20, 5*time.Minute), s.verifyAdminOTP)
	auth.POST("/password-reset/request", s.RateLimit("password-reset-request", 10, 15*time.Minute), s.passwordResetRequestHandler)
	auth.POST("/password-reset/complete", s.RateLimit("password-reset-complete", 20, 15*time.Minute), s.passwordResetComplete)
	auth.DELETE("/logout", s.RequireAuth(), s.logout)
	auth.GET("/me", s.RequireAuth(), s.me)
	auth.PATCH("/change-password", s.RequireAuth(), s.changePassword)
}

func (s *Server) registerMemberRoutes(api *gin.RouterGroup) {
	api.POST("/members/register", s.memberRegister)

	members := api.Group("/members")
	members.Use(s.RequireAuth())
	members.PATCH("/me/profile", s.updateMemberProfile)
	members.PATCH("/me/accept-rules", s.acceptLabRules)
	members.POST("/me/phone-verification/request", s.requestPhoneVerification)
	members.PATCH("/me/phone-verification", s.verifyPhone)
	members.GET("/me", s.RequirePasswordChangeCompleted(), s.showCurrentMember)
}

func (s *Server) registerAdminRoutes(api *gin.RouterGroup) {
	admins := api.Group("/admins")
	admins.Use(s.RequireAuth(), s.RequireAdmin())
	admins.GET("", s.listAdmins)
	admins.GET("/:id", s.showAdmin)
	admins.POST("", s.RequirePrimaryAdmin(), s.createAdmin)
	admins.PATCH("/:id", s.RequirePrimaryAdmin(), s.updateAdmin)
	admins.DELETE("/:id", s.RequirePrimaryAdmin(), s.deleteAdmin)
	admins.POST("/invite", s.RequirePrimaryAdmin(), s.inviteAdmin)
}

func (s *Server) registerMemberManagementRoutes(api *gin.RouterGroup) {
	members := api.Group("/members-management")
	members.Use(s.RequireAuth(), s.RequireAdmin())
	members.GET("", s.listMembers)
	members.POST("", s.createMember)
	members.GET("/:id", s.showMember)
	members.PATCH("/:id/status", s.updateMemberStatus)
	members.PATCH("/:id/_fields", s.updateMemberFields)
	members.PATCH("/:id/force-password-reset", s.forceMemberPasswordReset)
	members.POST("/:id/fingerprint-enrollment", s.startMemberFingerprintEnrollment)
	members.GET("/:id/fingerprint-enrollment", s.memberFingerprintEnrollmentStatus)
	members.GET("/:id/fingerprint-enrollment/readers", s.memberEnrollmentReaders)
	members.DELETE("/:id/fingerprint-enrollment", s.cancelMemberFingerprintEnrollment)
	members.DELETE("/:id", s.deactivateMember)
}
