package api

import "github.com/gin-gonic/gin"

// registerAttendanceTerminalRoutes wires the dedicated attendance subsystem.
// It is intentionally separate from registerBiometricRoutes and
// registerDoorAccessRoutes so the completed door-lock firmware keeps using the
// same endpoints and authorization rules it always has.
func (s *Server) registerAttendanceTerminalRoutes(api *gin.RouterGroup) {
	admin := api.Group("/attendance-terminals")
	admin.Use(s.RequireAuth(), s.RequirePasswordChangeCompleted(), s.RequireAdmin())
	admin.GET("", s.listAttendanceTerminals)
	admin.GET("/live", s.listAttendanceLive)
	admin.GET("/:id/mappings", s.listAttendanceTerminalMappings)
	admin.POST("/:id/enrollment-sessions", s.createAttendanceEnrollmentSession)
	admin.GET("/:id/enrollment-sessions/:sessionId", s.showAttendanceEnrollmentSession)
	admin.DELETE("/:id/enrollment-sessions/:sessionId", s.cancelAttendanceEnrollmentSession)

	// Device-facing API. Every request is authenticated with the existing
	// per-device API key / short-lived token; device_type must be
	// attendance_terminal for the handlers to accept it.
	device := api.Group("/attendance-terminal")
	device.Use(s.RequireDevice())
	device.POST("/attendance", s.submitAttendanceScan)
	device.POST("/attendance/batch", s.syncAttendanceScans)
	device.POST("/heartbeat", s.attendanceTerminalHeartbeat)
	device.GET("/enrollment-session", s.claimAttendanceEnrollmentSession)
	device.GET("/enrollment-sessions/:uid", s.showAttendanceEnrollmentSessionForDevice)
	device.POST("/enrollment-sessions/:uid/claim", s.claimAttendanceEnrollmentCommand)
	device.POST("/enrollment-sessions/:uid/progress", s.updateAttendanceEnrollmentProgress)
	device.POST("/enrollment-sessions/:uid/complete", s.completeAttendanceEnrollmentSession)
	device.POST("/enrollment-sessions/:uid/fail", s.failAttendanceEnrollmentSession)
	device.POST("/enrollment-sessions/:uid/cleanup", s.acknowledgeAttendanceEnrollmentCleanup)
}
