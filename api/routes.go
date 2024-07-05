package api

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
)

const port = "8080"

var (
	Requests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "requests_total",
			Help: "Total number of requests.",
		},
		[]string{"method", "endpoint"},
	)
	FailedRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "failed_requests_total",
			Help: "Total number of failed requests.",
		},
		[]string{"method", "endpoint"},
	)
	FailedDBRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "successful_requests_total",
			Help: "Total number of failed database requests.",
		},
		[]string{"method", "endpoint"},
	)
	ResponseTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "response_time",
			Help:    "Response time in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)
	DBResponseTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_response_time",
			Help:    "Database response time in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)
)

func Routes(server *echo.Echo) {
	http.Handle("/metrics", promhttp.Handler())
	server.POST("/deploy-unmanaged", DeployUnmanagedObjects)
	server.POST("/deploy-managed", DeployManagedObjects)
	server.GET("/get-deployment/:app-name", GetDeployment)
	server.GET("/get-all-deployments", GetAllDeployments)
	server.GET("/health/:app-name", HealthCheck)
	if err := server.Start(fmt.Sprintf("0.0.0.0:%s", port)); err != nil {
		log.Fatalf("Server failed to listen: %v", err)
	}
}
