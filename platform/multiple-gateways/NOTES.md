## Notes

Let's say we wish to expose two services, `echo-1` and `echo-2`, with routes on different listener ports. 

First, **add a port** called `http-2` to the `gateway-proxy` service:
`kubectl apply -f gateway-proxy-svc.yaml`

Second, **deploy the services**:
`kubectl apply -f echo-1.yaml echo-2.yaml`

Third, **deploy the routes**: 
`kubectl apply -f vs-1.yaml vs-2.yaml`

At this point, there should be a problem with the proxy, because two virtual services have the same domain on the same port. 

Fourth, **deploy the gateways**:
`kubectl apply -f gateway-1.yaml gateway-2.yaml`

Finally, **test the routes**:
```
gloo-ref-arch/security/auth/opa on  master [✘»!+?] at ☸️  gke_solo-test-236622_us-east1_gloo-ref-arch
➜ curl $(glooctl proxy url --port http)
echo-1
gloo-ref-arch/security/auth/opa on  master at ☸️  gke_solo-test-236622_us-east1_gloo-ref-arch
➜ curl $(glooctl proxy url --port http-2)
echo-2
```