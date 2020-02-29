_This doc was automatically created by Valet 0.4.3-4-gc835eeb from the workflow defined in workflow.yaml. To deploy the demo, you can use `valet ensure -f workflow.yaml` from this directory, or execute the steps manually. Do not modify this file directly, it will be overwritten the next time the docs are generated._

# Using SNI to route to specific domains with TLS

In this workflow, we'll set up a simple application that showcases how Gloo can be used to establish TLS between the gateway proxy and an upstream service.


This workflow assumes you already have a Kubernetes cluster, and you've installed Gloo Enterprise to the gloo-system namespace.


## Deploy the example application



 



```
kubectl apply -f sample-app.yaml
```

 



```yaml
apiVersion: gateway.solo.io/v1
kind: VirtualService
metadata:
  name: sample-app
  namespace: gloo-system
spec:
  virtualHost:
    domains:
      - '*'
    routes:
      - matchers:
          - exact: /hello
        routeAction:
          single:
            upstream:
              name: default-example-tls-server-8080
              namespace: gloo-system
```

 



## Configure Gloo for TLS with the upstream



 



```
kubectl apply -f tls-secret.yaml
```

 



```yaml
apiVersion: gloo.solo.io/v1
kind: Upstream
metadata:
  labels:
    discovered_by: kubernetesplugin
    service: example-tls-server
  name: default-example-tls-server-8080
  namespace: gloo-system
spec:
  discoveryMetadata: {}
  kube:
    selector:
      app: example-tls-server
    serviceName: example-tls-server
    serviceNamespace: default
    servicePort: 8080
  sslConfig:
    secretRef:
      name: upstream-tls
      namespace: default
```

 



 

