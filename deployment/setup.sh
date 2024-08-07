minikube tunnel
helm install kaas-api kaas-api/
helm install nginx-ingress ingress-nginx/ingress-nginx
helm install -f kaas-api/prometheus.yaml prometheus prometheus-community/prometheus
kubectl expose service prometheus-server --type=NodePort --target-port=9090 --name=prometheus-server-ext
helm install grafana grafana/grafana
kubectl expose service grafana --type=NodePort --target-port=3000 --name=grafana-ext
kubectl get secret --namespace default grafana -o jsonpath="{.data.admin-password}" | base64 --decode ; echo
echo "> Cluster Created Successfully! Run the next Commands"
#1. minikube service prometheus-server-ext
#2. minikube service grafana-ext
