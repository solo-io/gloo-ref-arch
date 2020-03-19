_This doc was automatically created by Valet 0.4.3-15-g4b36257 from the workflow defined in workflow.yaml. To deploy the demo, you can use `valet ensure -f workflow.yaml` from this directory, or execute the steps manually. Do not modify this file directly, it will be overwritten the next time the docs are generated._

 

## Deploy the Petstore Application

First, let's deploy the petstore application.

 

We can run the following commands to deploy the application to Kubernetes. These yaml files contain the Kubernetes deployment and service definitions for the application.


```
kubectl apply -f https://raw.githubusercontent.com/solo-io/gloo/v1.2.9/example/petstore/petstore.yaml
```

 

Make sure these pods are running by executing `kubectl get pod` and checking the readiness status for the two petclinic pods. It may take a few minutes to download the containers, depending on your connection.


### Create a route in Gloo

Now we can create a gloo virtual service that adds a route to the petstore application. Use `kubectl` to apply the following yaml:

```yaml
apiVersion: gateway.solo.io/v1
kind: VirtualService
metadata:
  name: petstore
  namespace: gloo-system
spec:
  virtualHost:
    domains:
      - '*'
    routes:
      - matchers:
          - exact: /sample-route-1
        routeAction:
          single:
            upstream:
              name: default-petstore-8080
              namespace: gloo-system
        options:
          prefixRewrite: /api/pets
```

 

To easily copy a yaml snippet into a command, copy it to the clipboard then run `pbcopy | kubectl apply -f -`.

### Test the route

To test this route, we will leverage glooctl to help identify the URL for the proxy, and issue a curl request to it:

`curl $(glooctl proxy url)/sample-route-1`

This should return a 200 and the following json:
```
[{"id":1,"name":"Dog","status":"available"},{"id":2,"name":"Cat","status":"pending"}]
```

 

 

 

```
kubectl apply -f vs-petstore-2.yaml
```

 

 

 

 