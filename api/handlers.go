package api

import (
	"KaaS/configs"
	"KaaS/models"
	"context"
	"fmt"
	"github.com/labstack/echo/v4"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	InternalError       = "Internal server error"
	BadRequest          = "Request body doesn't have correct format"
	ObjectExist         = "Object already exists"
	DeploymentExistence = "Deployment doesn't exist"
)

type CreateObjectRequest struct {
	AppName        string               `json:"AppName"`
	Replicas       int32                `json:"Replicas"`
	ImageAddress   string               `json:"ImageAddress"`
	ImageTag       string               `json:"ImageTag"`
	DomainAddress  string               `json:"DomainAddress"`
	ServicePort    int32                `json:"ServicePort"`
	Resources      models.Resource      `json:"Resources"`
	Envs           []models.Environment `json:"Envs"`
	Monitor        bool                 `json:"Monitor"`
	ExternalAccess bool                 `json:"ExternalAccess"`
}

type ManagedObjectRequest struct {
	Envs           []models.Environment
	ExternalAccess bool
}

type GetDeploymentResponse struct {
	DeploymentName string             `json:"DeploymentName"`
	Replicas       int32              `json:"Replicas"`
	ReadyReplicas  int32              `json:"ReadyReplicas"`
	PodStatuses    []models.PodStatus `json:"PodStatuses"`
}

