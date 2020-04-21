# Network encryption with Gloo

Let's explore L7 TLS and MTLS support in Gloo. 

## Set up root CA

In order to simulate certificate management, we'll generate a local root CA. We can create the root key and certificate
by running the following two commands:

```
openssl genrsa -des3 -out rootCA.key 4096
openssl req -x509 -new -nodes -key rootCA.key -sha256 -days 1024 -out rootCA.crt
```

The first command will ask for a password, and we'll use `test`. The second command will require that password and 
then ask you to fill out the fields for the root certificate. 

Now, we can start to generate certificates for our service. We can create a certificate signing request for those keys, and then 
we can create the certificate. We'll go ahead and create a certificate for a domain `spelunker.com`, which we'll 
be using throughout this example. 

```
openssl genrsa -out spelunker.com.key 2048
openssl req -new -key spelunker.com.key -out spelunker.com.csr
openssl x509 -req -in spelunker.com.csr -CA rootCA.crt -CAkey rootCA.key -CAcreateserial -out spelunker.com.crt -days 500 -sha256
```

## Deploy spelunker and Gloo upstreams

Let's deploy the spelunker test service, which will serve http and https connections. The response will indicate if the connection 
was http or https, or it will indicate if there is a problem. 

```
kubectl apply -f spelunker.yaml
```

In order to handle incoming https connections, the test server needs the TLS certificate mounted to the file-system.  
Let's save this as a TLS secret containing the key and certificate:
```
kubectl create secret tls tls.spelunker.com --key spelunker.com.key --cert spelunker.com.crt --namespace spelunker
```

We're naming the secret based on the domain for the cert, `spelunker.com`. And we'll prepend `tls` to the front of the name
to make it clear this secret can be used for incoming TLS connections and for initiating TLS connections with an upstream. 
However, later we'll create another secret containing the root certificate, necessary for enabling client-side upstream TLS 
verification in a mutual TLS flow. Let's come back to this later in the guide. 

## Create a route to the http upstream

Now let's create a route to the http port. First, we'll create an upstream:
```
kubectl apply -f upstream.http.spelunker.yaml
```

This upstream contains no SSL configuration, so Gloo knows that when a route is created to this upstream destination, 
it will use plain http between Envoy and the upstream. 

We'll create a route on a virtual service that doesn't have an SSL configuration on the virtual host, so Gloo knows that
this virtual service should be bound to the http listener in Envoy and exposed to end users as plain http. 

``` 
kubectl apply -f vs.http.spelunker.com.yaml
```

For convenience, we're going to grab the Host IP of Gloo's proxy so we can reference it in our commands:
```
export GLOO_HOST=$(kubectl get svc -l gloo=gateway-proxy -n gloo-system -o 'jsonpath={.items[0].status.loadBalancer.ingress[0].ip}')
```

Now we can test the http route with a standard curl:

```
➜ curl http://spelunker.com/ --resolve spelunker.com:80:$GLOO_HOST
This is an example http server.

&{GET / HTTP/1.1 1 1 map[Accept:[*/*] Content-Length:[0] User-Agent:[curl/7.54.0] X-Envoy-Expected-Rq-Timeout-Ms:[15000] X-Forwarded-Proto:[http] X-Request-Id:[608af6a2-651a-407f-816b-354c60cfff6d]] {} <nil> 0 [] false spelunker.com map[] map[] <nil> map[] 10.52.1.235:54848 / <nil> <nil> <nil> 0xc00005c140}
```

In each of our tests we'll explicitly tell curl how to resolve the URL, since we are bypassing DNS setup. 

## Create a route to the https upstreams

Now let's create a route to the https port. First, we'll create an upstream. This time, we'll supply it with an SSL 
configuration, so that Envoy can initiate SSL connections when routing traffic to this upstream destination. 

```
kubectl apply -f upstream.tls.spelunker.yaml
```

Now we can setup an https route to this upstream. We'll define it on a separate virtual service. On this virtual service, 
we'll supply an SSL configuration, so that Gloo knows to bind this route to the https listener in Envoy and users can connect
via https. Since the virtual service may utilize L7 routing features, Envoy will terminate SSL, and then re-initiate it to 
route to the upstream destination. 
``` 
kubectl apply -f vs.https.spelunker.com.yaml
```

