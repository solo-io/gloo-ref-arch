_This doc was automatically created by Valet 0.4.3-15-g4b36257 from the workflow defined in workflow.yaml. To deploy the demo, you can use `valet ensure -f workflow.yaml` from this directory, or execute the steps manually. Do not modify this file directly, it will be overwritten the next time the docs are generated._

# Delegating route ownership with Route Tables

In this workflow, we'll set up the petclinic application, which consists of a backend server and a database. Once this application is configured in Gloo, we'll look at how you can utilize **delegation** to enable teams to manage their own routes while an admin can manage the overall domain.


This workflow assumes you already have a Kubernetes cluster, and you've installed Gloo Enterprise to the gloo-system namespace.


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


## Delegate the route to a route table

Now we'll create this route in a route table. Then we'll change the virtual service route to use a `delegateAction` instead of the `routeAction`. This should behave exactly as before. The advantage to separating the route with a route table is that it provides the ability to utilize Kubernetes RBAC to allow users to own their own route tables, while enabling an admin to manage the virtual service and associated domains.


### Deploy the route table

First, we'll deploy the route table with the same route from our virtual service by applying the following yaml.

```yaml
apiVersion: gateway.solo.io/v1
kind: RouteTable
metadata:
  name: petclinic-routes
  namespace: gloo-system
spec:
  routes:
    - matchers:
        - prefix: /
      routeAction:
        single:
          upstream:
            name: default-petclinic-8080
            namespace: gloo-system
```

### Change the virtual service to use a delegate action

In order to start delegating to the route table we created, we'll udpate the virtual service to use a `delegate action`.

```yaml
apiVersion: gateway.solo.io/v1
kind: VirtualService
metadata:
  name: petclinic
  namespace: gloo-system
spec:
  virtualHost:
    domains:
      - "*"
    routes:
      - matchers:
          - prefix: /
        delegateAction:
          ref:
            name: petclinic-routes
            namespace: gloo-system
```

### Testing the route

 

First, we'll wait until Gloo updates the status on it's `Proxy` CRD to "1" or "ACCEPTED". This indicates that Gloo has validated the configuration and sent the updated configuration to Envoy.


 