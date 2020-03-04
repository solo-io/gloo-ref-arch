_This doc was automatically created by Valet 0.4.3-6-gc0254cf from the workflow defined in workflow.yaml. To deploy the demo, you can use `valet ensure -f workflow.yaml` from this directory, or execute the steps manually. Do not modify this file directly, it will be overwritten the next time the docs are generated._

# Oauth with Google

In this workflow, we'll deploy the petclinic application with Gloo. Then we'll set up oauth with Google as an OIDC provider.


This workflow assumes you already have a Kubernetes cluster, and you've installed Gloo Enterprise to the gloo-system namespace. 


## Deploy the Petclinic Monolith

First, let's deploy the petclinic monolith.

 

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


## Setup oauth

Now we will update the virtual service to require authentication via oauth with Google as the OIDC provider.

### Store the google client secret

In order to authenticate with Google, Gloo needs a client secret ID and value from Google. This can be created in the [google console](https://console.developers.google.com/apis/credentials), and will be associated with one of your GCP projects so that users in that project can authenticate with oauth.
Store this secret value in an environment variable called `CLIENT_SECRET`. Now this secret can be written to Kubernetes with `glooctl create secret --client-secret $CLIENT_SECRET google-oauth`.


### Store the auth config

Now we can create a gloo auth config that references this client secret and provides the rest of the google oauth information. Note that you should replace `<CLIENT_ID>` with the real ID associated with the secret value from the previous step.


```
apiVersion: enterprise.gloo.solo.io/v1
kind: AuthConfig
metadata:
  name: google-oauth
  namespace: gloo-system
spec:
  configs:
    - oauth:
        app_url: http://localhost:8080
        callback_path: /callback
        client_id: <CLIENT_ID>
        client_secret_ref:
          name: google-oauth
          namespace: gloo-system
        issuer_url: https://accounts.google.com
```

To easily copy a yaml snippet into a command, copy it to the clipboard then run `pbcopy | kubectl apply -f -`.
