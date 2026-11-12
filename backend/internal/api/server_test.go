package api

import "testing"

func TestRouterRegistersRoutes(t *testing.T) {
	server := &Server{}
	router := server.Router()
	routes := router.Routes()
	if len(routes) == 0 {
		t.Fatal("expected routes to be registered")
	}
}
