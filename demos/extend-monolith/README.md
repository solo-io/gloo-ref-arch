# Extending a Monlithic Application with Gloo

In this workflow, we'll set up the petclinic application, which is a "monolithic" application that consists of a backend server and a database. Once this application is configured in Gloo, we'll look at how you may deploy a new microservice and connect it to your application. Then we'll replace a buggy part of the application with a new implementation in AWS lambda.


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


## Extend the monolith with a new microservice

We want to modify the application's "vets" page, to include a new column in the table indicating the location of the vet. We will solve this by deploying a new microservice that serves the updated version of that page, and then add a route to Gloo so that requests for the `/vets` path will be routed to the new microservice.


 

We can use this command to deploy the microservice to Kubernetes.


```
kubectl apply -f https://raw.githubusercontent.com/sololabs/demos/b523571c66057a5591bce22ad896729f1fee662b/petclinic_demo/petclinic-vets.yaml
```

### Deploy the new route to Gloo

Now that the vets microservice is deployed, we can route requests to the vets page to this service. For all other requests, we can continue to route the request to the monolith. So we update the `petclinic` virtual service with this new route.


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
          - prefix: /vets
        routeAction:
          single:
            upstream:
              name: default-petclinic-vets-8080
              namespace: gloo-system
      - matchers:
          - prefix: /
        routeAction:
          single:
            upstream:
              name: default-petclinic-8080
              namespace: gloo-system
```

### Test the new route

The gateway proxy is still port-forwarded from before. In the browser, navigate to `localhost:8080/vets.html` and the vets table should now have a new location column.


We can also validate with a curl command that a request to that page contains some of the new data.

`curl -s localhost:8080/vets.html | grep Boston | wc -l`

This should return 1 - one of the vets in the table now includes a location of Boston.


## Extend the monolith to an AWS lambda

There is a bug in the monolithic application. If we open up the "contact" page, we'll see an error. Like above, we could solve this without modifying the monolith by adding another route to Gloo. In this case, we'll show how you can use Gloo to route to serverless functions. We will deploy a lambda to AWS with a new implementation of the contact page, and wire that to our application.

### Create a secret with your AWS credentials

In order to connect to a lambda, we need to provide AWS credentials to Envoy. We'll store those credentials in a kubernetes secret. Assuming you have the `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables defined, you can use the following command.


```
kubectl create secret generic -n  gloo-system aws-creds --from-env=aws_access_key_id=$AWS_ACCESS_KEY_ID --from-env=aws_secret_access_key=$AWS_SECRET_ACCESS_KEY
```

### Create a Gloo upstream for AWS

Now we need to create an upstream in Gloo, representing a routing destination in AWS and referencing the credentials.


```yaml
apiVersion: gloo.solo.io/v1
kind: Upstream
metadata:
  name: aws
  namespace: gloo-system
spec:
  aws:
    region: us-east-1
    secretRef:
      name: aws-creds
      namespace: gloo-system
```

Gloo function discovery will run as soon as this upstream is added, and it should quickly detect lambdas in your account. Those lambdas will be added as discovered functions on the upstream.


### Add a route to a lambda

Now that we have modeled the AWS lambda destination, we can route to it. Let's update the virtual service with a new route so that requests to the contact page are now forwarded to a lambda. In this case, we've deployed a lambda to our AWS account called `contact-form:3`. The json response will contain a field with html; we specify to transform the response, so that html is returned from Envoy.


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
          - prefix: /contact
        routeAction:
          single:
            destinationSpec:
              aws:
                logicalName: contact-form:3
                responseTransformation: true
            upstream:
              name: aws
              namespace: gloo-system
      - matchers:
          - prefix: /vets
        routeAction:
          single:
            upstream:
              name: default-petclinic-vets-8080
              namespace: gloo-system
      - matchers:
          - prefix: /
        routeAction:
          single:
            upstream:
              name: default-petclinic-8080
              namespace: gloo-system
```

### Test the new route

Now that we've defined this route, we can navigate to the contact page and should now see a form: `localhost:8080/contact.html`.

As before, we can also test this with curl:

`curl localhost:8080/contact.html`

This should return a 200 response code.


In local testing, it can take up to 30 seconds for the route to start working.