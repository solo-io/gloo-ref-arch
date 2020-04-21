This repo contains a set of [Valet](https://github.com/solo-io/valet) workflows (and associated resources) to setup and step through Gloo demos, examples, and deployment architectures. 

## How to Use This Repo

This repos contain a set of examples that can be run manually with README's for explanation. The READMEs assume the local directory is the working directory, so clone this repo to simplify executing the steps. See below for a table of contents. 

## Table of Contents

### Introductory demo

* [Petclinic](petclinic): Deploy a monolith and expose it to users with Gloo. Deploy a microservice, see the new upstream discovered, and add a route to change the application without touching the monolith. Then add a new upstream for AWS, discover lambdas, and create a route to one to fix a bug in the monolith. 

### Two-phased canary rollout

A series on how to implement a safe, scalable workflow for canary testing new versions of services in production environments with Gloo. 

* [Part 1](two-phased-canary/part1): Perform a canary rollout in two phases. First, route a small slice of traffic to the new version for correctness testing. Then, use weighted destinations to shift the load to the new version. 
* [Part 2](two-phased-canary/part2): Like part 1, but now with multiple independent teams. Use route table delegation to break up ownership of the proxy across a central ops team, responsible for the domain, and different dev teams responsible for routes to their service. Use route replacement to ensure one team's mistake doesn't block another team. 
* [Part 3](two-phased-canary/part3): Create a Helm chart based on part 2, so that the workflow can be driven by different teams using `helm upgrade` and updating Helm values. 
* [Part 4 (IN PROGRESS)](two-phased-canary/part4): Expand our Helm chart to enable customizing your deployment and routes, and enabling route options like shadowing. 

### User auth and auditing

The start of a series on how you can leverage Gloo as an API gateway for applications that require user login, covering 
authentication with OIDC providers, authorization with open policy agent, auditing with Envoy access logs, and more. 

* [Part 1](user-auth-and-audit/part1) Expose an application securely by integrating with Google as an Identity Provider. Chain OIDC login via Google with additional authorization checks, by writing an OPA check against the JWT. Setup multiple access loggers to record traffic through the proxy. 
* [Part 2](user-auth-and-audit/part2) Keycloak integration

### Exposing APIs

The start of a series on how you can expose APIs with Gloo, including leveraging features like JWT verification, claim-based 
authorization, rate limiting, and Web Application Firewall (WAF). 

* [Part 1](exposing-apis/part1): Explore increasingly complex use cases for rate limiting on APIs exposed through Gloo. Combine rate limiting with the JWT validation filter in Envoy, Gloo's WAF capabilities, and extra JWT authorization in OPA to maximize security in your production environment.  

### Encryption

The start of a series that dives deep into how Gloo helps solve security concerns related to network encryption.  

* [Part 1](encryption/part1): Deploy a test server and explore different ways to set up SSL verification and termination for L7 (http) proxying. 
* [Part 2](encryption/part2): Like part 1, but for L4 (tcp) proxying. 