We can test the route similar to above, by providing curl with the resolution for the domain we're using. This time, the resolver
will be for the SSL port 443, since that's what we're connecting to. Additionally, we'll pass the cacert to the curl command, so it
can do client-side certificate verification for mutual TLS. 
```
➜ curl https://spelunker.com/ --resolve spelunker.com:443:$GLOO_HOST --cacert rootCA.crt
This is an example https server.

&{GET / HTTP/1.1 1 1 map[Accept:[*/*] Content-Length:[0] User-Agent:[curl/7.54.0] X-Envoy-Expected-Rq-Timeout-Ms:[15000] X-Forwarded-Proto:[https] X-Request-Id:[1cbbfbe2-13c9-4eb7-a628-f44de5e1cbc5]] {} <nil> 0 [] false spelunker.com map[] map[] <nil> map[] 10.52.1.235:38020 / 0xc0001462c0 <nil> <nil> 0xc00012e400}
```

## Create an http client to https upstream route

Now let's start managing a second domain, `spelunker2.com`. For this domain, we want to modify the behavior of Gloo. Now, we'll
route plain http requests into Envoy to the https upstream, configuring Envoy to initial SSL. And we'll terminate SSL for
 https requests into Envoy and route them to the plain http upstream. 

Let's deploy the http route that routes to the https upstream:
```
kubectl apply -f vs.http.spelunker2.com.yaml
```

This looks like a very simple virtual service, since the SSL configuration lives on the upstream already. If we 
test this route, we'll get the expected https response:
``` 
➜ curl http://spelunker2.com/ --resolve spelunker2.com:80:$GLOO_HOST
This is an example https server.

&{GET / HTTP/1.1 1 1 map[Accept:[*/*] Content-Length:[0] User-Agent:[curl/7.54.0] X-Envoy-Expected-Rq-Timeout-Ms:[15000] X-Forwarded-Proto:[http] X-Request-Id:[0a0c5526-7b25-4b13-bdec-ce560aadf929]] {} <nil> 0 [] false spelunker2.com map[] map[] <nil> map[] 10.52.1.235:38118 / 0xc000146370 <nil> <nil> 0xc00012e580}
```

## Create an https client to http upstream route

Now let's try to deploy this virtual service, which will accept https client connections, terminate SSL, and route to an 
http endpoint:
``` 
k apply -f vs.https.spelunker2.com.yaml
```

If we run `glooctl check`, we'll see an error:
``` 
➜ glooctl check
Checking deployments... OK
Checking pods... OK
Checking upstreams... OK
Checking upstream groups... OK
Checking auth configs... OK
Checking secrets... OK
Checking virtual services... OK
Checking gateways... OK
Checking proxies... An update to your gateway-proxy deployment was rejected due to schema/validation errors. The envoy_listener_manager_lds_update_rejected{} metric increased.
You may want to try using the `glooctl proxy logs` or `glooctl debug logs` commands.
Problems detected!
```

The problem here is we haven't included enough information in our virtual services to inform Envoy how to select 
certificates for the two different domains we have on the same https listener. We can resolve this by setting up 
SNI domains that match the domains on each virtual service:
``` 
k apply -f vs.https.spelunker.com-sni.yaml
k apply -f vs.https.spelunker2.com-sni.yaml
```

Now if we run `glooctl check` we'll see the error is resolved:
``` 
➜ glooctl check
Checking deployments... OK
Checking pods... OK
Checking upstreams... OK
Checking upstream groups... OK
Checking auth configs... OK
Checking secrets... OK
Checking virtual services... OK
Checking gateways... OK
Checking proxies... OK
No problems detected.
```

If we test the route at this point, we will still see an error:
```
➜ curl https://spelunker2.com/ --resolve spelunker2.com:443:$GLOO_HOST --cacert rootCA.crt
curl: (51) SSL: certificate subject name 'spelunker.com' does not match target host name 'spelunker2.com'
```

The problem is we didn't create a new certificate for the new domain we set up. Instead, we copied the other https 
virtual service and tried to use the old domain's certificate. We can fix this by creating a new certificate for the new domain:

```
openssl genrsa -out spelunker2.com.key 2048
openssl req -new -key spelunker2.com.key -out spelunker2.com.csr
openssl x509 -req -in spelunker2.com.csr -CA rootCA.crt -CAkey rootCA.key -CAcreateserial -out spelunker2.com.crt -days 500 -sha256
```

And we can put this certificate into a TLS secret:
```
kubectl create secret tls tls.spelunker2.com --key spelunker2.com.key --cert spelunker2.com.crt --namespace spelunker
```

Now we can update our route to use the new certificate, this time for the domain matching the virtual service and SNI domains:
``` 
k apply -f vs.https.spelunker2.com-sni-fixed.yaml
```

