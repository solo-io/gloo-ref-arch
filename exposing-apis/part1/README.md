# Exposing APIs with Gloo, Part 1

In this series, we'll be looking at how Gloo Enterprise can support common enterprise needs when 
using it to expose APIs to external users or services. 

For the first part, we're going to look at increasingly complex -- and powerful -- ways to express different 
rate limits on your routes. We'll also look to enhance the security of those routes using JWT verification in Envoy, 
Open Policy Agent for additional JWT authorization, and the Web Application Firewall -- all available out of the box 
with Enterprise Gloo. 

## Setup

This guide assumes you already have a Kubernetes cluster running, and have installed Enterprise Gloo. 

If you don't yet have access to Enterprise, you can request a 30-day trial license [here](TODO). 

We'll install Gloo Enterprise into the `gloo-system` namespace with the default helm values:

```
glooctl install gateway enterprise --license-key $LICENSE_KEY 
```

Once the installation is complete, you should see all of the pods running in the `gloo-system` namespace. We 
are now ready to deploy our application. 

## Deploy the petstore application

For this post, we'll use the petstore application. This is a simple REST API that we'll deploy to the default namespace. 
Gloo's discovery backend will detect the Kubernetes deployment and service were added, and create an upstream 
to represent the destination for traffic. 

First, let's create the petstore resources:

```
kubectl apply -f petstore.yaml
```

Wait until the pods in namespace 'default' are ready. Use `kubectl get pods -n default` to check the status.

Now, we can add a route to Gloo by creating a basic virtual service that looks like this:

```yaml
apiVersion: gateway.solo.io/v1
kind: VirtualService
metadata:
  name: petstore
  namespace: gloo-system
spec:
  virtualHost:
    domains:
      - '*'
    routes:
      - matchers:
          - exact: /sample-route-1
        routeAction:
          single:
            upstream:
              name: default-petstore-8080
              namespace: gloo-system
        options:
          prefixRewrite: /api/pets
```

This is a simple route that exposes the API in Gloo. The only feature we've enabled so far is a prefix 
rewrite, so we can expose the API with the desired URI, which we'll use throughout the workflow. 
In the subsequent steps, we'll explore how to turn on 
options on this route. We can apply this route to the cluster with the following command:

```
kubectl apply -f vs-petstore-1.yaml
```

Finally, let's test the application. We'll use the nested command `%(glooctl proxy url)` to 
determine the http address for the external gateway proxy service, and issue curl commands to our 
route:

```
➜ curl $(glooctl proxy url)/sample-route-1
[{"id":1,"name":"Dog","status":"available"},{"id":2,"name":"Cat","status":"pending"}]
```

The API is exposed in Gloo. Now, let's begin exploring how to apply rate limiting to this route. 

## Initial rate limiting setup

There are two areas we need to configure to start using rate limiting. 

First, we need to update the Gloo settings to define one or more rules, that specify limits associated 
with unique sets of descriptors. To start, let's use a very simple example. 

spec:
  ratelimit:
    descriptors:
      - key: generic_key
        rateLimit:
          requestsPerUnit: 1
          unit: MINUTE
        value: some_value


This configuration defines a counter for a request that use the descriptor `generic_key: some_value`, and 
a limit of one per minute. We can apply it to the cluster with the following command:

kubectl patch -n gloo-system settings default --type merge --patch "$(cat settings-patch-1.yaml)"

Now we can add update our virtual service to increment this rate limiting counter. 

```yaml
apiVersion: gateway.solo.io/v1
kind: VirtualService
metadata:
  name: petstore
  namespace: gloo-system
spec:
  virtualHost:
    domains:
      - '*'
    routes:
      - matchers:
          - exact: /sample-route-1
        routeAction:
          single:
            upstream:
              name: default-petstore-8080
              namespace: gloo-system
        options:
          prefixRewrite: /api/pets
    options:
      ratelimit:
        rateLimits:
          - actions:
              - generic_key:
                  descriptor_value: "some_value"

```

Note that we've hard-coded the rate limiting action to use the key `generic_key`, and value `some_value` as a 
hard-coded literal. Let's deploy this to the cluster:

```
kubectl apply -f vs-petstore-2.yaml
```

And finally, we can test the rate limit by issuing a few curl requests with the `-v` flag and inspecting the response:

``` 
➜ curl $(glooctl proxy url)/sample-route-1 -v
*   Trying 35.227.127.150...
* TCP_NODELAY set
* Connected to 35.227.127.150 (35.227.127.150) port 80 (#0)
> GET /sample-route-1 HTTP/1.1
> Host: 35.227.127.150
> User-Agent: curl/7.54.0
> Accept: */*
>
< HTTP/1.1 200 OK
< content-type: application/xml
< date: Tue, 14 Apr 2020 19:20:26 GMT
< content-length: 86
< x-envoy-upstream-service-time: 2
< server: envoy
<
[{"id":1,"name":"Dog","status":"available"},{"id":2,"name":"Cat","status":"pending"}]
* Connection #0 to host 35.227.127.150 left intact

➜ curl $(glooctl proxy url)/sample-route-1 -v
*   Trying 35.227.127.150...
* TCP_NODELAY set
* Connected to 35.227.127.150 (35.227.127.150) port 80 (#0)
> GET /sample-route-1 HTTP/1.1
> Host: 35.227.127.150
> User-Agent: curl/7.54.0
> Accept: */*
>
< HTTP/1.1 429 Too Many Requests
< x-envoy-ratelimited: true
< date: Tue, 14 Apr 2020 19:20:27 GMT
< server: envoy
< content-length: 0
<
* Connection #0 to host 35.227.127.150 left intact
```

On the second request, we see a `429 Too Many Requests` response. Our simple rate limit rule worked. 
Next, we'll make the actions more dynamic by referencing headers in the request
 
## Advanced rate limit rules and actions

Now we're going to make the scenario a little more realistic. Let's say this API is serving requests that 
carry information related to messaging - the type of message, and a number. For now, let's say values are 
defined in headers `x-type` and `x-number`, respectively. 

We wish to express a few different rules. First, let's limit requests of type `Messenger` to two per minute. 
Second, we will limit requests of type `Whatsapp` to one per minute, **except** if the request is to number `411`, 
in which case we'll allow 30 requests per second. 

We can express that in our settings with the following rule configuration:

spec:
  ratelimit:
    descriptors:
      # Rule 1, limit all Messenger requests to 2/min
      - key: type
        value: Messenger
        rateLimit:
          requestsPerUnit: 2
          unit: MINUTE
      # Rule 2, limit all Whatsapp requests to 1/min
      - key: type
        value: Whatsapp
        rateLimit:
          requestsPerUnit: 1
          unit: MINUTE
        descriptors:
          # Rule 3, limit all Whatsapp requests to number '411' to 100/min
          - key: number
            rateLimit:
              requestsPerUnit: 30
              unit: MINUTE
            value: "411"
            weight: 1 # Rule 3 takes priority over other rules

