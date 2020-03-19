```
curl -L https://istio.io/downloadIstio | sh -
istioctl manifest apply --set profile=demo
kubectl label namespace default istio-injection=enabled

kubectl apply -f samples/bookinfo/platform/kube/bookinfo.yaml

```