func DeployUnmanagedObjects(ctx echo.Context) error {
	startTime := time.Now()
	method := ctx.Request().Method
	endpoint := ctx.Request().URL.Path
	Requests.WithLabelValues(method, endpoint).Inc()
	req := new(CreateObjectRequest)
	if err := ctx.Bind(req); err != nil {
		FailedRequests.WithLabelValues(method, endpoint).Inc()
		ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
		return ctx.JSON(http.StatusBadRequest, BadRequest)
	}
	appName := strings.ToLower(req.AppName)
	err := CheckExistence(ctx, appName)
	if err != nil {
		FailedRequests.WithLabelValues(method, endpoint).Inc()
		ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
		return err
	}
	secretData := make(map[string][]byte)
	configMapData := make(map[string]string)
	for _, env := range req.Envs {
		if env.IsSecret {
			secretData[env.Key] = []byte(env.Value)
		} else {
			configMapData[env.Key] = env.Value
		}
	}
	secret := CreateSecret(secretData, appName, false)
	_, err = configs.Client.CoreV1().Secrets("default").Create(context.Background(), secret, metav1.CreateOptions{})
	if err != nil {
		log.Println("secret:", err.Error())
		FailedRequests.WithLabelValues(method, endpoint).Inc()
		ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	configMap := CreateConfigMap(configMapData, appName, false)
	_, err = configs.Client.CoreV1().ConfigMaps("default").Create(context.Background(), configMap, metav1.CreateOptions{})
	if err != nil {
		log.Println("configmap:", err.Error())
		FailedRequests.WithLabelValues(method, endpoint).Inc()
		ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	deployment := CreateDeployment(appName, req.ImageAddress, req.ImageTag, req.Replicas, req.ServicePort, req.Resources, false, req.Monitor)
	_, err = configs.Client.AppsV1().Deployments("default").Create(context.Background(), deployment, metav1.CreateOptions{})
	if err != nil {
		log.Println("deployment:", err.Error())
		FailedRequests.WithLabelValues(method, endpoint).Inc()
		ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	service := CreateService(appName, req.ServicePort, false)
	_, err = configs.Client.CoreV1().Services("default").Create(context.Background(), service, metav1.CreateOptions{})
	if err != nil {
		log.Println("service:", err.Error())
		FailedRequests.WithLabelValues(method, endpoint).Inc()
		ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	if req.Monitor {
		cronJob := CreateCronJob(appName, req.ImageAddress, req.ImageTag, req.ServicePort)
		_, err = configs.Client.BatchV1().CronJobs(v1.NamespaceDefault).Create(context.Background(), cronJob, metav1.CreateOptions{})
		if err != nil {
			log.Println("cronJob:", err.Error())
			FailedRequests.WithLabelValues(method, endpoint).Inc()
			ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
			return ctx.JSON(http.StatusInternalServerError, InternalError)
		}
		go GetJobsLogs(appName, method, endpoint)
	}
	if req.ExternalAccess {
		ingressObject := CreateIngress(appName, req.ServicePort, false)
		_, err = configs.Client.NetworkingV1().Ingresses("default").Create(context.Background(), ingressObject, metav1.CreateOptions{})
		if err != nil {
			log.Println("ingress: ", err.Error())
			FailedRequests.WithLabelValues(method, endpoint).Inc()
			ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
			return ctx.JSON(http.StatusInternalServerError, InternalError)
		}
		message := fmt.Sprintf("for external access, domain address is: %s.kaas.local", req.AppName)
		ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
		return ctx.JSON(http.StatusOK, message)
	} else {
		message := fmt.Sprintf("for internal access, service name is: %s-service", req.AppName)
		ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
		return ctx.JSON(http.StatusOK, message)
	}
}

func DeployManagedObjects(ctx echo.Context) error {
	startTime := time.Now()
	method := ctx.Request().Method
	endpoint := ctx.Request().URL.Path
	Requests.WithLabelValues(method, endpoint).Inc()
	var res struct {
		Username string `json:"Username"`
		Password string `json:"Password"`
		Message  string `json:"Message"`
	}
	req := new(ManagedObjectRequest)
	if err := ctx.Bind(req); err != nil {
		FailedRequests.WithLabelValues(method, endpoint).Inc()
		ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
		return ctx.JSON(http.StatusBadRequest, BadRequest)
	}
	configMapData := make(map[string]string)
	secretData := make(map[string][]byte)
	for _, env := range req.Envs {
		configMapData[env.Key] = env.Value
	}
	code := PostgresExistence(configMapData)
	if code == "" {
		FailedRequests.WithLabelValues(method, endpoint).Inc()
		ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	secretData["POSTGRES_USERNAME"] = []byte(fmt.Sprintf("user-%s", code))
	secretData["POSTGRES_PASSWORD"] = []byte(GeneratePassword(10, true, true, true))
	secret := CreateSecret(secretData, code, true)
	_, err := configs.Client.CoreV1().Secrets("default").Create(context.Background(), secret, metav1.CreateOptions{})
	if err != nil {
		log.Println("secret:", err.Error())
		FailedRequests.WithLabelValues(method, endpoint).Inc()
		ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	configMap := CreateConfigMap(configMapData, code, true)
	_, err = configs.Client.CoreV1().ConfigMaps("default").Create(context.Background(), configMap, metav1.CreateOptions{})
	if err != nil {
		log.Println("config:", err.Error())
		FailedRequests.WithLabelValues(method, endpoint).Inc()
		ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	resource := models.Resource{
		CPU: "500m",
		RAM: "1Gi",
	}
	service := CreateService(code, 5432, true)
	_, err = configs.Client.CoreV1().Services("default").Create(context.Background(), service, metav1.CreateOptions{})
	if err != nil {
		log.Println("service:", err.Error())
		FailedRequests.WithLabelValues(method, endpoint).Inc()
		ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	stateFulSet := CreateStatefulSet(code, "postgres", "13-alpine", 1, 5432, resource)
	_, err = configs.Client.AppsV1().StatefulSets("default").Create(context.Background(), stateFulSet, metav1.CreateOptions{})
	if err != nil {
		log.Println("statefulset:", err.Error())
		FailedRequests.WithLabelValues(method, endpoint).Inc()
		ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	res.Username = string(secretData["DATABASE_USERNAME"])
	res.Password = string(secretData["DATABASE_PASSWORD"])
	if req.ExternalAccess {
		ingressObject := CreateIngress(code, 5432, true)
		_, err = configs.Client.NetworkingV1().Ingresses("default").Create(context.Background(), ingressObject, metav1.CreateOptions{})
		if err != nil {
			log.Println("ingress:", err.Error())
			FailedRequests.WithLabelValues(method, endpoint).Inc()
			ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
			return ctx.JSON(http.StatusInternalServerError, InternalError)
		}
		message := fmt.Sprintf("for external access, domain name is: postgres.%s.kaas.local", code)
		res.Message = message
		ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
		return ctx.JSON(http.StatusOK, res)
	} else {
		message := fmt.Sprintf("for internal access, service name: is postgres-%s-service", code)
		res.Message = message
		ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
		return ctx.JSON(http.StatusOK, res)
	}
}

func GetDeployment(ctx echo.Context) error {
	startTime := time.Now()
	method := ctx.Request().Method
	endpoint := ctx.Request().URL.Path
	Requests.WithLabelValues(method, endpoint).Inc()
	res := GetDeploymentResponse{}
	appName := ctx.Param("app-name")
	deployment, err := configs.Client.AppsV1().Deployments("default").Get(context.Background(), fmt.Sprintf("%s-deployment", appName), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			FailedRequests.WithLabelValues(method, endpoint).Inc()
			ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
			return ctx.JSON(http.StatusNotAcceptable, DeploymentExistence)
		}
	}
	pods, err := configs.Client.CoreV1().Pods("default").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Println("pods list:", err.Error())
		FailedRequests.WithLabelValues(method, endpoint).Inc()
		ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	var filteredPods []*v1.Pod
	for _, pod := range pods.Items {
		if pod.Labels["app"] == appName {
			filteredPods = append(filteredPods, &pod)
		}
	}
	podStatuses := make([]models.PodStatus, len(filteredPods))
	for i, pod := range filteredPods {
		podStatuses[i] = models.PodStatus{
			Name:      pod.Name,
			Phase:     string(pod.Status.Phase),
			HostID:    pod.Status.HostIP,
			PodIP:     pod.Status.PodIP,
			StartTime: pod.Status.StartTime.String(),
		}
	}
	res.DeploymentName = deployment.Name
	res.Replicas = deployment.Status.Replicas
	res.ReadyReplicas = deployment.Status.ReadyReplicas
	res.PodStatuses = podStatuses
	ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
	return ctx.JSON(http.StatusOK, res)
}

func GetAllDeployments(ctx echo.Context) error {
	startTime := time.Now()
	method := ctx.Request().Method
	endpoint := ctx.Request().URL.Path
	Requests.WithLabelValues(method, endpoint).Inc()
	var res []GetDeploymentResponse
	deployments, err := configs.Client.AppsV1().Deployments("default").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Println("deployment:", err.Error())
		FailedRequests.WithLabelValues(method, endpoint).Inc()
		ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	pods, err := configs.Client.CoreV1().Pods("default").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Println("pods list:", err.Error())
		FailedRequests.WithLabelValues(method, endpoint).Inc()
		ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	res = make([]GetDeploymentResponse, len(deployments.Items))
	for i, deployment := range deployments.Items {
		var filteredPods []*v1.Pod
		for _, pod := range pods.Items {
			deploymentName := fmt.Sprintf("%s-deployment", pod.Labels["app"])
			if deploymentName == deployment.Name {
				filteredPods = append(filteredPods, &pod)
			}
		}
		podStatuses := make([]models.PodStatus, len(filteredPods))
		for j, pod := range filteredPods {
			podStatuses[j] = models.PodStatus{
				Name:      pod.Name,
				Phase:     string(pod.Status.Phase),
				HostID:    pod.Status.HostIP,
				PodIP:     pod.Status.PodIP,
				StartTime: pod.Status.StartTime.String(),
			}
		}
		res[i].DeploymentName = deployment.Name
		res[i].Replicas = deployment.Status.Replicas
		res[i].ReadyReplicas = deployment.Status.ReadyReplicas
		res[i].PodStatuses = podStatuses
	}
	ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
	return ctx.JSON(http.StatusOK, res)
}

func HealthCheck(ctx echo.Context) error {
	startTime := time.Now()
	method := ctx.Request().Method
	endpoint := ctx.Request().URL.Path
	Requests.WithLabelValues(method, endpoint).Inc()
	var res struct {
		AppName      string    `json:"AppName"`
		FailureCount int       `json:"FailureCount"`
		SuccessCount int       `json:"SuccessCount"`
		LastFailure  time.Time `json:"LastFailure"`
		LastSuccess  time.Time `json:"LastSuccess"`
		CreatedAt    time.Time `json:"CreatedAt"`
	}
	appName := ctx.Param("app-name")
	DBStartTime := time.Now()
	var record models.HealthCheck
	result := configs.DB.Table("health_check").Where("app_name = ?", appName).Find(&record)
	if result.Error != nil {
		log.Println("db:", result.Error.Error())
		FailedDBRequests.WithLabelValues(method, endpoint).Inc()
		FailedRequests.WithLabelValues(method, endpoint).Inc()
		ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	DBResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(DBStartTime).Seconds())
	res.AppName = appName
	res.FailureCount = record.FailureCount
	res.SuccessCount = record.SuccessCount
	res.LastFailure = record.LastFailure
	res.LastSuccess = record.LastSuccess
	res.CreatedAt = record.CreatedAt
	ResponseTime.WithLabelValues(method, endpoint).Observe(time.Since(startTime).Seconds())
	return ctx.JSON(http.StatusOK, res)
}
