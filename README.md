This repo contains a set of [Valet](https://github.com/solo-io/valet) workflows (and associated resources) to setup and step through Gloo demos, examples, and deployment architectures. 

## How to Use This Repo

This repos contain a set of examples that can be run manually with README's for explanation. The READMEs assume the local directory is the working directory, so clone this repo to simplify executing the steps. See below for a table of contents. 

### Automation

Using Valet, each of these can be automatically executed or stepped through. By convention, there are three options:

* `valet ensure -f workflow.yaml`: This runs the entire workflow, as an automated test. This leaves the cluster in the state at the end of the workflow, so `valet teardown -f workflow.yaml` can be used to clean up. 
* `valet ensure -f 1-...yaml`: By convention, each workflow is broken down into a few sub-workflows, which can be individually run execute specific parts of the overall workflow. 
* `valet ensure -f workflow.yaml -s`: The `-s|--step` flag tells Valet to step through the workflow and pause at each step, for the most granular execution. 

### Installing Valet

Valet can be installed from source(https://github.com/solo-io/valet) or can be downloaded and added to your PATH manually:
* [OSX](https://storage.googleapis.com/valet-bin/0.5.0/valet-osx-amd64)
* [Linux](https://storage.googleapis.com/valet-bin/0.5.0/valet-linux-amd64)

## Table of Contents

### Standard Demos

* [Petclinic - Migrating to Microservices](demos/extend-monolith)

### Platform Operations

* Decentralized Config Management
  * [Simple Delegation](platform/delegation/simple)
* [Multiple Gateways](platform/multiple-gateways)
* Multiple Proxies
  * [Internal-External](platform/multiple-proxies/internal-external)
* [Namespaced Gloo](platform/namespaced)
* Canary Rollout
  * [Two-Phased Approach with Open Source Gloo](platform/prog-delivery/two-phased-with-os-gloo)
  
### Security Examples

* [Access Logging](security/access-log)
* [Data Loss Prevention](security/dlp)

#### Rate Limiting

* [Basic](security/rate-limit/basic)
* [Multiple Rules and Priority](security/rate-limit/rule-priority)
* [From JWT Claims](security/rate-limit/from-jwt-claims)

#### Auth

* [Basic](security/auth/basic)
* OAuth
  * [Google](security/auth/oauth/google)
* [OPA](security/auth/opa)
  
#### (m)TLS

* Server TLS
  * [Basic](security/tls/server-tls/basic)
  * [SNI](security/tls/server-tls/sni)
* [Upstream TLS](security/tls/upstream-tls)

#### WAF    
  
* [Basic](security/waf/basic)

### Traffic Management

* [GRPC](traffic-management/grpc)
* [Response Transformations](traffic-management/transformations/response)
