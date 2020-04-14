This repo contains a set of [Valet](https://github.com/solo-io/valet) workflows (and associated resources) to setup and step through Gloo demos, examples, and deployment architectures. 

## How to Use This Repo

This repos contain a set of examples that can be run manually with README's for explanation. The READMEs assume the local directory is the working directory, so clone this repo to simplify executing the steps. See below for a table of contents. 

### Automation

Each of these workflows can be run as a go e2e test, which calls out to a library called `Valet` to automate various 
steps. Simply navigate to the desired directory and run `go test .`. 

## Table of Contents

* [Petclinic](petclinic): Deploy a monolith and expose it to users with Gloo. Deploy a microservice, see the new upstream discovered, and add a route to change the application without touching the monolith. Then add a new upstream for AWS, discover lambdas, and create a route to one to fix a bug in the monolith. 
* Two-phased canary rollout
    * [Part 1](two-phased-canary/part1): Perform a canary rollout in two phases. First, route a small slice of traffic to the new version for correctness testing. Then, use weighted destinations to shift the load to the new version. 
    * [Part 2](two-phased-canary/part2): Like part 1, but now with multiple independent teams. Use route table delegation to break up ownership of the proxy across a central ops team, responsible for the domain, and different dev teams responsible for routes to their service. Use route replacement to ensure one team's mistake doesn't block another team. 
* User Auth and Auditing 
    * [Part 1](user-auth-and-audit/part1) Expose an application securely by integrating with Google as an Identity Provider. Chain OIDC login via Google with additional authorization checks, by writing an OPA check against the JWT. Setup multiple access loggers to record traffic through the proxy. 
    * [Part 2](user-auth-and-audit/part2) Keycloak integration
* Exposing APIs
    * [Part 1](exposing-apis/part1): Explore increasingly complex use cases for rate limiting on APIs exposed through Gloo. Combine rate limiting with the JWT validation filter in Envoy, Gloo's WAF capabilities, and extra JWT authorization in OPA to maximize security in your production environment.  
* Encryption 
    * [Part 1](encryption/part1): Deploy a test server and explore different ways to set up SSL verification and termination. 