This expresses both our `Messenger` rule, and our nested `Whatsapp` rules. By setting a weight on the 
nested rule for `Whatsapp` messages to number `411`, we can ensure that has priority. Let's apply that 
to the cluster:

kubectl patch -n gloo-system settings default --type merge --patch "$(cat settings-patch-2.yaml)"

Now, we can update our virtual service with new actions that inform if the request matches one or more
of the rules in our settings:

```yaml
apiVersion: gateway.solo.io/v1
kind: VirtualService
metadata:
  name: petstore
  namespace: gloo-system
spec:
  virtualHost:
    domains:
      - '*'
    routes:
      - matchers:
          - exact: /sample-route-1
        routeAction:
          single:
            upstream:
              name: default-petstore-8080
              namespace: gloo-system
        options:
          prefixRewrite: /api/pets
    options:
      ratelimit:
        rateLimits:
          # generate descriptors for Rule 1, 2 (and 4)
          - actions:
              - requestHeaders:
                  descriptorKey: type
                  headerName: x-type
          # generate descriptors for Rule 3
          - actions:
              - requestHeaders:
                  descriptorKey: type
                  headerName: x-type
              - requestHeaders:
                  descriptorKey: number
                  headerName: x-number

```

Here we define two actions. First, we generate descriptors associated with just the type of the message, 
which we extract from the `x-type` header. We also generate a second set of descriptors based on both 
the `type` and `number` combination, extracted from the `x-type` and `x-number` headers respectively. 

Let's apply that to the cluster:

```
kubectl apply -f vs-petstore-3.yaml
``` 

Now we can test the various limits. First, if we curl with `x-type: Messenger` we'll see the third request
rate-limited:

```
➜ curl $(glooctl proxy url)/sample-route-1 -H "x-type: Messenger"
[{"id":1,"name":"Dog","status":"available"},{"id":2,"name":"Cat","status":"pending"}]

➜ curl $(glooctl proxy url)/sample-route-1 -H "x-type: Messenger"
[{"id":1,"name":"Dog","status":"available"},{"id":2,"name":"Cat","status":"pending"}]

➜ curl $(glooctl proxy url)/sample-route-1 -H "x-type: Messenger" -v
*   Trying 35.227.127.150...
* TCP_NODELAY set
* Connected to 35.227.127.150 (35.227.127.150) port 80 (#0)
> GET /sample-route-1 HTTP/1.1
> Host: 35.227.127.150
> User-Agent: curl/7.54.0
> Accept: */*
> x-type: Messenger
>
< HTTP/1.1 429 Too Many Requests
< x-envoy-ratelimited: true
< date: Tue, 14 Apr 2020 20:14:14 GMT
< server: envoy
< content-length: 0
<
* Connection #0 to host 35.227.127.150 left intact
```

Next, if we curl with headers `x-type: Whatsapp` and `x-number: 311`, we'll get rate limited 
on the second request:

``` 
➜ curl $(glooctl proxy url)/sample-route-1 -v -H "x-type: Whatsapp" -H "x-number: 311"
[{"id":1,"name":"Dog","status":"available"},{"id":2,"name":"Cat","status":"pending"}]

➜ curl $(glooctl proxy url)/sample-route-1 -v -H "x-type: Whatsapp" -H "x-number: 311"
*   Trying 35.227.127.150...
* TCP_NODELAY set
* Connected to 35.227.127.150 (35.227.127.150) port 80 (#0)
> GET /sample-route-1 HTTP/1.1
> Host: 35.227.127.150
> User-Agent: curl/7.54.0
> Accept: */*
> x-type: Whatsapp
> x-number: 311
>
< HTTP/1.1 429 Too Many Requests
< x-envoy-ratelimited: true
< date: Tue, 14 Apr 2020 20:17:37 GMT
< server: envoy
< content-length: 0
<
* Connection #0 to host 35.227.127.150 left intact
```

But, if we curl to `x-number: 411`, the request will succeed, since for these values specifically 
the limit is 30/minute. 

```
➜ curl $(glooctl proxy url)/sample-route-1 -v -H "x-type: Whatsapp" -H "x-number: 411"
[{"id":1,"name":"Dog","status":"available"},{"id":2,"name":"Cat","status":"pending"}]

➜ curl $(glooctl proxy url)/sample-route-1 -v -H "x-type: Whatsapp" -H "x-number: 411"
[{"id":1,"name":"Dog","status":"available"},{"id":2,"name":"Cat","status":"pending"}]
```

### Adding a fallback rule

Now, if we issue requests with a new type, there will be no rate limits applied to the request, since
we only have defined rules for types `Messenger` and `Whatsapp`. We can see that this request 
repeatedly succeeds:

```
➜ curl $(glooctl proxy url)/sample-route-1 -H "x-type: SMS"
[{"id":1,"name":"Dog","status":"available"},{"id":2,"name":"Cat","status":"pending"}] 
```

To fix this, let's add one more fallback rule, so that any other type of message is limited to 1/min. 

spec:
  ratelimit:
    descriptors:
      # Rule 1, limit all Messenger requests to 2/min
      - key: type
        value: Messenger
        rateLimit:
          requestsPerUnit: 2
          unit: MINUTE

      # Rule 2, limit all Whatsapp requests to 1/min
      - key: type
        value: Whatsapp
        rateLimit:
          requestsPerUnit: 1
          unit: MINUTE
        descriptors:
          # Rule 3, limit all Whatsapp requests to number '411' to 100/min
          - key: number
            rateLimit:
              requestsPerUnit: 30
              unit: MINUTE
            value: "411"
            weight: 1 # Rule 3 takes priority over other rules

      # Rule 4 - all other types are rate limited to 1/min
      - key: type
        rateLimit:
          requestsPerUnit: 1
          unit: MINUTE

We can apply this with the following command: 

kubectl patch -n gloo-system settings default --type merge --patch "$(cat settings-patch-3.yaml)"

Now if we issue the same curl, we'll see the second request rate limited:

``` 
➜ curl $(glooctl proxy url)/sample-route-1 -H "x-type: SMS"
[{"id":1,"name":"Dog","status":"available"},{"id":2,"name":"Cat","status":"pending"}]

➜ curl $(glooctl proxy url)/sample-route-1 -H "x-type: SMS" -v
*   Trying 35.227.127.150...
* TCP_NODELAY set
* Connected to 35.227.127.150 (35.227.127.150) port 80 (#0)
> GET /sample-route-1 HTTP/1.1
> Host: 35.227.127.150
> User-Agent: curl/7.54.0
> Accept: */*
> x-type: SMS
>
< HTTP/1.1 429 Too Many Requests
< x-envoy-ratelimited: true
< date: Tue, 14 Apr 2020 20:22:56 GMT
< server: envoy
< content-length: 0
<
* Connection #0 to host 35.227.127.150 left intact
```

All our other rules continue to work as before. 

## Extracting the descriptors from JWT claims

Now that we have our basic rate limiting rules defined, we want to enhance the security of our API. 
The first thing we want to do is prevent users from being able to manually bypass the rate limiting 
rules by changing header values on the client side. Instead, we'll require requests contain a valid 
JWT token, and we'll extract the `type` and `number` headers from claims in the JWT. 

