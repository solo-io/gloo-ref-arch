# Setup

This setup requires a Kubernetes cluster with no Gloo installed, and `glooctl` on the command line. For these 
demos a GKE cluster was used. 

## Install

First, upgrade `glooctl` to the latest so we can try out Gloo Enterprise 1.4.0:

`glooctl upgrade`

Next, we can install Gloo Enterprise with the default settings:

`glooctl install gateway enterprise --license-key $LICENSE_KEY`

And we can confirm that Gloo Enterprise v1.4.0 is installed with:

`glooctl version`

## Deploy App

Next, we'll deploy our demo app:

`k apply -f spelunker.yaml`

And a virtual service to expose it via Gloo:

`k apply -f vs.yaml`

And now we should be able to query our service:

`curl $(glooctl proxy url)/ -v`




