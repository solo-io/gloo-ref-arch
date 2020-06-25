# Per Client rate limit

## Set up remote address forwarding

To start, the remote address is incorrect:

`curl $(glooctl proxy url)/ -v`

We can make a few changes to set up x-forwarded-for header to be correct:

```
kubectl patch -n gloo-system gateway gateway-proxy --type merge --patch "$(cat gateway-patch.yaml)"
kubectl patch -n gloo-system svc gateway-proxy --type merge --patch "$(cat gateway-proxy-svc-patch.yaml)"
```

Now we can see the remote address is preserved:

`curl $(glooctl proxy url)/ -v`

## Set up basic limit on remote address

We can set up the descriptors in the following patch:

`k patch -n gloo-system settings default --type merge --patch "$(cat settings-patch-1.yaml)"`

For this to take effect, we need to create actions on our route:

`k apply -f vs-1.yaml`

Now we can see the rate limit happen as expected:

`curl $(glooctl proxy url)/ -v`

We can inspect Redis and see the actual counter value for our remote address:

``` 
k port-forward -n gloo-system deploy/redis 6379
redis-cli
scan 0
get ...
```

Let's prove that remote address works as we expect. Let's overload the service:

`while true; do curl $(glooctl proxy url) -v ; done`

Now let's log into a VM with a different IP and see what happens:

``` 
k exec -it -n spelunker deploy/spelunker /bin/sh
curl http://34.74.148.50:80
```

## More complex limits

Let's imagine we want to provide two limits per client - one limit per long duration, and one limit to prevent short
bursts of traffic.

```
k patch -n gloo-system settings default --type merge --patch "$(cat settings-patch-2.yaml)"
k apply -f vs-2.yaml
```

Now we can see both take effect when we send large bursts of traffic:

`while true; do curl $(glooctl proxy url) -v ; done`