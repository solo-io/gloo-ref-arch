For this demo, I want to highlight a key change we made to Gloo config to support multiple auth servers. 

I'm starting with an empty cluster using `glooctl` 1.4.5. 

## Install Gloo

`glooctl install gateway enterprise license-key=$LICENSE_KEY`

Ensure pods running. 

## Deploy example app

`k apply -f speklunker.yaml`
`k apply -f vs.yaml`

## Make sure everything is working

`curl $(glooctl proxy url)/`

## Deploy custom auth server

`k apply -f custom-auth.yaml`

## Update Gateway to use new auth server

Look at existing settings. 