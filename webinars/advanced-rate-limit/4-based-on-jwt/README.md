# Rate limiting based on headers and JWTs

Now we are going to look at how to extract values from headers and JWTs to rate limit. 

## Headers

First, let's understand how headers work. We can apply these settings:

`k patch -n gloo-system settings default --type merge --patch "$(cat settings-patch-1.yaml)"`

And take these rate limit actions on our route: 

`k apply -f vs-1.yaml`

Now we have limits for different types of messages and numbers. We utilized nesting and rule priority to get the exact behavior 
we want based on two different headers. 

curl $(glooctl proxy url)/ -v -H "x-type: Messenger"
curl $(glooctl proxy url)/ -v -H "x-type: Whatsapp" 
curl $(glooctl proxy url)/ -v -H "x-type: Whatsapp" -H "x-number: 411"

However, for other types we aren't rate limited at all:

`curl $(glooctl proxy url)/ -v -H "x-type: SMS"`

Let's fix that with a fallback limit for any type not specified:

```
k patch -n gloo-system settings default --type merge --patch "$(cat settings-patch-2.yaml)"
k apply -f vs-2.yaml
```

Now we should see a limit for our new type, "SMS":

`curl $(glooctl proxy url)/ -v -H "x-type: SMS"`

## Using JWT claims instead

This is the behavior we want, but we want to improve the implementation in two ways. First, we want to only consider
rate limiting for authorized requests, and we want to use JWT tokens as a way to verify authorization. The client 
is required to provide a valid JWT. 

Second, we want to extract the value of type and number from claims in the JWT, rather than the headers directly. This 
shifts the source of truth for these values onto the identity management system, rather than relying on the client to supply 
headers. 

Fortunately, we can leverage JWT verification in Envoy, and a feature called "claims to headers", to enable this. We can update
the virtual service to expect a valid JWT:

`k apply -f vs-2.yaml`

type: Messenger
```
eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiaXNzIjoic29sby5pbyIsImlhdCI6MTUxNjIzOTAyMiwidHlwZSI6Ik1lc3NlbmdlciJ9.GKRZQclSfr57ugPIjRXf2MrEGxUa4r3bgPaBrxq4JwOPQrqMCr3WT2RFvGGps3nbBOzqJb_SDbJ6UjjdOtsaQAAYpvnMlzZLbPnr1naN7lb3ynkWeQbLVm8c3qMwWne2lHZfhxBHe2wyhxeqio7-g21vrUwaBJtugCSO8qZh0_uYRa4oQCOZASTQzW3q6LO9tWpGSer0Th3WlhiqZ2ld7-ifL7wYyfKHn-omigidlqK1aIeC1ItjWqilYFoWeB3KFDSCM-HuHZkCw87S9PaJl6kVDrnbCcyWA7BqdxN62G6A3wz4MLMyusQjEcOlQUB4GDwOiqBSYYSklWO9-Vsxbg
```

type: Whatsapp
```
eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiaXNzIjoic29sby5pbyIsImlhdCI6MTUxNjIzOTAyMiwidHlwZSI6IldoYXRzYXBwIn0.qpGpMtP1D9SUZNrI310becc82Wzv3G8bxXKLz0egnBOt5WNOknZHeEGbeRY6z5PkEbZ7VKpbIzK9dMTsjw_91mn0lvuphDzNqyhXR_AP7F4qKjw9viLqMDRktANkl-lClJSadPbysQqVo0Y5Z4x5p_Fj6FBbTc_tzY7Tcx49cEO2KD8nbPWanOLxiJWHhtTHUtd-z3zLm62uBhINcGYD6-uqs6G6J_qA1kEl9pc55iYQiva2DziELZZjUADsYs7XlGmj0RRJy6tGh-MQhusbWNTmZDwj8YBTpNyeCDxjaDgCOCzpabuKVgHjHDb1QHr-sRA9HLyy-lcTXKeR9zmxWA
```

type: Whatsapp, number: 411
```
eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiaXNzIjoic29sby5pbyIsImlhdCI6MTUxNjIzOTAyMiwidHlwZSI6IldoYXRzYXBwIiwibnVtYmVyIjoiNDExIn0.hL-zwDhSCRtKzsnypCg2mk7KAbUtO9fg0hQ0dWTke_VqLDoZ3oIZtBkWlpZh1XpXW11VtLPwtZ3Ae_ehVxXnd2X_R4dXIoaMb90ewpguEO_JspMrm1ezxF-E27IvKfhRFuuWd-ABkPUI8OizFA0qDC2QEP9p5NVQx8HvEWFjl6DxfdNOZckn60i2ltWf_hhF7PqNuhvUgUjkLhSEzlH1PL_gjlX7Lz6NXNQqRbnlDrtBVlC6D4iJALD4jDgKVGkN_7RXPr0sPOysAtips5_gkLp7QrzKz3LjWze9cIXDvoTLBuPMbDg1gsSwLoErZ0CEiQEnD5aruYR6KHYhAojtuQ
```

