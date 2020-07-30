To setup, make sure you have a Kubernetes cluster in your context that doesn't have Gloo installed. 
I will also use `glooctl` so have that from the command line. If you have an older version of `glooctl`, 
you may want to upgrade with `glooctl upgrade` to grab the latest, or you can grab a specific release 
from the github releases page. 

## Install Gloo

```
gloo-ref-arch/webinars/gloo-1.4 on  master [?] at ☸️  gke_solo-test-236622_us-east1_gloo-ref-arch
➜ glooctl upgrade
downloading glooctl-darwin-amd64 from release tag v1.4.5
successfully downloaded and installed glooctl version v1.4.5 to /Users/rick/.gloo/bin/glooctl

gloo-ref-arch/webinars/gloo-1.4 on  master [?] at ☸️  gke_solo-test-236622_us-east1_gloo-ref-arch took 18s
➜ glooctl version
Client: {"version":"1.4.5"}
Server: version undefined, could not find any version of gloo running
```

I want to install the latest 1.4.x open source release. Since I have the latest 1.4 CLI, I can just run
`glooctl install gateway`. 

The pods should look healthy and `glooctl` should detect the server version:

```
➜ k get pods -n gloo-system
NAME                             READY   STATUS    RESTARTS   AGE
discovery-6f55547fbf-d8cj5       1/1     Running   0          48s
gateway-7cc866484b-9htcc         1/1     Running   0          48s
gateway-proxy-5759b4b664-mzmsr   1/1     Running   0          48s
gloo-5fc696cf8b-pp9m2            1/1     Running   0          48s

gloo-ref-arch/webinars/gloo-1.4 on  master [?] at ☸️  gke_solo-test-236622_us-east1_gloo-ref-arch
➜ glooctl version
Client: {"version":"1.4.5"}
Server: {"type":"Gateway","kubernetes":{"containers":[{"Tag":"1.4.5","Name":"discovery","Registry":"quay.io/solo-io"},{"Tag":"1.4.5","Name":"gateway","Registry":"quay.io/solo-io"},{"Tag":"1.4.5","Name":"gloo-envoy-wrapper","Registry":"quay.io/solo-io"},{"Tag":"1.4.5","Name":"gloo","Registry":"quay.io/solo-io"}],"namespace":"gloo-system"}}
```

## Deploy our demo application

`k apply -f spelunker`
`k apply -f vs.yaml`

## Make sure our route works 

`curl $(glooctl proxy url)/ -v` 

## Deploy our wasm filter

`k patch -n gloo-system gateway gateway-proxy --type merge --patch "$(cat gateway-patch-2.yaml)"`

Wait for the proxy to be accepted. 

## See the new response header 

`curl $(glooctl proxy url)/ -v` 

## Check out more resources

* Web Assembly Hub: a collection of community-built wasm filters for Envoy
* wasme: our open source tooling to simplify the process of building, distributing, and deploying wasm modules
* Slack: join the slack community, follow the web assembly community notes in Google Docs. 

Betty will be sending out links to all these resources for anyone here today. 
