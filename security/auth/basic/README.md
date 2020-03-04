_This doc was automatically created by Valet 0.4.3-7-g78e3ed9 from the workflow defined in workflow.yaml. To deploy the demo, you can use `valet ensure -f workflow.yaml` from this directory, or execute the steps manually. Do not modify this file directly, it will be overwritten the next time the docs are generated._

# Deploying Gloo with basic authentication

In this workflow, we'll set up the petstore application. Then we'll turn on basic authentication on the route, requiring a username and password be provided for the request to pass through Envoy.


This workflow assumes you already have a Kubernetes cluster, and you've installed Gloo Enterprise to the gloo-system namespace.


 



 

We can run the following commands to deploy the application to Kubernetes. These yaml files contain the Kubernetes deployment and service definitions for the application.


```
kubectl apply -f https://raw.githubusercontent.com/solo-io/gloo/v1.2.9/example/petstore/petstore.yaml
```

Make sure these pods are running by executing `kubectl get pod` and checking the readiness status for the two petclinic pods. It may take a few minutes to download the containers, depending on your connection.


### Create a route in Gloo

Now we can create a gloo virtual service that adds a route to the petclinic application. Use `kubectl` to apply the following yaml:


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

 



### Create an auth config

Now we can create an auth config in gloo to represent basic auth. Basic auth requires credentials in the APR format. We can generate a password with: `htpasswd -nbm user password`. This outputs something like `user:$apr1$TYiryv0/$8BvzLUO9IfGPGGsPnAgSu1`, which contains the salted username and password that we'll include in the auth config. Use `kubectl` to apply the following yaml:


```yaml
apiVersion: enterprise.gloo.solo.io/v1
kind: AuthConfig
metadata:
  name: basic-auth
  namespace: gloo-system
spec:
  configs:
    - basicAuth:
        apr:
          users:
            user:
              hashedPassword: 8BvzLUO9IfGPGGsPnAgSu1
              salt: TYiryv0/
        realm: gloo
```

### Update the virtual service to add auth to the routes

Now we can update the virtual service. In this case, we'll add the auth config to all routes on the virtual service.


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
    options:
      extauth:
        configRef:
          # references the example AuthConfig we defined earlier
          name: basic-auth
          namespace: gloo-system
```

### Test the route

If we curl the request as before:

`curl $(glooctl proxy url)/sample-route-1`

This should now return a 401 Unauthorized error, since the request doesn't contain the authorization header.


### Test the route

We can add the authorization header to the request to satisfy the auth check. First, encode the `user:password` with this command:

`echo -n "user:password" | base64`

Now we can issue a curl request with this value as the authorization header:

`curl -H "Authorization: basic dXNlcjpwYXNzd29yZA==" $(glooctl proxy url)/sample-route-1`

This should return a 200 and the following json:
```
[{"id":1,"name":"Dog","status":"available"},{"id":2,"name":"Cat","status":"pending"}]