For testing, we can create a private and public key pair with the following commands:

``` 
openssl genrsa 2048 > private-key.pem
openssl rsa -in private-key.pem -pubout > public-key.pem
```

Now we have a private and public key to help us construct and verify JWTs. First, let's add JWT 
verification to our route by updating the virtual service with `jwt` options:

```yaml
apiVersion: gateway.solo.io/v1
kind: VirtualService
metadata:
  name: petstore
  namespace: gloo-system
spec:
  virtualHost:
    domains:
      - '*'
    routes:
      - matchers:
          - exact: /sample-route-1
        routeAction:
          single:
            upstream:
              name: default-petstore-8080
              namespace: gloo-system
        options:
          prefixRewrite: /api/pets
    options:
      jwt:
        providers:
          solo:
            tokenSource:
              headers:
                - header: x-token
              queryParams:
                - token
            claimsToHeaders:
              - claim: type
                header: x-type
              - claim: number
                header: x-number
            issuer: solo.io
            jwks:
              local:
                key: |
                  -----BEGIN PUBLIC KEY-----
                  MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAxdil+UiTSKYKV90YkeZ/
                  9CWvb4XfUgqYDeW/OG1Le+/BvSVkAFc1s3Fg0l9Zo4yvS4OGQszsNGJNl1mYya/L
                  sSTTD7suKLXY7FBTaBB8CQvvW873yRij1F4EaygOJ1ujuFmpgBGvQLSS5rceNaCl
                  Qzo+bmf3z0UGbhCxgsjDsJK2/aW2D/3dep/kF1GyEOGz8fewnwVp0zVyuS4UMidV
                  2cdnAobX2GvPwpjAeIeqhHG3HX4fen+TwU8rkej3y4efKHNj/GbKQmtt2EoOhEox
                  iK/JALlhQNAJqYn+81amRM7wGWeLEByt0+mwyAfnShOr6MFwrhQjsm4orwAx7yHB
                  jwIDAQAB
                  -----END PUBLIC KEY-----
      ratelimit:
        rateLimits:
          # generate descriptors for Rule 1, 2, 4
          - actions:
              - requestHeaders:
                  descriptorKey: type
                  headerName: x-type
          # generate descriptors for Rule 3
          - actions:
              - requestHeaders:
                  descriptorKey: type
                  headerName: x-type
              - requestHeaders:
                  descriptorKey: number
                  headerName: x-number

```

This configuration tells Envoy to find the JWT in the `x-token` header or `token` query parameter, verify
it (in this case, using the local public key that we generated), and then update the request with the `x-type`
and `x-number` headers extracted from the `type` and `header` claims respectively. 

```
kubectl apply -f vs-petstore-4.yaml
```

Let's confirm this configuration was applied by curling the API. We should get a `401 Unauthorized` response:

``` 
➜ curl $(glooctl proxy url)/sample-route-1 -v
*   Trying 35.227.127.150...
* TCP_NODELAY set
* Connected to 35.227.127.150 (35.227.127.150) port 80 (#0)
> GET /sample-route-1 HTTP/1.1
> Host: 35.227.127.150
> User-Agent: curl/7.54.0
> Accept: */*
>
< HTTP/1.1 401 Unauthorized
< content-length: 14
< content-type: text/plain
< date: Tue, 14 Apr 2020 20:36:40 GMT
< server: envoy
<
* Connection #0 to host 35.227.127.150 left intact
Jwt is missing%
```

