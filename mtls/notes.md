Create Root CA

```
openssl genrsa -des3 -out rootCA.key 4096
openssl req -x509 -new -nodes -key rootCA.key -sha256 -days 1024 -out rootCA.crt
```

Generate Cert
```
openssl genrsa -out valet-test.com.key 2048
openssl req -new -key valet-test.com.key -out valet-test.com.csr
openssl x509 -req -in valet-test.com.csr -CA rootCA.crt -CAkey rootCA.key -CAcreateserial -out valet-test.com.crt -days 500 -sha256
```

Create TLS secret
```
kubectl create secret tls tls.valet-test.com --key valet-test.com.key --cert valet-test.com.crt --namespace valet-test-server
```

Create upstream for http 
```
kubectl apply -f upstream.http.valet-test-server.yaml
```

Create upstream for tls
```
kubectl apply -f upstream.tls.valet-test-server.yaml
```

Set up http and https routes
```
kubectl apply -f vs.http.valet-test.com.yaml
kubectl apply -f vs.https.valet-test.com.yaml
```

Glooctl check

Test routes
```
➜ curl http://valet-test.com/ --resolve valet-test.com:80:35.227.127.150 --resolve valet-test.com:443:35.227.127.150 --cacert test-server/rootCA.crt
This is an example http server.

&{GET / HTTP/1.1 1 1 map[Accept:[*/*] Content-Length:[0] User-Agent:[curl/7.54.0] X-Envoy-Expected-Rq-Timeout-Ms:[15000] X-Forwarded-Proto:[http] X-Request-Id:[0760976d-063f-4086-8de6-179b420b092b]] {} <nil> 0 [] false valet-test.com map[] map[] <nil> map[] 10.52.1.92:39522 / <nil> <nil> <nil> 0xc00005c980}

➜ curl https://valet-test.com/ --resolve valet-test.com:80:35.227.127.150 --resolve valet-test.com:443:35.227.127.150 --cacert test-server/rootCA.crt
This is an example https server.

&{GET / HTTP/1.1 1 1 map[Accept:[*/*] Content-Length:[0] User-Agent:[curl/7.54.0] X-Envoy-Expected-Rq-Timeout-Ms:[15000] X-Forwarded-Proto:[https] X-Request-Id:[91818159-51cd-4ef8-ac35-207143884e38]] {} <nil> 0 [] false valet-test.com map[] map[] <nil> map[] 10.52.1.92:45538 / 0xc00010c630 <nil> <nil> 0xc00005ca80}
```

Deploy new http route on new domain

``` 
➜ curl http://valet-test2.com/ --resolve valet-test2.com:80:35.227.127.150 --resolve valet-test2.com:443:35.227.127.150 --cacert test-server/rootCA.crt
This is an example https server.

&{GET / HTTP/1.1 1 1 map[Accept:[*/*] Content-Length:[0] User-Agent:[curl/7.54.0] X-Envoy-Expected-Rq-Timeout-Ms:[15000] X-Forwarded-Proto:[http] X-Request-Id:[c7847d8d-8243-4305-bbf7-fc0f079b8e04]] {} <nil> 0 [] false valet-test2.com map[] map[] <nil> map[] 10.52.1.92:45538 / 0xc00010c630 <nil> <nil> 0xc00009ec00}
```

Now deploy other https domain

``` 
k apply -f vs.https.valet-test2.com.yaml
```

Glooctl check

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

Set up SNI domains

``` 
k apply -f vs.https.valet-test.com-sni.yaml
k apply -f vs.https.valet-test2.com-sni.yaml
```

Glooctl check

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

Test http route

```
➜ curl http://valet-test2.com/ --resolve valet-test2.com:80:35.227.127.150 --resolve valet-test2.com:443:35.227.127.150 --cacert test-server/rootCA.crt
This is an example https server.

&{GET / HTTP/1.1 1 1 map[Accept:[*/*] Content-Length:[0] User-Agent:[curl/7.54.0] X-Envoy-Expected-Rq-Timeout-Ms:[15000] X-Forwarded-Proto:[http] X-Request-Id:[5434f722-cbde-436a-b989-61d217baf4d9]] {} <nil> 0 [] false valet-test2.com map[] map[] <nil> map[] 10.52.1.92:46716 / 0xc00010c790 <nil> <nil> 0xc00005cb80}
```

Test https route
```
➜ curl https://valet-test2.com/ --resolve valet-test2.com:80:35.227.127.150 --resolve valet-test2.com:443:35.227.127.150 --cacert test-server/rootCA.crt
curl: (51) SSL: certificate subject name 'valet-test.com' does not match target host name 'valet-test2.com'
```

