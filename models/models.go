package models

import (
	"gorm.io/gorm"
	"time"
)

type Environment struct {
	Key      string `json:"Key"`
	Value    string `json:"Value"`
	IsSecret bool   `json:"IsSecret"`
}

type EnvironmentConfig struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

type Resource struct {
	CPU string `json:"CPU"`
	RAM string `json:"RAM"`
}

type PodStatus struct {
	Name      string `json:"Name"`
	Phase     string `json:"Phase"`
	HostID    string `json:"HostID"`
	PodIP     string `json:"PodIP"`
	StartTime string `json:"StartTime"`
}

type HealthCheck struct {
	gorm.Model
	AppName      string
	FailureCount int
	SuccessCount int
	LastFailure  time.Time
	LastSuccess  time.Time
}
