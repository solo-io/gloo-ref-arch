# Network encryption with Gloo

Let's explore L7 TLS and MTLS support in Gloo. 

## Set up root CA

In order to simulate certificate management, we'll generate a local root CA. We can create the root key and certificate
by running the following two commands:

```
openssl genrsa -des3 -out rootCA.key 4096
openssl req -x509 -new -nodes -key rootCA.key -sha256 -days 1024 -out rootCA.crt
```

Then, we can start to generate local keys. We can create a certificate signing request for those keys, and then 
we can create the certificate. We'll go ahead and create a certificate for a domain `valet-test.com`, which we'll 
be using throughout this example. 

```
openssl genrsa -out valet-test.com.key 2048
openssl req -new -key valet-test.com.key -out valet-test.com.csr
openssl x509 -req -in valet-test.com.csr -CA rootCA.crt -CAkey rootCA.key -CAcreateserial -out valet-test.com.crt -days 500 -sha256
```

## Deploy test server and Gloo upstreams

Let's deploy the test server, which will serve http and https connections. The response will indicate if the connection 
was http or https, or it will indicate if there is a problem. 

```
kubectl apply -f test-server.yaml
```

In order to handle incoming https connections, the test server needs the TLS certificate mounted to the file-system.  
Let's save this as a TLS secret containing the key and certificate:
```
kubectl create secret tls tls.valet-test.com --key valet-test.com.key --cert valet-test.com.crt --namespace valet-test-server
```

We're naming the secret based on the domain for the cert, `valet-test.com`. And we'll prepend `tls` to the front of the name
to make it clear this secret can be used for incoming TLS connections and for initiating TLS connections with an upstream. 
However, later we'll create another secret containing the root certificate, necessary for enabling client-side upstream TLS 
verification in a mutual TLS flow. Let's come back to this later in the guide. 

## Create a route to the http upstream

Now let's create a route to the http port. First, we'll create an upstream:
```
kubectl apply -f upstream.http.valet-test-server.yaml
```

This upstream contains no SSL configuration, so Gloo knows that when a route is created to this upstream destination, 
it will use plain http between Envoy and the upstream. 

We'll create a route on a virtual service that doesn't have an SSL configuration on the virtual host, so Gloo knows that
this virtual service should be bound to the http listener in Envoy and exposed to end users as plain http. 

``` 
kubectl apply -f vs.http.valet-test.com.yaml
```

Now we can test the http route with a standard curl:

```
➜ curl http://valet-test.com/ --resolve valet-test.com:80:35.227.127.150 
This is an example http server.

&{GET / HTTP/1.1 1 1 map[Accept:[*/*] Content-Length:[0] User-Agent:[curl/7.54.0] X-Envoy-Expected-Rq-Timeout-Ms:[15000] X-Forwarded-Proto:[http] X-Request-Id:[0760976d-063f-4086-8de6-179b420b092b]] {} <nil> 0 [] false valet-test.com map[] map[] <nil> map[] 10.52.1.92:39522 / <nil> <nil> <nil> 0xc00005c980}
```

Note that we are issuing the request to the domain, so we tell curl to resolve http requests to that domain to the 
gateway proxy IP address that is externally visible from our cluster. 

## Create a route to the https upstreams

Now let's create a route to the https port. First, we'll create an upstream. This time, we'll supply it with an SSL 
configuration, so that Envoy can initiate SSL connections when routing traffic to this upstream destination. 

```
kubectl apply -f upstream.https.valet-test-server.yaml
```

Now we can setup an https route to this upstream. We'll define it on a separate virtual service. On this virtual service, 
we'll supply an SSL configuration, so that Gloo knows to bind this route to the https listener in Envoy and users can connect
via https. Since the virtual service may utilize L7 routing features, Envoy will terminate SSL, and then re-initiate it to 
route to the upstream destination. 
``` 
kubectl apply -f vs.https.valet-test.com.yaml
```

