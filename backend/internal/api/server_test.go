package api

import (
	"testing"
)

func TestRouterRegistersRoutes(t *testing.T) {
	server := &Server{}
	router := server.Router()
	routes := router.Routes()
	if len(routes) == 0 {
		t.Fatal("expected routes to be registered")
	}
}

func TestAdminRoutesRegistered(t *testing.T) {
	server := &Server{}
	router := server.Router()
	routes := router.Routes()

	// Verify admin login route exists (under /api/v1)
	adminLoginFound := false
	for _, route := range routes {
		if route.Method == "POST" && route.Path == "/api/v1/auth/admin-login" {
			adminLoginFound = true
			break
		}
	}
	if !adminLoginFound {
		t.Fatal("expected /api/v1/auth/admin-login POST route to be registered")
	}

	// Verify admin verify OTP route exists (under /api/v1)
	adminVerifyFound := false
	for _, route := range routes {
		if route.Method == "POST" && route.Path == "/api/v1/auth/admin-verify-otp" {
			adminVerifyFound = true
			break
		}
	}
	if !adminVerifyFound {
		t.Fatal("expected /api/v1/auth/admin-verify-otp POST route to be registered")
	}

	t.Log("Admin routes (admin-login and admin-verify-otp) are registered correctly")
}

func TestDoorAccessRouteRegistered(t *testing.T) {
	server := &Server{}
	for _, route := range server.Router().Routes() {
		if route.Method == "POST" && route.Path == "/api/v1/biometric/door-access" {
			return
		}
	}
	t.Fatal("expected /api/v1/biometric/door-access POST route to be registered")
}