We can use [jwt.io](https://jwt.io) to help us construct valid JWTs, using the public and private keys 
we generated, in order to make sure the rate limits are still applied.

First, we can make a JWT that has the claims `type: Messenger` and `number: 311` and we can see the 
request rate limited starting on the third try:

```
➜ curl $(glooctl proxy url)/sample-route-1 -H "x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJNZXNzZW5nZXIiLCJudW1iZXIiOiIzMTEifQ.svbQgUcAUuKHlf6U8in0O3DPGuAIQqgsPv83UIoof1ZnTjOdidqhC-i1p94bLzt67NW5NU_GICZNJU21ZRL3Dmb2ZU8Ee6t708S9rBq3z6hvHt_H-2LuYOfEmj44GqHmwAQm47p4xCaL-3DCZuoFpGUJkB6YCEf5p-r-iWYe76W7WXLqA9LJwmcnZDgasLGlFuf0sTjDzD2-dilFQhY-QFLhQ7iHjmSA6-DHqd021EhsiSrs-pb9Br9e7t39QmUqZM13SMi0VA19oyK6ORNF8zndntPf2KJ2y5M7Pf8tUi2eKTkTA_CpTjFrbsY5KsehA4V1lt-Z4QDukiVtXgSMmg"
[{"id":1,"name":"Dog","status":"available"},{"id":2,"name":"Cat","status":"pending"}]

➜ curl $(glooctl proxy url)/sample-route-1 -H "x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJNZXNzZW5nZXIiLCJudW1iZXIiOiIzMTEifQ.svbQgUcAUuKHlf6U8in0O3DPGuAIQqgsPv83UIoof1ZnTjOdidqhC-i1p94bLzt67NW5NU_GICZNJU21ZRL3Dmb2ZU8Ee6t708S9rBq3z6hvHt_H-2LuYOfEmj44GqHmwAQm47p4xCaL-3DCZuoFpGUJkB6YCEf5p-r-iWYe76W7WXLqA9LJwmcnZDgasLGlFuf0sTjDzD2-dilFQhY-QFLhQ7iHjmSA6-DHqd021EhsiSrs-pb9Br9e7t39QmUqZM13SMi0VA19oyK6ORNF8zndntPf2KJ2y5M7Pf8tUi2eKTkTA_CpTjFrbsY5KsehA4V1lt-Z4QDukiVtXgSMmg"
[{"id":1,"name":"Dog","status":"available"},{"id":2,"name":"Cat","status":"pending"}]

➜ curl $(glooctl proxy url)/sample-route-1 -H "x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJNZXNzZW5nZXIiLCJudW1iZXIiOiIzMTEifQ.svbQgUcAUuKHlf6U8in0O3DPGuAIQqgsPv83UIoof1ZnTjOdidqhC-i1p94bLzt67NW5NU_GICZNJU21ZRL3Dmb2ZU8Ee6t708S9rBq3z6hvHt_H-2LuYOfEmj44GqHmwAQm47p4xCaL-3DCZuoFpGUJkB6YCEf5p-r-iWYe76W7WXLqA9LJwmcnZDgasLGlFuf0sTjDzD2-dilFQhY-QFLhQ7iHjmSA6-DHqd021EhsiSrs-pb9Br9e7t39QmUqZM13SMi0VA19oyK6ORNF8zndntPf2KJ2y5M7Pf8tUi2eKTkTA_CpTjFrbsY5KsehA4V1lt-Z4QDukiVtXgSMmg" -v
*   Trying 35.227.127.150...
* TCP_NODELAY set
* Connected to 35.227.127.150 (35.227.127.150) port 80 (#0)
> GET /sample-route-1 HTTP/1.1
> Host: 35.227.127.150
> User-Agent: curl/7.54.0
> Accept: */*
> x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJNZXNzZW5nZXIiLCJudW1iZXIiOiIzMTEifQ.svbQgUcAUuKHlf6U8in0O3DPGuAIQqgsPv83UIoof1ZnTjOdidqhC-i1p94bLzt67NW5NU_GICZNJU21ZRL3Dmb2ZU8Ee6t708S9rBq3z6hvHt_H-2LuYOfEmj44GqHmwAQm47p4xCaL-3DCZuoFpGUJkB6YCEf5p-r-iWYe76W7WXLqA9LJwmcnZDgasLGlFuf0sTjDzD2-dilFQhY-QFLhQ7iHjmSA6-DHqd021EhsiSrs-pb9Br9e7t39QmUqZM13SMi0VA19oyK6ORNF8zndntPf2KJ2y5M7Pf8tUi2eKTkTA_CpTjFrbsY5KsehA4V1lt-Z4QDukiVtXgSMmg
>
< HTTP/1.1 429 Too Many Requests
< x-envoy-ratelimited: true
< date: Tue, 14 Apr 2020 20:41:56 GMT
< server: envoy
< content-length: 0
<
* Connection #0 to host 35.227.127.150 left intact
``` 

Next, let's test a JWT with `type: Whatsapp` and `number: 311`, which will get rate limited on the second 
request:

```
➜ curl $(glooctl proxy url)/sample-route-1 -H "x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJXaGF0c2FwcCIsIm51bWJlciI6IjMxMSJ9.HpZKZZ6NG9Zy8R7FD87G7A6bY03xiHSub1NqADC7uCGJZM5k6Rvk4_AcKsHYrGIlSIONoPxv63gvEuesPtqP1KseBrjuNDYJ9hmgAS6E-s8IGcxhL4h5Urm_GWBlAOZbnYRBv26spEqbkpPMttmbne4mq8K8najlMMO2WbLXO0G3XSau--HTyy28rBCNrww1Nz-94Rv4brnka4rGgTb8262Qz-CJZDqhenzT9OSIkUcDTA9EkC1b3sJ_fMB1w06yzW2Ey5SCAaByf6ARtJfApmZwC6dOOlgvBw7NJQFnXOHl22r-_1gRanT2xOzWsAHjSdQjNW1ohIjyiDqrlnCKEg" -v
[{"id":1,"name":"Dog","status":"available"},{"id":2,"name":"Cat","status":"pending"}]

➜ curl $(glooctl proxy url)/sample-route-1 -H "x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJXaGF0c2FwcCIsIm51bWJlciI6IjMxMSJ9.HpZKZZ6NG9Zy8R7FD87G7A6bY03xiHSub1NqADC7uCGJZM5k6Rvk4_AcKsHYrGIlSIONoPxv63gvEuesPtqP1KseBrjuNDYJ9hmgAS6E-s8IGcxhL4h5Urm_GWBlAOZbnYRBv26spEqbkpPMttmbne4mq8K8najlMMO2WbLXO0G3XSau--HTyy28rBCNrww1Nz-94Rv4brnka4rGgTb8262Qz-CJZDqhenzT9OSIkUcDTA9EkC1b3sJ_fMB1w06yzW2Ey5SCAaByf6ARtJfApmZwC6dOOlgvBw7NJQFnXOHl22r-_1gRanT2xOzWsAHjSdQjNW1ohIjyiDqrlnCKEg" -v
*   Trying 35.227.127.150...
* TCP_NODELAY set
* Connected to 35.227.127.150 (35.227.127.150) port 80 (#0)
> GET /sample-route-1 HTTP/1.1
> Host: 35.227.127.150
> User-Agent: curl/7.54.0
> Accept: */*
> x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJXaGF0c2FwcCIsIm51bWJlciI6IjMxMSJ9.HpZKZZ6NG9Zy8R7FD87G7A6bY03xiHSub1NqADC7uCGJZM5k6Rvk4_AcKsHYrGIlSIONoPxv63gvEuesPtqP1KseBrjuNDYJ9hmgAS6E-s8IGcxhL4h5Urm_GWBlAOZbnYRBv26spEqbkpPMttmbne4mq8K8najlMMO2WbLXO0G3XSau--HTyy28rBCNrww1Nz-94Rv4brnka4rGgTb8262Qz-CJZDqhenzT9OSIkUcDTA9EkC1b3sJ_fMB1w06yzW2Ey5SCAaByf6ARtJfApmZwC6dOOlgvBw7NJQFnXOHl22r-_1gRanT2xOzWsAHjSdQjNW1ohIjyiDqrlnCKEg
>
< HTTP/1.1 429 Too Many Requests
< x-envoy-ratelimited: true
< date: Tue, 14 Apr 2020 20:43:30 GMT
< server: envoy
< content-length: 0
<
* Connection #0 to host 35.227.127.150 left intact
```

Next, let's test a JWT with `type: Whatsapp` and `number: 411`, which should still get through due to our nested rule:

```
➜ curl $(glooctl proxy url)/sample-route-1 -H "x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJXaGF0c2FwcCIsIm51bWJlciI6IjQxMSJ9.nKxJufSAaW7FcM5qhUVXicn55n5tUCwVHElsnE_EfTYjveAbt7VytcrnihFZctUacrK4XguXb3HPbkb4rQ5wuS2BXoJLNJSao_9N9XtTMabGnpBp9M88dUQ7D-H2nAp-ufcbcQntl5B-gVzTcKwuWckiiMS60gdDMJ2MVcqXskeuftGGt8-Qyygi5NV5eHrlVx6I3McsBkwaw1mxgBEDhMPkgM3PTAcwfihJMdO9T25wY4APwuGB2bTyZyJ86L6xRvu-yMVHS5HouEQY--Xp-AMCbJW1Da-tyCJRBUqw8HIGEOp9wIjPNcPvZ5AZkQ1kvseSVBvtRX-QJXlHBHU6Og"
[{"id":1,"name":"Dog","status":"available"},{"id":2,"name":"Cat","status":"pending"}]
```

Finally, let's make sure requests of a different type (we'll use `type: SMS` and `number: 200`) are rate limited with our 
fallback rule:

