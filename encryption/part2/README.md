# Network encryption with Gloo, part 2

In part 1 we looked at L7 TLS and MTLS support in Gloo. Now let's look at L4 (TCP) encryption. 

## Set up

Let's deploy spelunker:
```
kubectl apply -f spelunker.yaml
```

This needs a secret with a TLS certificate, so let's deploy that too. Envoy and the server will both be involved in the TLS
communication and need to agree on the secret. We'll avoid cross-namespace secret references and instead just save the 
secret in both:
```
kubectl apply -f secret.tls.spelunker.com-server.yaml
kubectl apply -f secret.tls.spelunker.com-envoy.yaml
```

We're using all the same certificates and conventions as part 1, they are just stored here as yaml secrets for convenience. 

## Create upstreams

Now let's create the two upstreams from part 1:

```
kubectl apply -f upstream.http.spelunker.com
kubectl apply -f upstream.tls.spelunker.com
```

Here, we'll use `http` to refer to the server receiving plain http requests on the default port, though Envoy itself
will only be passing through TCP. And then we'll use `tls` for the server receiving https requests on the default ssl 
port, with Envoy again passing through TCP. 

## Update gateways

We're going to re-purpose the standard http and https gateways, to configure Envoy to act as a TCP proxy for both TLS
and non-TLS traffic. 

