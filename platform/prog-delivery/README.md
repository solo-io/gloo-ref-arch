_This doc was automatically created by Valet 0.4.3-15-g4b36257 from the workflow defined in workflow.yaml. To deploy the demo, you can use `valet ensure -f workflow.yaml` from this directory, or execute the steps manually. Do not modify this file directly, it will be overwritten the next time the docs are generated._

# Progressive delivery with Gloo Upstream Groups

## Deploy the Petclinic Monolith

Let's deploy the petclinic monolith.

 

We can run the following commands to deploy the application to Kubernetes. These yaml files contain the Kubernetes deployment and service definitions for the application.


```
kubectl apply -f https://raw.githubusercontent.com/sololabs/demos/b523571c66057a5591bce22ad896729f1fee662b/petclinic_demo/petclinic.yaml
kubectl apply -f https://raw.githubusercontent.com/sololabs/demos/b523571c66057a5591bce22ad896729f1fee662b/petclinic_demo/petclinic-db.yaml
```

 

Make sure these pods are running by executing `kubectl get pod` and checking the readiness status for the two petclinic pods. It may take a few minutes to download the containers, depending on your connection.


### Create a route in Gloo

Now we can create a gloo virtual service that adds a route to the petclinic application. In this example, we'll use the domain `*` to match on any domain, though we could use a specific domain if the `Host` header is set. Use `kubectl` to apply the following yaml:


```yaml
apiVersion: gateway.solo.io/v1
kind: VirtualService
metadata:
  name: petclinic
  namespace: gloo-system
spec:
  virtualHost:
    domains:
      # We can use the domain "*" to match on any domain, avoiding the need for a host / host header when testing the route.
      - "*"
    routes:
      - matchers:
          - prefix: /
        routeAction:
          single:
            upstream:
              name: default-petclinic-8080
              namespace: gloo-system
```

 

To easily copy a yaml snippet into a command, copy it to the clipboard then run `pbcopy | kubectl apply -f -`.


### Test the route

To test this route, we can open the application in a browser by port-forwarding the gateway proxy, like so:

`kubectl port-forward -n gloo-system deploy/gateway-proxy 8080`

Now you can open the application in your browser by navigating to `localhost:8080`.

We can also invoke a curl command to ensure the service is available.

`curl localhost:8080`

This should return a 200 and the html for the page.


 

 

```
kubectl apply -f https://raw.githubusercontent.com/sololabs/demos/b523571c66057a5591bce22ad896729f1fee662b/petclinic_demo/petclinic-vets.yaml
```

 

```
kubectl apply -f upstream-group-1.yaml
```

 

```
kubectl apply -f vs-2.yaml
```