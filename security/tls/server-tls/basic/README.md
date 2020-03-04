_This doc was automatically created by Valet 0.4.3-6-gc0254cf from the workflow defined in workflow.yaml. To deploy the demo, you can use `valet ensure -f workflow.yaml` from this directory, or execute the steps manually. Do not modify this file directly, it will be overwritten the next time the docs are generated._

# Using TLS between Client and Gateway Proxy

In this workflow, we'll set up a simple application that showcases how Gloo can be used to establish TLS between users or clients and the gateway proxy. In this case, SSL is terminated at the proxy and the request is forwarded to the upstream service with standard http.


This workflow assumes you already have a Kubernetes cluster, and you've installed Gloo Enterprise to the gloo-system namespace.


## Deploy the Petstore Application

First, let's deploy the petstore application.

 

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

## Setup TLS between the client and the gateway proxy

We need to acquire a certificate, and then add it as a configuration on our virtual service.

 

For demonstration purposes, we will generate a self-signed cert locally using `openssl`:

`openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout tls.key -out tls.crt -subj "/CN=petstore.example.com"`

Then, we can store the certificate in Kubernetes as a `tls` type secret. This can be created with the following command:

`kubectl create secret tls gateway-tls --key tls.key --cert tls.crt --namespace gloo-system`

For convenience, this secret has been saved and can be applied as a manifest:


```
kubectl apply -f tls-secret.yaml
```

### Update the virtual service with an SSL configuration

By default, since no SSL config was provided on the virtual service, the route was bound to the http port. We
can instead expose this via the https port by updating the virtual service with an SSL config that references
the secret we just created:


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
  sslConfig:
    secretRef:
      name: gateway-tls
      namespace: gloo-system
```

### Test the route

To test this route, we will leverage glooctl to help determine the https port, and issue a curl request to it:

`curl -k $(glooctl proxy url --port https)/sample-route-1`

Note that we used the `port` flag to indicate https. This should return a 200 and the following json:

```
[{"id":1,"name":"Dog","status":"available"},{"id":2,"name":"Cat","status":"pending"}]
```

Note that we turned off client-side verification with the `-k` option in curl. We could enable full mTLS by
associating a real domain instead of `*` to this virtual service, updating DNS to map that domain to the proxy
IP address, and issuing the curl request to the actual domain and https port of the proxy. We would also need to
provide a cacert (`tls.crt` here would suffice, though in practice you want the certificates to be generated
by a known certificate authority).