We can test the route similar to above, by providing curl with the resolution for the domain we're using. This time, the resolver
will be for the SSL port 443, since that's what we're connecting to. Additionally, we'll pass the cacert to the curl command, so it
can do client-side certificate verification for mutual TLS. 
```
➜ curl https://valet-test.com/ --resolve valet-test.com:443:35.227.127.150 --cacert rootCA.crt
This is an example https server.

&{GET / HTTP/1.1 1 1 map[Accept:[*/*] Content-Length:[0] User-Agent:[curl/7.54.0] X-Envoy-Expected-Rq-Timeout-Ms:[15000] X-Forwarded-Proto:[https] X-Request-Id:[91818159-51cd-4ef8-ac35-207143884e38]] {} <nil> 0 [] false valet-test.com map[] map[] <nil> map[] 10.52.1.92:45538 / 0xc00010c630 <nil> <nil> 0xc00005ca80}
```

## Create an http client to https upstream route

Now let's start managing a second domain, `valet-test2.com`. For this domain, we want to modify the behavior of Gloo. Now, we'll
route plain http requests into Envoy to the https upstream, configuring Envoy to initial SSL. And we'll terminate SSL for
 https requests into Envoy and route them to the plain http upstream. 

Let's deploy the http route that routes to the https upstream:
```
kubectl apply -f vs.http.valet-test2.com.yaml
```

This looks like a very simple virtual service, since the SSL configuration lives on the upstream already. If we 
test this route, we'll get the expected https response:
``` 
➜ curl http://valet-test2.com/ --resolve valet-test2.com:80:35.227.127.150
This is an example https server.

&{GET / HTTP/1.1 1 1 map[Accept:[*/*] Content-Length:[0] User-Agent:[curl/7.54.0] X-Envoy-Expected-Rq-Timeout-Ms:[15000] X-Forwarded-Proto:[http] X-Request-Id:[c7847d8d-8243-4305-bbf7-fc0f079b8e04]] {} <nil> 0 [] false valet-test2.com map[] map[] <nil> map[] 10.52.1.92:45538 / 0xc00010c630 <nil> <nil> 0xc00009ec00}
```

## Create an https client to http upstream route

Now let's try to deploy this virtual service:
``` 
k apply -f vs.https.valet-test2.com.yaml
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
k apply -f vs.https.valet-test.com-sni.yaml
k apply -f vs.https.valet-test2.com-sni.yaml
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
➜ curl https://valet-test2.com/ --resolve valet-test2.com:443:35.227.127.150 --cacert rootCA.crt
curl: (51) SSL: certificate subject name 'valet-test.com' does not match target host name 'valet-test2.com'
```

The problem is we didn't create a new certificate for the new domain we set up. Instead, we copied the other https 
virtual service and tried to use the old domain's certificate. We can fix this by creating a new certificate for the new domain:
```
kubectl create secret tls tls.valet-test2.com --key valet-test2.com.key --cert valet-test2.com.crt --namespace valet-test-server
``` 

Now we can update our route to use the new certificate, this time for the domain matching the virtual service and SNI domains:
``` 
k apply -f vs.https.valet-test2.com-sni-fixed.yaml
```

Now if we test the route, we'll see the requests return as we expect. The client sends a request to envoy that is encrypted, 
Envoy terminates SSL and forwards the request to the plain http port. 
``` 
➜ curl https://valet-test2.com/ --resolve valet-test2.com:443:35.227.127.150 --cacert rootCA.crt
This is an example http server.

&{GET / HTTP/1.1 1 1 map[Accept:[*/*] Content-Length:[0] User-Agent:[curl/7.54.0] X-Envoy-Expected-Rq-Timeout-Ms:[15000] X-Forwarded-Proto:[https] X-Request-Id:[628eb5d3-2cd5-4e89-8080-70f79b3bed60]] {} <nil> 0 [] false valet-test2.com map[] map[] <nil> map[] 10.52.1.92:41322 / <nil> <nil> <nil> 0xc00009ecc0}
```

We now have routes for all four cases working: http client to http upstream, https client to https upstream, http client to https upstream, 
and finally https client to http upstream. 