type: SMS
```
eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiaXNzIjoic29sby5pbyIsImlhdCI6MTUxNjIzOTAyMiwidHlwZSI6IlNNUyJ9.qvXV0ktsDvib6LGZ4lNmY5K2xbUtC0RSwZRptDyY-jZ7AWFbwWImU5KwkSoDTaB3jHT54ZZPq4ezG5bzPHTKLatJaVf-hrGl4cSqgECRuXzrJnNLGO1yNtQOZsQRzRWl82q0ckF4ZjyFFN-yNT3Fy88dPenIuXHjNqSyjWzjUpB3Mwy0hwbysgnSsp40h5WYi3RCOHJ-jUl7jGPoIc60M8cZ7P88HQ2n8VfSrKPuh8DMzs3ceRP2HeZOGQgYMZdBIVHrQdZg13hFU85iUuFTSvg3HE5XDNqklZabqimlGl53uQMMhMgmicerOnFJ2Rgp4rOkkUArOkVcXesGIJt4sQ
```

## Layering on more security - WAF and OPA

Add the OPA config

`k apply -f allow-jwt.yaml`

Add the auth config:

`k appply -f auth-config.yaml`

Update the VS to enable WAF and OPA auth:

`k apply -f vs-3.yaml`

Now we should see WAF happen:

```
➜ curl $(glooctl proxy url)/ -H "x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiaXNzIjoic29sby5pbyIsImlhdCI6MTUxNjIzOTAyMiwidHlwZSI6IldoYXRzYXBwIiwibnVtYmVyIjoiNDExIn0.hL-zwDhSCRtKzsnypCg2mk7KAbUtO9fg0hQ0dWTke_VqLDoZ3oIZtBkWlpZh1XpXW11VtLPwtZ3Ae_ehVxXnd2X_R4dXIoaMb90ewpguEO_JspMrm1ezxF-E27IvKfhRFuuWd-ABkPUI8OizFA0qDC2QEP9p5NVQx8HvEWFjl6DxfdNOZckn60i2ltWf_hhF7PqNuhvUgUjkLhSEzlH1PL_gjlX7Lz6NXNQqRbnlDrtBVlC6D4iJALD4jDgKVGkN_7RXPr0sPOysAtips5_gkLp7QrzKz3LjWze9cIXDvoTLBuPMbDg1gsSwLoErZ0CEiQEnD5aruYR6KHYhAojtuQ" -H "User-Agent: scammer"
ModSecurity: intervention occurred%
```

And OPA auth will consider SMS messages forbidden:

``` 
➜ curl $(glooctl proxy url)/ -H "x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiaXNzIjoic29sby5pbyIsImlhdCI6MTUxNjIzOTAyMiwidHlwZSI6IlNNUyJ9.qvXV0ktsDvib6LGZ4lNmY5K2xbUtC0RSwZRptDyY-jZ7AWFbwWImU5KwkSoDTaB3jHT54ZZPq4ezG5bzPHTKLatJaVf-hrGl4cSqgECRuXzrJnNLGO1yNtQOZsQRzRWl82q0ckF4ZjyFFN-yNT3Fy88dPenIuXHjNqSyjWzjUpB3Mwy0hwbysgnSsp40h5WYi3RCOHJ-jUl7jGPoIc60M8cZ7P88HQ2n8VfSrKPuh8DMzs3ceRP2HeZOGQgYMZdBIVHrQdZg13hFU85iUuFTSvg3HE5XDNqklZabqimlGl53uQMMhMgmicerOnFJ2Rgp4rOkkUArOkVcXesGIJt4sQ" -v
*   Trying 35.190.154.99...
* TCP_NODELAY set
* Connected to 35.190.154.99 (35.190.154.99) port 80 (#0)
> GET / HTTP/1.1
> Host: 35.190.154.99
> User-Agent: curl/7.54.0
> Accept: */*
> x-token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiaXNzIjoic29sby5pbyIsImlhdCI6MTUxNjIzOTAyMiwidHlwZSI6IlNNUyJ9.qvXV0ktsDvib6LGZ4lNmY5K2xbUtC0RSwZRptDyY-jZ7AWFbwWImU5KwkSoDTaB3jHT54ZZPq4ezG5bzPHTKLatJaVf-hrGl4cSqgECRuXzrJnNLGO1yNtQOZsQRzRWl82q0ckF4ZjyFFN-yNT3Fy88dPenIuXHjNqSyjWzjUpB3Mwy0hwbysgnSsp40h5WYi3RCOHJ-jUl7jGPoIc60M8cZ7P88HQ2n8VfSrKPuh8DMzs3ceRP2HeZOGQgYMZdBIVHrQdZg13hFU85iUuFTSvg3HE5XDNqklZabqimlGl53uQMMhMgmicerOnFJ2Rgp4rOkkUArOkVcXesGIJt4sQ
>
< HTTP/1.1 403 Forbidden
< date: Thu, 25 Jun 2020 18:06:35 GMT
< server: envoy
< content-length: 0
<
* Connection #0 to host 35.190.154.99 left intact
```
