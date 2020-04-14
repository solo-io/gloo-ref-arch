``` 
k create ns keycloak
helm install --namespace keycloak keycloak codecentric/keycloak --values keycloak-values.yaml
kubectl create secret generic realm-secret --from-file ~/realm.json -n keycloak --dry-run -oyaml | k apply -f -
```