## Enabling client-side verification in Envoy for mutual TLS

So far, we have set up TLS between Envoy and Upstream destinations, and Mutual TLS between clients and Envoy. Now let's 
enable Mutual TLS between Envoy and our upstream destinations. To do that, we need to use a secret that contains the 
Root CA certificate, so that Envoy can verify the client certificate during TLS connection initialization with the upstream. 
If the Root CA exists on the secret, then Gloo uses Mutual TLS; otherwise, it skips client verification. 

To create the secret for mutual TLS, let's use `glooctl` and provide the rootca. Note that we deployed the test server 
originally with certs for `valet-test.com`, so we will set up an mtls secret containing those certificates and the root CA:
```
glooctl create secret tls --privatekey valet-test.com.key --certchain valet-test.com.crt --rootca rootCA.crt mtls.valet-test.com
```

Now we'll create an upstream that references this Mutual TLS secret, so Envoy will see the root CA when initiating TLS. 
```
k apply -f upstream.mtls.valet-test-server.yaml
```

We'll update both virtual services that routed to the tls upstream to now route to the mtls upstream instead. 
```
k apply -f vs.https.valet-test.com-mtls.yaml
k apply -f vs.http.valet-test2.com-mtls.yaml
```

Now we can see all four routes behaving as expected, with mtls enabled for all SSL communication. 

``` 
➜ curl http://valet-test.com/ --resolve valet-test.com:80:35.227.127.150
This is an example http server.

&{GET / HTTP/1.1 1 1 map[Accept:[*/*] Content-Length:[0] User-Agent:[curl/7.54.0] X-Envoy-Expected-Rq-Timeout-Ms:[15000] X-Forwarded-Proto:[http] X-Request-Id:[09c58505-2f41-40b4-b205-201d2d4bd4c8]] {} <nil> 0 [] false valet-test.com map[] map[] <nil> map[] 10.52.1.92:39522 / <nil> <nil> <nil> 0xc00009ed40}

➜ curl https://valet-test.com/ --resolve valet-test.com:443:35.227.127.150 --cacert rootCA.crt
This is an example https server.

&{GET / HTTP/1.1 1 1 map[Accept:[*/*] Content-Length:[0] User-Agent:[curl/7.54.0] X-Envoy-Expected-Rq-Timeout-Ms:[15000] X-Forwarded-Proto:[https] X-Request-Id:[ddedadd9-a421-4818-bb58-748df7cd67fa]] {} <nil> 0 [] false valet-test.com map[] map[] <nil> map[] 10.52.1.92:47662 / 0xc0000b48f0 <nil> <nil> 0xc00009ee40}

➜ curl http://valet-test2.com/ --resolve valet-test2.com:80:35.227.127.150
This is an example https server.

&{GET / HTTP/1.1 1 1 map[Accept:[*/*] Content-Length:[0] User-Agent:[curl/7.54.0] X-Envoy-Expected-Rq-Timeout-Ms:[15000] X-Forwarded-Proto:[http] X-Request-Id:[762fe132-48ee-4b67-ad5f-d3602cfb734c]] {} <nil> 0 [] false valet-test2.com map[] map[] <nil> map[] 10.52.1.92:47622 / 0xc0001be4d0 <nil> <nil> 0xc00018e700}

➜ curl https://valet-test2.com/ --resolve valet-test2.com:443:35.227.127.150 --cacert rootCA.crt
This is an example http server.

&{GET / HTTP/1.1 1 1 map[Accept:[*/*] Content-Length:[0] User-Agent:[curl/7.54.0] X-Envoy-Expected-Rq-Timeout-Ms:[15000] X-Forwarded-Proto:[https] X-Request-Id:[fbfeb68d-9266-45fd-898d-35e1cd7ebba4]] {} <nil> 0 [] false valet-test2.com map[] map[] <nil> map[] 10.52.1.92:41322 / <nil> <nil> <nil> 0xc00009ed00}

```


