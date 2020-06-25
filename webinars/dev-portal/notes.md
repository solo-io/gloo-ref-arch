glooctl install gateway enterprise --license-key=$LICENSE_KEY --values values.yaml
k apply -f petstore.yaml
k apply -f vs.yaml

curl -H "host: localhost:8080" $(glooctl proxy url)/api/pets

kubectl port-forward -n gloo-system deployment/api-server 8081:8080

k port-forward -n gloo-system deploy/gateway-proxy 8080
