_This doc was automatically created by Valet 0.4.3-4-gc835eeb from the workflow defined in workflow.yaml. To deploy the demo, you can use `valet ensure -f workflow.yaml` from this directory, or execute the steps manually. Do not modify this file directly, it will be overwritten the next time the docs are generated._

# Extending a Monlithic Application with Gloo

In this workflow, we'll set up the petclinic application, which is a "monolithic" application that consists of a backend server and a database. Once this application is configured in Gloo, we'll look at how you may deploy a new microservice and connect it to your application. Then we'll replace a buggy part of the application with a new implementation in AWS lambda.


This workflow assumes you already have a Kubernetes cluster, and you've installed Gloo Enterprise to the gloo-system namespace. 


## Deploy the Petclinic Monolith

First, let's deploy the petclinic monolith.

## Extend the monolith with a new microservice

We want to modify the application's "vets" page, to include a new column in the table indicating the location of the vet. We will solve this by deploying a new microservice that serves the updated version of that page, and then add a route to Gloo so that requests for the `/vets` path will be routed to the new microservice.


## Extend the monolith to an AWS lambda

There is a bug in the monolithic application. If we open up the "contact" page, we'll see an error. Like above, we could solve this without modifying the monolith by adding another route to Gloo. In this case, we'll show how you can use Gloo to route to serverless functions. We will deploy a lambda to AWS with a new implementation of the contact page, and wire that to our application.