``` 
➜ curl $(glooctl proxy url)/sample-route-1 -H "x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJTTVMiLCJudW1iZXIiOiIyMDAifQ.quxs99EylhY2Eod3Ns-NkGRAVbM3riZLQLaCHvPPcpeTn7fEmcATPL82rZvUENLX6nsj_FXetd5dpvAJwPTCTRFhnEmVlK6J9i46nNqlA2JAFwXTww4WlrrpoD6p1fGoq5cGqzqdNBrfK-uee1w5N-c5de3waLAQXK7W6_x-L-0ovAqb0wz4i-fIcTKHGELpReGCh762rrj_iMuwaZMg3SJmIfSbGB7SFfdCcY1kE8fTdwZayoxzG1EzeNFTHd7D-h1Y_odafi_PGn5zwkpU4NkBqTcPx2TbZCS5QPG9VjSgWIi5cWW1tQiPyuv7UOmjgmgZFbXXG-Uf_SBpPZdUhg"
[{"id":1,"name":"Dog","status":"available"},{"id":2,"name":"Cat","status":"pending"}]

➜ curl $(glooctl proxy url)/sample-route-1 -H "x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJTTVMiLCJudW1iZXIiOiIyMDAifQ.quxs99EylhY2Eod3Ns-NkGRAVbM3riZLQLaCHvPPcpeTn7fEmcATPL82rZvUENLX6nsj_FXetd5dpvAJwPTCTRFhnEmVlK6J9i46nNqlA2JAFwXTww4WlrrpoD6p1fGoq5cGqzqdNBrfK-uee1w5N-c5de3waLAQXK7W6_x-L-0ovAqb0wz4i-fIcTKHGELpReGCh762rrj_iMuwaZMg3SJmIfSbGB7SFfdCcY1kE8fTdwZayoxzG1EzeNFTHd7D-h1Y_odafi_PGn5zwkpU4NkBqTcPx2TbZCS5QPG9VjSgWIi5cWW1tQiPyuv7UOmjgmgZFbXXG-Uf_SBpPZdUhg" -v
*   Trying 35.227.127.150...
* TCP_NODELAY set
* Connected to 35.227.127.150 (35.227.127.150) port 80 (#0)
> GET /sample-route-1 HTTP/1.1
> Host: 35.227.127.150
> User-Agent: curl/7.54.0
> Accept: */*
> x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJTTVMiLCJudW1iZXIiOiIyMDAifQ.quxs99EylhY2Eod3Ns-NkGRAVbM3riZLQLaCHvPPcpeTn7fEmcATPL82rZvUENLX6nsj_FXetd5dpvAJwPTCTRFhnEmVlK6J9i46nNqlA2JAFwXTww4WlrrpoD6p1fGoq5cGqzqdNBrfK-uee1w5N-c5de3waLAQXK7W6_x-L-0ovAqb0wz4i-fIcTKHGELpReGCh762rrj_iMuwaZMg3SJmIfSbGB7SFfdCcY1kE8fTdwZayoxzG1EzeNFTHd7D-h1Y_odafi_PGn5zwkpU4NkBqTcPx2TbZCS5QPG9VjSgWIi5cWW1tQiPyuv7UOmjgmgZFbXXG-Uf_SBpPZdUhg
>
< HTTP/1.1 429 Too Many Requests
< x-envoy-ratelimited: true
< date: Tue, 14 Apr 2020 20:47:47 GMT
< server: envoy
< content-length: 0
<
* Connection #0 to host 35.227.127.150 left intact
```

Great! Now we have the complex rate limiting semantics that we want, implemented safely by verifying 
a JWT and then extracting JWT claims into headers, which are then used as before to determine if a request
should be rate limited. 

## Blocking unwanted traffic with Web Application Firewall (WAF)

Now that we've started to lock down our APIs, we may start to grow concerned about the potential for 
bots, spammers, hackers, or other malicious users flooding our APIs with traffic, degrading the performance 
of Envoy, and causing problems for our users. 