Create secret for tls to valet-test2.com
```
kubectl create secret tls tls.valet-test2.com --key valet-test2.com.key --cert valet-test2.com.crt --namespace valet-test-server
```

Update https route
``` 
k apply -f vs.https.valet-test2.com-sni-fixed.yaml
```

Test the route
``` 
➜ curl https://valet-test2.com/ --resolve valet-test2.com:80:35.227.127.150 --resolve valet-test2.com:443:35.227.127.150 --cacert test-server/rootCA.crt
This is an example http server.

&{GET / HTTP/1.1 1 1 map[Accept:[*/*] Content-Length:[0] User-Agent:[curl/7.54.0] X-Envoy-Expected-Rq-Timeout-Ms:[15000] X-Forwarded-Proto:[https] X-Request-Id:[628eb5d3-2cd5-4e89-8080-70f79b3bed60]] {} <nil> 0 [] false valet-test2.com map[] map[] <nil> map[] 10.52.1.92:41322 / <nil> <nil> <nil> 0xc00009ecc0}
```

Create secret for mtls
```
glooctl create secret tls --privatekey valet-test2.com.key --certchain valet-test2.com.crt --rootca test-server/rootCA.crt tls-glooctl.valet-test2.com
```

Create mtls upstream
```
k apply -f upstream.mtls.valet-test-server.yaml
```

Update https routes to use mtls destination to turn on client-side verification
```
k apply -f vs.https.valet-test.com-mtls.yaml
k apply -f vs.http.valet-test2.com-mtls.yaml
```

Now we can see all four routes behaving as expected, with mtls enabled for all SSL communication. 

``` 
➜ curl http://valet-test.com/ --resolve valet-test.com:80:35.227.127.150 --resolve valet-test.com:443:35.227.127.150 --cacert test-server/rootCA.crt
This is an example http server.

&{GET / HTTP/1.1 1 1 map[Accept:[*/*] Content-Length:[0] User-Agent:[curl/7.54.0] X-Envoy-Expected-Rq-Timeout-Ms:[15000] X-Forwarded-Proto:[http] X-Request-Id:[09c58505-2f41-40b4-b205-201d2d4bd4c8]] {} <nil> 0 [] false valet-test.com map[] map[] <nil> map[] 10.52.1.92:39522 / <nil> <nil> <nil> 0xc00009ed40}

➜ curl https://valet-test.com/ --resolve valet-test.com:80:35.227.127.150 --resolve valet-test.com:443:35.227.127.150 --cacert test-server/rootCA.crt
This is an example https server.

&{GET / HTTP/1.1 1 1 map[Accept:[*/*] Content-Length:[0] User-Agent:[curl/7.54.0] X-Envoy-Expected-Rq-Timeout-Ms:[15000] X-Forwarded-Proto:[https] X-Request-Id:[ddedadd9-a421-4818-bb58-748df7cd67fa]] {} <nil> 0 [] false valet-test.com map[] map[] <nil> map[] 10.52.1.92:47662 / 0xc0000b48f0 <nil> <nil> 0xc00009ee40}

➜ curl http://valet-test2.com/ --resolve valet-test2.com:80:35.227.127.150 --resolve valet-test2.com:443:35.227.127.150 --cacert test-server/rootCA.crt
This is an example https server.

&{GET / HTTP/1.1 1 1 map[Accept:[*/*] Content-Length:[0] User-Agent:[curl/7.54.0] X-Envoy-Expected-Rq-Timeout-Ms:[15000] X-Forwarded-Proto:[http] X-Request-Id:[762fe132-48ee-4b67-ad5f-d3602cfb734c]] {} <nil> 0 [] false valet-test2.com map[] map[] <nil> map[] 10.52.1.92:47622 / 0xc0001be4d0 <nil> <nil> 0xc00018e700}

➜ curl https://valet-test2.com/ --resolve valet-test2.com:80:35.227.127.150 --resolve valet-test2.com:443:35.227.127.150 --cacert test-server/rootCA.crt
This is an example http server.

&{GET / HTTP/1.1 1 1 map[Accept:[*/*] Content-Length:[0] User-Agent:[curl/7.54.0] X-Envoy-Expected-Rq-Timeout-Ms:[15000] X-Forwarded-Proto:[https] X-Request-Id:[fbfeb68d-9266-45fd-898d-35e1cd7ebba4]] {} <nil> 0 [] false valet-test2.com map[] map[] <nil> map[] 10.52.1.92:41322 / <nil> <nil> <nil> 0xc00009ed00}

```