Now if we test the route, we'll see the requests return as we expect. The client sends a request to envoy that is encrypted, 
Envoy terminates SSL and forwards the request to the plain http port. 
``` 
➜ curl https://spelunker2.com/ --resolve spelunker2.com:443:$GLOO_HOST --cacert rootCA.crt
This is an example http server.

&{GET / HTTP/1.1 1 1 map[Accept:[*/*] Content-Length:[0] User-Agent:[curl/7.54.0] X-Envoy-Expected-Rq-Timeout-Ms:[15000] X-Forwarded-Proto:[https] X-Request-Id:[4fb08ae3-7313-46eb-b184-2181ba22cdd3]] {} <nil> 0 [] false spelunker2.com map[] map[] <nil> map[] 10.52.1.235:55992 / <nil> <nil> <nil> 0xc0000b40c0}
```

We now have routes for all four cases working: http client to http upstream, https client to https upstream, http client to https upstream, 
and finally https client to http upstream. 

## Enabling client-side verification in Envoy for mutual TLS

So far, we have set up TLS between Envoy and Upstream destinations, and Mutual TLS between clients and Envoy. Now let's 
enable Mutual TLS between Envoy and our upstream destinations. To do that, we need to use a secret that contains the 
Root CA certificate, so that Envoy can verify the client certificate during TLS connection initialization with the upstream. 
If the Root CA exists on the secret, then Gloo uses Mutual TLS; otherwise, it skips client verification. 

To create the secret for mutual TLS, let's use `glooctl` and provide the rootca. Note that we deployed the test server 
originally with certs for `spelunker.com`, so we will set up an mtls secret containing those certificates and the root CA:
```
glooctl create secret tls --privatekey spelunker.com.key --certchain spelunker.com.crt --rootca rootCA.crt mtls.spelunker.com
```

Note that this secret will be written to the `gloo-system` namespace, since it is used by the `gateway-proxy` deployment 
during the TLS handshake. 

Now we'll create an upstream that references this Mutual TLS secret, so Envoy will see the root CA when initiating TLS. 
```
k apply -f upstream.mtls.spelunker.yaml
```

We'll update both virtual services that routed to the tls upstream to now route to the mtls upstream instead. 
```
k apply -f vs.https.spelunker.com-mtls.yaml
k apply -f vs.http.spelunker2.com-mtls.yaml
```

Now we can see all four routes behaving as expected, with mtls enabled for all SSL communication. 

``` 
➜ curl http://spelunker.com/ --resolve spelunker.com:80:$GLOO_HOST
This is an example http server.

&{GET / HTTP/1.1 1 1 map[Accept:[*/*] Content-Length:[0] User-Agent:[curl/7.54.0] X-Envoy-Expected-Rq-Timeout-Ms:[15000] X-Forwarded-Proto:[http] X-Request-Id:[725f6e74-9e4c-45a7-a9a5-74a12c366454]] {} <nil> 0 [] false spelunker.com map[] map[] <nil> map[] 10.52.1.235:56722 / <nil> <nil> <nil> 0xc00012e680}

➜ curl https://spelunker.com/ --resolve spelunker.com:443:$GLOO_HOST --cacert rootCA.crt
This is an example https server.

&{GET / HTTP/1.1 1 1 map[Accept:[*/*] Content-Length:[0] User-Agent:[curl/7.54.0] X-Envoy-Expected-Rq-Timeout-Ms:[15000] X-Forwarded-Proto:[https] X-Request-Id:[085bd2f4-d368-4b7c-8bb6-31b7b023437c]] {} <nil> 0 [] false spelunker.com map[] map[] <nil> map[] 10.52.1.235:38118 / 0xc000146370 <nil> <nil> 0xc0000b4140}

➜ curl http://spelunker2.com/ --resolve spelunker2.com:80:$GLOO_HOST
This is an example https server.

&{GET / HTTP/1.1 1 1 map[Accept:[*/*] Content-Length:[0] User-Agent:[curl/7.54.0] X-Envoy-Expected-Rq-Timeout-Ms:[15000] X-Forwarded-Proto:[http] X-Request-Id:[47bd3617-8ca1-447c-b9b5-1569be3f90fd]] {} <nil> 0 [] false spelunker2.com map[] map[] <nil> map[] 10.52.1.235:38020 / 0xc0001462c0 <nil> <nil> 0xc00012e6c0}

➜ curl https://spelunker2.com/ --resolve spelunker2.com:443:$GLOO_HOST --cacert rootCA.crt
This is an example http server.

&{GET / HTTP/1.1 1 1 map[Accept:[*/*] Content-Length:[0] User-Agent:[curl/7.54.0] X-Envoy-Expected-Rq-Timeout-Ms:[15000] X-Forwarded-Proto:[https] X-Request-Id:[b021ebc5-4052-4d9b-aaa2-6ea0f833f3bb]] {} <nil> 0 [] false spelunker2.com map[] map[] <nil> map[] 10.52.1.235:56834 / <nil> <nil> <nil> 0xc0000b4240}

```


