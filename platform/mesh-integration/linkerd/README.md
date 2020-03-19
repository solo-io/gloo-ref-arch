

```
linkerd install | k apply -f -
k get deploy -oyaml | linkerd inject - | kubectl apply -f -
kubectl patch settings -n gloo-system default -p '{"spec":{"linkerd":true}}' --type=merge
```