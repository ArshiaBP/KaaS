# KaaS — Kubernetes as a Service

A REST API, written in Go, that lets you deploy and manage apps on a Kubernetes cluster without touching `kubectl` or Helm directly. You send a JSON request with an image, resources, env vars, and a few flags — KaaS creates the Deployment, Service, ConfigMap, Secret, and (optionally) Ingress for you.

## What it does

- **Deploy any app** — give it an image + tag, replica count, CPU/RAM, env vars (plain or secret), and it spins up a full Deployment/Service/ConfigMap/Secret set.
- **Managed PostgreSQL** — deploy a ready-to-use Postgres instance (StatefulSet) with auto-generated username/password, no config needed.
- **External access** — optionally expose an app through an Ingress at `<app-name>.kaas.local`.
- **Health monitoring** — pass a health-check URL and KaaS schedules a CronJob that pings it periodically and logs success/failure counts to Postgres, queryable via `/health/:app-name`.
- **Deployment status** — check the status and pod info of one deployment or all of them.
- **Metrics out of the box** — request counts, failure counts, and response times (API + DB) exposed at `/metrics` for Prometheus, with Grafana for dashboards.

## Stack

Go, [Echo](https://echo.labstack.com/) for the HTTP layer, [client-go](https://github.com/kubernetes/client-go) to talk to the Kubernetes API, GORM + PostgreSQL for health-check history, Prometheus + Grafana for monitoring. Ships as a Helm chart.

## API

| Method | Endpoint | What it does |
|---|---|---|
| `POST` | `/deploy-unmanaged` | Deploy a custom app (image, replicas, resources, env, ingress, monitoring) |
| `POST` | `/deploy-managed` | Deploy a managed PostgreSQL instance |
| `GET` | `/get-deployment/:app-name` | Status + pods for one deployment |
| `GET` | `/get-all-deployments` | Status + pods for every deployment |
| `GET` | `/health/:app-name` | Health-check history for an app (requires monitoring enabled) |
| `GET` | `/metrics` | Prometheus metrics |

## Running project

KaaS runs inside the cluster (it uses in-cluster config to talk to the Kubernetes API), so it needs to be deployed, not run locally against a remote cluster.

```bash
# build & push the image, then:
minikube tunnel
helm install kaas-api deployment/kaas-api/
helm install nginx-ingress ingress-nginx/ingress-nginx
```

It also needs a Postgres database reachable via `DATABASE_HOST`, `DATABASE_PORT`, `DATABASE_USERNAME`, `DATABASE_PASSWORD`, `DATABASE_DB` env vars, and a service account with permission to manage Deployments, Services, ConfigMaps, Secrets, Ingresses, and CronJobs (see `deployment/kaas-api/cluster-role.yaml`).

For Prometheus/Grafana setup, see `deployment/PROMETHOUES-GRAFANA.MD`.
