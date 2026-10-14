# Acme Search

Our search API runs on Kubernetes. The manifests in `k8s/` define the API deployment, its service and a CronJob that
warms the cache every 15 minutes. Our enterprise customers run Kubernetes clusters too.
