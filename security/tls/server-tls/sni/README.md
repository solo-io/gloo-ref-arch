_This doc was automatically created by Valet 0.4.3-4-gc835eeb from the workflow defined in workflow.yaml. To deploy the demo, you can use `valet ensure -f workflow.yaml` from this directory, or execute the steps manually. Do not modify this file directly, it will be overwritten the next time the docs are generated._

# Using SNI to route to specific domains with TLS

In this workflow, we'll set up a simple application that showcases how Gloo can be used to establish TLS between users or clients and the gateway proxy. In this case, SSL is terminated at the proxy and the request is forwarded to the upstream service with standard http. SNI will be used to determine the domain in the request.


This workflow assumes you already have a Kubernetes cluster, and you've installed Gloo Enterprise to the gloo-system namespace.


 



### Deploy the Petstore Application

First, let's deploy the petstore application. We can run the following commands to deploy the application to Kubernetes. These yaml files contain the Kubernetes deployment and service definitions for the application.


```
kubectl apply -f https://raw.githubusercontent.com/solo-io/gloo/v1.2.9/example/petstore/petstore.yaml
```

Make sure these pods are running by executing `kubectl get pod` and checking the readiness status for the two petclinic pods. It may take a few minutes to download the containers, depending on your connection.


### Create a TLS secret with the SNI domain

For demonstration purposes, we will generate a self-signed cert locally using `openssl`:

`openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout tls.key -out tls.crt -subj "/CN=animalstore.example.com"`

Then, we can store the certificate in Kubernetes as a `tls` type secret. This can be created with the following command:

`kubectl create secret tls animal-certs --key tls.key --cert tls.crt --namespace gloo-system`

For convenience, this secret has been saved and can be applied as a manifest:


```
kubectl apply -f tls-sni-secret.yaml
```

### Create a virtual service with an SSL and SNI configuration

We can create a route with TLS by providing an SSL config in the virtual service, and we can help Envoy route to specific domains by configuring SNI.


```yaml
apiVersion: gateway.solo.io/v1
kind: VirtualService
metadata:
  name: animal
  namespace: gloo-system
spec:
  displayName: animal
  sslConfig:
    secretRef:
      name: animal-certs
      namespace: gloo-system
    sniDomains:
      - animalstore.example.com
  virtualHost:
    domains:
      - animalstore.example.com
    routes:
      - matchers:
          - exact: /animals
        options:
          prefixRewrite: /api/pets
        routeAction:
          single:
            upstream:
              name: default-petstore-8080
              namespace: gloo-system
```

To easily copy a yaml snippet into a command, copy it to the clipboard then run `pbcopy | kubectl apply -f -`.


### Test the route

To test this route, we will leverage glooctl to help determine the https port, and issue a curl request to it:

`curl -k -H "animalstore.example.com" $(glooctl proxy url --port https)/sample-route-1`

Note that we used the `port` flag to indicate https. This should return a 200 and the following json:

```
[{"id":1,"name":"Dog","status":"available"},{"id":2,"name":"Cat","status":"pending"}]
```
