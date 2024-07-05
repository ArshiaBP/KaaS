package main

import (
	"KaaS/api"
	"KaaS/configs"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	prometheus.MustRegister(api.Requests)
	prometheus.MustRegister(api.FailedRequests)
	prometheus.MustRegister(api.FailedDBRequests)
	prometheus.MustRegister(api.ResponseTime)
	prometheus.MustRegister(api.DBResponseTime)
}

func main() {
	configs.CreateClient()
	configs.ConnectToDatabase()
	server := echo.New()
	api.Routes(server)
}