A Web Application Firewall, or WAF, is commonly used in front of an API gateway in order to help shield
platforms from this concern. Fortunately, Enterprise Gloo comes with a built-in WAF, built on top of the open 
source [modsecurity](https://modsecurity.org/) WAF library in the Apache product. Modsecurity has a powerful
and customizable rule set that can cover a wide variety of firewall use cases. 

Let's enable a simple WAF rule on our API, blocking traffic with the User-Agent "scammer". In a future 
post, we can do a deeper dive on ways you may want to configure the WAF filter for production use cases. 

To turn on this WAF rule, we just add a basic `waf` section to the options on our virtual service, and define our rule inline. 

```yaml
apiVersion: gateway.solo.io/v1
kind: VirtualService
metadata:
  name: petstore
  namespace: gloo-system
spec:
  virtualHost:
    domains:
      - '*'
    routes:
      - matchers:
          - exact: /sample-route-1
        routeAction:
          single:
            upstream:
              name: default-petstore-8080
              namespace: gloo-system
        options:
          prefixRewrite: /api/pets
    options:
      jwt:
        providers:
          solo:
            tokenSource:
              headers:
                - header: x-token
              queryParams:
                - token
            claimsToHeaders:
              - claim: type
                header: x-type
              - claim: number
                header: x-number
            issuer: solo.io
            jwks:
              local:
                key: |
                  -----BEGIN PUBLIC KEY-----
                  MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAxdil+UiTSKYKV90YkeZ/
                  9CWvb4XfUgqYDeW/OG1Le+/BvSVkAFc1s3Fg0l9Zo4yvS4OGQszsNGJNl1mYya/L
                  sSTTD7suKLXY7FBTaBB8CQvvW873yRij1F4EaygOJ1ujuFmpgBGvQLSS5rceNaCl
                  Qzo+bmf3z0UGbhCxgsjDsJK2/aW2D/3dep/kF1GyEOGz8fewnwVp0zVyuS4UMidV
                  2cdnAobX2GvPwpjAeIeqhHG3HX4fen+TwU8rkej3y4efKHNj/GbKQmtt2EoOhEox
                  iK/JALlhQNAJqYn+81amRM7wGWeLEByt0+mwyAfnShOr6MFwrhQjsm4orwAx7yHB
                  jwIDAQAB
                  -----END PUBLIC KEY-----
      ratelimit:
        rateLimits:
          # generate descriptors for Rule 1, 2, 4
          - actions:
              - requestHeaders:
                  descriptorKey: type
                  headerName: x-type
          # generate descriptors for Rule 3
          - actions:
              - requestHeaders:
                  descriptorKey: type
                  headerName: x-type
              - requestHeaders:
                  descriptorKey: number
                  headerName: x-number
      waf:
        ruleSets:
          - ruleStr: |
              # Turn rule engine on
              SecRuleEngine On
              SecRule REQUEST_HEADERS:User-Agent "scammer" "deny,status:403,id:107,phase:1,msg:'blocked scammer'"

```

We can apply this with the following command: 

```
kubectl apply -f vs-petstore-5.yaml
```

Now, let's try sending a request to the proxy:

```
➜ curl $(glooctl proxy url)/sample-route-1 -H "x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJXaGF0c2FwcCIsIm51bWJlciI6IjQxMSJ9.nKxJufSAaW7FcM5qhUVXicn55n5tUCwVHElsnE_EfTYjveAbt7VytcrnihFZctUacrK4XguXb3HPbkb4rQ5wuS2BXoJLNJSao_9N9XtTMabGnpBp9M88dUQ7D-H2nAp-ufcbcQntl5B-gVzTcKwuWckiiMS60gdDMJ2MVcqXskeuftGGt8-Qyygi5NV5eHrlVx6I3McsBkwaw1mxgBEDhMPkgM3PTAcwfihJMdO9T25wY4APwuGB2bTyZyJ86L6xRvu-yMVHS5HouEQY--Xp-AMCbJW1Da-tyCJRBUqw8HIGEOp9wIjPNcPvZ5AZkQ1kvseSVBvtRX-QJXlHBHU6Og"
[{"id":1,"name":"Dog","status":"available"},{"id":2,"name":"Cat","status":"pending"}]
```

If we add a header `User-Agent: scammer`, we will see a `403 Forbidden` response caused by a modsecurity intervention:

```
➜ curl $(glooctl proxy url)/sample-route-1 -H "x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJXaGF0c2FwcCIsIm51bWJlciI6IjQxMSJ9.nKxJufSAaW7FcM5qhUVXicn55n5tUCwVHElsnE_EfTYjveAbt7VytcrnihFZctUacrK4XguXb3HPbkb4rQ5wuS2BXoJLNJSao_9N9XtTMabGnpBp9M88dUQ7D-H2nAp-ufcbcQntl5B-gVzTcKwuWckiiMS60gdDMJ2MVcqXskeuftGGt8-Qyygi5NV5eHrlVx6I3McsBkwaw1mxgBEDhMPkgM3PTAcwfihJMdO9T25wY4APwuGB2bTyZyJ86L6xRvu-yMVHS5HouEQY--Xp-AMCbJW1Da-tyCJRBUqw8HIGEOp9wIjPNcPvZ5AZkQ1kvseSVBvtRX-QJXlHBHU6Og" -H "User-Agent: scammer"
ModSecurity: intervention occured
```

We could expand this rule set to protect against more types of suspicious incoming traffic. 

## Limiting authorization using Open Policy Agent

So far, our API will authorize any valid JWT, and will only restrict access if the incoming requests 
trigger a rate limit. We've also added a WAF in front of the gateway to block malicious traffic. 

However, there are a few ways that we still need to tighten up security. The first thing we might 
discover is that requests that don't contain a `type` claim in the JWT are not currently blocked. We 
could add a rate limit rule for this, but we actually want to consider this request invalid and thus
unauthorized. 

Let's try that out quickly by creating a new JWT that is missing a type:

``` 
➜ curl $(glooctl proxy url)/sample-route-1 -H "x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsIm51bWJlciI6IjIwMCJ9.fg3CoYao9ba9etH8koAnaE6grrWorxt-TPvgjKR4vAKnGFnP-UKt3-Dgxmwf5YeogYsIX_YCxngBR-PMgR0ruS0JyraogwritWZRHJb8M032zWDWzVMZBi9OeYXrxsWx756VQtJafKfPubHIFYP9DXtrbg2fb9aNhsm_nfWWNCPJ7agAyCjSqw72niJKevvtSyj0jjWsIsxzvm7FMGUD5j59puRMue6LQibibFqqG7Cbfc4XT9jE1ByV3sVWR1m3iglFuftCp3EkSS0KYQZkXZJPCimNx4onVtZ6IiC6Qn_MFpDfkeJvA0khoBbuJTnkm8HpA8AXaZGY2mTZ8vMn5A"
[{"id":1,"name":"Dog","status":"available"},{"id":2,"name":"Cat","status":"pending"}]
```

We want to block this request with an authorization check. 
Similarly, we may not have a valid use case for `SMS` requests to our API, so we will consider requests
with `type: SMS` unauthorized. 

We can implement these rules -- and much more -- by taking advantage of Gloo's native integration with 
Open Policy Agent (OPA) in the external auth server, and configuring a simple authorization policy for the route. 

First, we'll write a `rego` script, which is OPA's language for defining policy. 

```yaml
apiVersion: v1
data:
  opa-policy.rego: |-
    package test
    default allow = false
    allow {
        [header, payload, signature] = io.jwt.decode(input.http_request.headers["x-token"])
        trace(sprintf("%v", [payload]))
        [sms] = payload["type"] == "SMS"
        [unknown] = payload["type"] == ""
        !sms && !unknown
    }
kind: ConfigMap
metadata:
  name: allow-jwt
  namespace: gloo-system

```

Our rego script is very simple. It decodes the JWT, extracts the type from the payload, and then 
returns true if the type is non-empty and doesn't equal "SMS". 

Next, we write an `AuthConfig` resource to define the OPA authorization step and reference the configuration 
we just created. 

```yaml
apiVersion: enterprise.gloo.solo.io/v1
kind: AuthConfig
metadata:
  name: opa-auth
  namespace: gloo-system
spec:
  configs:
    - opa_auth:
        modules:
          - name: allow-jwt
            namespace: gloo-system
        query: "data.test.allow == true"
```

Finally, we reference this `AuthConfig` from our virtual service:

```yaml
apiVersion: gateway.solo.io/v1
kind: VirtualService
metadata:
  name: petstore
  namespace: gloo-system
spec:
  virtualHost:
    domains:
      - '*'
    routes:
      - matchers:
          - exact: /sample-route-1
        routeAction:
          single:
            upstream:
              name: default-petstore-8080
              namespace: gloo-system
        options:
          prefixRewrite: /api/pets
    options:
      jwt:
        providers:
          solo:
            tokenSource:
              headers:
                - header: x-token
              queryParams:
                - token
            claimsToHeaders:
              - claim: type
                header: x-type
              - claim: number
                header: x-number
            issuer: solo.io
            jwks:
              local:
                key: |
                  -----BEGIN PUBLIC KEY-----
                  MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAxdil+UiTSKYKV90YkeZ/
                  9CWvb4XfUgqYDeW/OG1Le+/BvSVkAFc1s3Fg0l9Zo4yvS4OGQszsNGJNl1mYya/L
                  sSTTD7suKLXY7FBTaBB8CQvvW873yRij1F4EaygOJ1ujuFmpgBGvQLSS5rceNaCl
                  Qzo+bmf3z0UGbhCxgsjDsJK2/aW2D/3dep/kF1GyEOGz8fewnwVp0zVyuS4UMidV
                  2cdnAobX2GvPwpjAeIeqhHG3HX4fen+TwU8rkej3y4efKHNj/GbKQmtt2EoOhEox
                  iK/JALlhQNAJqYn+81amRM7wGWeLEByt0+mwyAfnShOr6MFwrhQjsm4orwAx7yHB
                  jwIDAQAB
                  -----END PUBLIC KEY-----
      ratelimit:
        rateLimits:
          # generate descriptors for Rule 1, 2, 4
          - actions:
              - requestHeaders:
                  descriptorKey: type
                  headerName: x-type
          # generate descriptors for Rule 3
          - actions:
              - requestHeaders:
                  descriptorKey: type
                  headerName: x-type
              - requestHeaders:
                  descriptorKey: number
                  headerName: x-number
      waf:
        ruleSets:
          - ruleStr: |
              # Turn rule engine on
              SecRuleEngine On
              SecRule REQUEST_HEADERS:User-Agent "scammer" "deny,status:403,id:107,phase:1,msg:'blocked scammer'"
      extauth:
        configRef:
          name: opa-auth
          namespace: gloo-system

```

We can apply these resources with the following commands:

```
kubectl apply -f allow-jwt.yaml
```

```
kubectl apply -f auth-config.yaml
```

```
kubectl apply -f vs-petstore-6.yaml
```

Now let's issue the curl request from before, with no `type` claim in the JWT:

``` 
➜ curl $(glooctl proxy url)/sample-route-1 -H "x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsIm51bWJlciI6IjIwMCJ9.fg3CoYao9ba9etH8koAnaE6grrWorxt-TPvgjKR4vAKnGFnP-UKt3-Dgxmwf5YeogYsIX_YCxngBR-PMgR0ruS0JyraogwritWZRHJb8M032zWDWzVMZBi9OeYXrxsWx756VQtJafKfPubHIFYP9DXtrbg2fb9aNhsm_nfWWNCPJ7agAyCjSqw72niJKevvtSyj0jjWsIsxzvm7FMGUD5j59puRMue6LQibibFqqG7Cbfc4XT9jE1ByV3sVWR1m3iglFuftCp3EkSS0KYQZkXZJPCimNx4onVtZ6IiC6Qn_MFpDfkeJvA0khoBbuJTnkm8HpA8AXaZGY2mTZ8vMn5A"
*   Trying 35.227.127.150...
* TCP_NODELAY set
* Connected to 35.227.127.150 (35.227.127.150) port 80 (#0)
> GET /sample-route-1 HTTP/1.1
> Host: 35.227.127.150
> User-Agent: curl/7.54.0
> Accept: */*
> x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsIm51bWJlciI6IjIwMCJ9.fg3CoYao9ba9etH8koAnaE6grrWorxt-TPvgjKR4vAKnGFnP-UKt3-Dgxmwf5YeogYsIX_YCxngBR-PMgR0ruS0JyraogwritWZRHJb8M032zWDWzVMZBi9OeYXrxsWx756VQtJafKfPubHIFYP9DXtrbg2fb9aNhsm_nfWWNCPJ7agAyCjSqw72niJKevvtSyj0jjWsIsxzvm7FMGUD5j59puRMue6LQibibFqqG7Cbfc4XT9jE1ByV3sVWR1m3iglFuftCp3EkSS0KYQZkXZJPCimNx4onVtZ6IiC6Qn_MFpDfkeJvA0khoBbuJTnkm8HpA8AXaZGY2mTZ8vMn5A
>
< HTTP/1.1 403 Forbidden
< date: Tue, 14 Apr 2020 21:36:12 GMT
< server: envoy
< content-length: 0
<
* Connection #0 to host 35.227.127.150 left intact
```

As we expect, this is now forbidden. And if we issue a request with `type: SMS`, we'll also see that blocked:

``` 
➜ curl $(glooctl proxy url)/sample-route-1 -H "x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJTTVMiLCJudW1iZXIiOiIyMDAifQ.quxs99EylhY2Eod3Ns-NkGRAVbM3riZLQLaCHvPPcpeTn7fEmcATPL82rZvUENLX6nsj_FXetd5dpvAJwPTCTRFhnEmVlK6J9i46nNqlA2JAFwXTww4WlrrpoD6p1fGoq5cGqzqdNBrfK-uee1w5N-c5de3waLAQXK7W6_x-L-0ovAqb0wz4i-fIcTKHGELpReGCh762rrj_iMuwaZMg3SJmIfSbGB7SFfdCcY1kE8fTdwZayoxzG1EzeNFTHd7D-h1Y_odafi_PGn5zwkpU4NkBqTcPx2TbZCS5QPG9VjSgWIi5cWW1tQiPyuv7UOmjgmgZFbXXG-Uf_SBpPZdUhg" -v
*   Trying 35.227.127.150...
* TCP_NODELAY set
* Connected to 35.227.127.150 (35.227.127.150) port 80 (#0)
> GET /sample-route-1 HTTP/1.1
> Host: 35.227.127.150
> User-Agent: curl/7.54.0
> Accept: */*
> x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJTTVMiLCJudW1iZXIiOiIyMDAifQ.quxs99EylhY2Eod3Ns-NkGRAVbM3riZLQLaCHvPPcpeTn7fEmcATPL82rZvUENLX6nsj_FXetd5dpvAJwPTCTRFhnEmVlK6J9i46nNqlA2JAFwXTww4WlrrpoD6p1fGoq5cGqzqdNBrfK-uee1w5N-c5de3waLAQXK7W6_x-L-0ovAqb0wz4i-fIcTKHGELpReGCh762rrj_iMuwaZMg3SJmIfSbGB7SFfdCcY1kE8fTdwZayoxzG1EzeNFTHd7D-h1Y_odafi_PGn5zwkpU4NkBqTcPx2TbZCS5QPG9VjSgWIi5cWW1tQiPyuv7UOmjgmgZFbXXG-Uf_SBpPZdUhg
>
< HTTP/1.1 403 Forbidden
< date: Tue, 14 Apr 2020 21:38:15 GMT
< server: envoy
< content-length: 0
<
* Connection #0 to host 35.227.127.150 left intact
``` 

With OPA, we can now enforce fine-grained authorization to this API in the gateway. Gloo makes it easy 
to iterate on the OPA policies by logging the OPA traces at debug level, but we'll save a deep dive on 
OPA policies for a future post. 

## Route level configuration

Let's say we add a new API to our virtual service, and we want to apply the same security checks 
to it, but we don't want the same rate limit actions. We can move the rate limiting configuration 
we created in this example to the route, and leave the rest of the virtual service intact. Then we 
can add another route without rate limiting enabled. 

```yaml
apiVersion: gateway.solo.io/v1
kind: VirtualService
metadata:
  name: petstore
  namespace: gloo-system
spec:
  virtualHost:
    domains:
      - '*'
    routes:
      - matchers:
          - exact: /sample-route-2
        routeAction:
          single:
            upstream:
              name: default-petstore-8080
              namespace: gloo-system
        options:
          prefixRewrite: /api/pets
      - matchers:
          - exact: /sample-route-1
        routeAction:
          single:
            upstream:
              name: default-petstore-8080
              namespace: gloo-system
        options:
          prefixRewrite: /api/pets
          ratelimit:
            rateLimits:
              # generate descriptors for Rule 1, 2, 4
              - actions:
                  - requestHeaders:
                      descriptorKey: type
                      headerName: x-type
              # generate descriptors for Rule 3
              - actions:
                  - requestHeaders:
                      descriptorKey: type
                      headerName: x-type
                  - requestHeaders:
                      descriptorKey: number
                      headerName: x-number
    options:
      jwt:
        providers:
          solo:
            tokenSource:
              headers:
                - header: x-token
              queryParams:
                - token
            claimsToHeaders:
              - claim: type
                header: x-type
              - claim: number
                header: x-number
            issuer: solo.io
            jwks:
              local:
                key: |
                  -----BEGIN PUBLIC KEY-----
                  MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAxdil+UiTSKYKV90YkeZ/
                  9CWvb4XfUgqYDeW/OG1Le+/BvSVkAFc1s3Fg0l9Zo4yvS4OGQszsNGJNl1mYya/L
                  sSTTD7suKLXY7FBTaBB8CQvvW873yRij1F4EaygOJ1ujuFmpgBGvQLSS5rceNaCl
                  Qzo+bmf3z0UGbhCxgsjDsJK2/aW2D/3dep/kF1GyEOGz8fewnwVp0zVyuS4UMidV
                  2cdnAobX2GvPwpjAeIeqhHG3HX4fen+TwU8rkej3y4efKHNj/GbKQmtt2EoOhEox
                  iK/JALlhQNAJqYn+81amRM7wGWeLEByt0+mwyAfnShOr6MFwrhQjsm4orwAx7yHB
                  jwIDAQAB
                  -----END PUBLIC KEY-----
      waf:
        ruleSets:
          - ruleStr: |
              # Turn rule engine on
              SecRuleEngine On
              SecRule REQUEST_HEADERS:User-Agent "scammer" "deny,status:403,id:107,phase:1,msg:'blocked scammer'"
      extauth:
        configRef:
          name: opa-auth
          namespace: gloo-system

```

And deploy it to the cluster:

```
kubectl apply -f vs-petstore-7.yaml
```

Now we can see requests to `/sample-route-1` rate limited, while requests to `/sample-route-2` are not. 

``` 
➜ curl $(glooctl proxy url)/sample-route-1 -H "x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJXaGF0c2FwcCIsIm51bWJlciI6IjMxMSJ9.HpZKZZ6NG9Zy8R7FD87G7A6bY03xiHSub1NqADC7uCGJZM5k6Rvk4_AcKsHYrGIlSIONoPxv63gvEuesPtqP1KseBrjuNDYJ9hmgAS6E-s8IGcxhL4h5Urm_GWBlAOZbnYRBv26spEqbkpPMttmbne4mq8K8najlMMO2WbLXO0G3XSau--HTyy28rBCNrww1Nz-94Rv4brnka4rGgTb8262Qz-CJZDqhenzT9OSIkUcDTA9EkC1b3sJ_fMB1w06yzW2Ey5SCAaByf6ARtJfApmZwC6dOOlgvBw7NJQFnXOHl22r-_1gRanT2xOzWsAHjSdQjNW1ohIjyiDqrlnCKEg" -v
[{"id":1,"name":"Dog","status":"available"},{"id":2,"name":"Cat","status":"pending"}]

➜ curl $(glooctl proxy url)/sample-route-1 -H "x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJXaGF0c2FwcCIsIm51bWJlciI6IjMxMSJ9.HpZKZZ6NG9Zy8R7FD87G7A6bY03xiHSub1NqADC7uCGJZM5k6Rvk4_AcKsHYrGIlSIONoPxv63gvEuesPtqP1KseBrjuNDYJ9hmgAS6E-s8IGcxhL4h5Urm_GWBlAOZbnYRBv26spEqbkpPMttmbne4mq8K8najlMMO2WbLXO0G3XSau--HTyy28rBCNrww1Nz-94Rv4brnka4rGgTb8262Qz-CJZDqhenzT9OSIkUcDTA9EkC1b3sJ_fMB1w06yzW2Ey5SCAaByf6ARtJfApmZwC6dOOlgvBw7NJQFnXOHl22r-_1gRanT2xOzWsAHjSdQjNW1ohIjyiDqrlnCKEg" -v
*   Trying 35.227.127.150...
* TCP_NODELAY set
* Connected to 35.227.127.150 (35.227.127.150) port 80 (#0)
> GET /sample-route-1 HTTP/1.1
> Host: 35.227.127.150
> User-Agent: curl/7.54.0
> Accept: */*
> x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJXaGF0c2FwcCIsIm51bWJlciI6IjMxMSJ9.HpZKZZ6NG9Zy8R7FD87G7A6bY03xiHSub1NqADC7uCGJZM5k6Rvk4_AcKsHYrGIlSIONoPxv63gvEuesPtqP1KseBrjuNDYJ9hmgAS6E-s8IGcxhL4h5Urm_GWBlAOZbnYRBv26spEqbkpPMttmbne4mq8K8najlMMO2WbLXO0G3XSau--HTyy28rBCNrww1Nz-94Rv4brnka4rGgTb8262Qz-CJZDqhenzT9OSIkUcDTA9EkC1b3sJ_fMB1w06yzW2Ey5SCAaByf6ARtJfApmZwC6dOOlgvBw7NJQFnXOHl22r-_1gRanT2xOzWsAHjSdQjNW1ohIjyiDqrlnCKEg
>
< HTTP/1.1 429 Too Many Requests
< x-envoy-ratelimited: true
< date: Tue, 14 Apr 2020 21:45:32 GMT
< server: envoy
< content-length: 0
<
* Connection #0 to host 35.227.127.150 left intact

➜ curl $(glooctl proxy url)/sample-route-2 -H "x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJXaGF0c2FwcCIsIm51bWJlciI6IjMxMSJ9.HpZKZZ6NG9Zy8R7FD87G7A6bY03xiHSub1NqADC7uCGJZM5k6Rvk4_AcKsHYrGIlSIONoPxv63gvEuesPtqP1KseBrjuNDYJ9hmgAS6E-s8IGcxhL4h5Urm_GWBlAOZbnYRBv26spEqbkpPMttmbne4mq8K8najlMMO2WbLXO0G3XSau--HTyy28rBCNrww1Nz-94Rv4brnka4rGgTb8262Qz-CJZDqhenzT9OSIkUcDTA9EkC1b3sJ_fMB1w06yzW2Ey5SCAaByf6ARtJfApmZwC6dOOlgvBw7NJQFnXOHl22r-_1gRanT2xOzWsAHjSdQjNW1ohIjyiDqrlnCKEg"
[{"id":1,"name":"Dog","status":"available"},{"id":2,"name":"Cat","status":"pending"}]

➜ curl $(glooctl proxy url)/sample-route-2 -H "x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJXaGF0c2FwcCIsIm51bWJlciI6IjMxMSJ9.HpZKZZ6NG9Zy8R7FD87G7A6bY03xiHSub1NqADC7uCGJZM5k6Rvk4_AcKsHYrGIlSIONoPxv63gvEuesPtqP1KseBrjuNDYJ9hmgAS6E-s8IGcxhL4h5Urm_GWBlAOZbnYRBv26spEqbkpPMttmbne4mq8K8najlMMO2WbLXO0G3XSau--HTyy28rBCNrww1Nz-94Rv4brnka4rGgTb8262Qz-CJZDqhenzT9OSIkUcDTA9EkC1b3sJ_fMB1w06yzW2Ey5SCAaByf6ARtJfApmZwC6dOOlgvBw7NJQFnXOHl22r-_1gRanT2xOzWsAHjSdQjNW1ohIjyiDqrlnCKEg"
[{"id":1,"name":"Dog","status":"available"},{"id":2,"name":"Cat","status":"pending"}]
```
