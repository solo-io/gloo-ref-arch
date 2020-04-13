# Extending a Monlithic Application with Gloo

In this workflow, we'll set up the petclinic application, which is a "monolithic" application that consists of a backend server and a database. 
Once this application is configured in Gloo, we'll look at how you may deploy a new microservice and connect it to your application. 
Then we'll replace a buggy part of the application with a new implementation in AWS lambda.

This workflow assumes you already have a Kubernetes cluster, and you've installed Gloo to the gloo-system namespace.

## Deploy the Petclinic Monolith

Let's deploy the petclinic monolith. 

We can run the following commands to deploy the application to Kubernetes. 
These yaml files contain the Kubernetes deployment and service definitions for the application.

{{%valet 
workflow: workflow.yaml
step: deploy-monolith
%}}

{{%valet 
workflow: workflow.yaml
step: wait-1
%}}

### Create a route in Gloo

Because Gloo is running with **upstream discovery** enabled, and is **watching** the default namespace, Gloo's 
 discovery component will automatically detect the petclinic service we just installed, and it will write an 
 `Upstream` CRD into the `gloo-system` namespace to represent that traffic destination. 

To expose the application through Gloo, we will create a route from Gloo's gateway proxy to the petclinic upstream by 
writing a `VirtualService` CRD, with a route that references the petclinic `Upstream` CRD. 

{{%valet 
workflow: workflow.yaml
step: vs-1
flags:
  - YamlOnly
%}}

Notice that we're referencing the `Upstream` CRD, so we are using the `gloo-system` namespace. Also notice that we're 
using the domain `*` to represent our virtual host. This makes it easier to test, since we can issue http requests 
directly to the IP address and port of the proxy listener, rather than set up DNS or provide a host header. 

You can apply this config with the following command:

{{%valet 
workflow: workflow.yaml
step: vs-1
%}}

Gloo's `gateway` component is watching for changes to `VirtualService` CRDs and should immediately output an updated
`Proxy` CRD. And the `gloo` component should immediately detect the updated `Proxy` CRD, translate it to Envoy 
config, and transmit that configuration over the Envoy `xds` protocol to the `gateway-proxy` component. At this point, 
the new route is active. 

### Test the route

To test the route, we'll issue requests to the `gateway-proxy` service. To get the URL, we need to look up the external 
IP if the service is a `LoadBanlancer` type, or we need to infer the URL potentially with some special logic depending on 
the Kubernetes flavor. For convenience, we'll grab the URL by running the following command:

```
glooctl proxy url
```

Navigate to that address in the browser to see the petclinic monolith. 

## Extend the monolith with a new microservice

If we look at the "vets" tab in the petclinic application, we'll see a two-column table. 

Now we want to modify this page, to include a new column indicating the location of the vet. 
We will solve this by deploying a new microservice that serves the updated version of that page, and then add a route to 
Gloo so that requests for the `/vets` path will be routed to the new microservice. 

We can use this command to deploy the microservice to Kubernetes.

{{%valet 
workflow: workflow.yaml
step: deploy-vets
%}}

{{%valet 
workflow: workflow.yaml
step: wait-2
%}}

### Deploy the new route to Gloo

Like before, as soon as the vets service was deployed, the upstream was discovered and an `Upstream` resource was written.  
Now we can add a route so that requests to the vets page go to this upstream:

{{%valet 
workflow: workflow.yaml
step: vs-2
flags:
  - YamlOnly
%}}

We'll add the new route to the beginning of the routes list, so it matches first. Any request that doesn't match the `/vets` 
URI will continue to be routed to our monolith. 

{{%valet 
workflow: workflow.yaml
step: vs-2
%}}

### Test the new route

Refresh the page and the new column should now be visible in the vets table. All the other pages should continue to 
behave as they did before. 

## Extend the monolith to an AWS lambda

If we open up the "contact" page, we'll see an error. Like above, we can solve this without modifying the monolith 
by adding another route to Gloo. In this case, we'll show how you can use Gloo to route to serverless functions. 
We will deploy a lambda to AWS with a new implementation of the contact page, and wire that to our application.

### Create a secret with your AWS credentials

In order to connect to a lambda, we need to provide AWS credentials to Envoy. 
We'll store those credentials in a kubernetes secret. 
Assuming you have the `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables defined, 
you can use the following command.

{{%valet 
workflow: workflow.yaml
step: aws-creds
%}}

### Create a Gloo upstream for AWS

Now we need to create an upstream in Gloo, representing a routing destination in AWS and referencing the credentials.

{{%valet 
workflow: workflow.yaml
step: upstream-aws
flags:
  - YamlOnly
%}}

We can apply it to the cluster with the following command: 

{{%valet 
workflow: workflow.yaml
step: upstream-aws
%}}

As soon as this is written, the **discovery** component will see the new upstream and run **function discovery** 
to enrich it with specific lambdas that are available for routing. One of those is called `contact-form:3`, and this 
is the lambda with the new contact form. 

### Add a route to a lambda

Now that we have modeled the AWS lambda destination, we can route to it. 
Let's update the virtual service with a new route so that requests to the contact page are now forwarded to a lambda. 
The json response from AWS will contain a field with html; we specify to transform the response, so that html is returned from Envoy.

{{%valet 
workflow: workflow.yaml
step: vs-3
flags:
  - YamlOnly
%}}

We can apply it to the cluster with the following command: 

{{%valet 
workflow: workflow.yaml
step: vs-3
%}}

### Test the new route

Now if we refresh the "Contact" page, we should see an updated contact form. 
In local testing, it can take up to 30 seconds for the route to start working.