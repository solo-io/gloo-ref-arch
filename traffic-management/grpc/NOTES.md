## Native GRPC

- Install

`k apply -f grpcstore.yaml`

- Expose the grpc store demo directly outside of the cluster

`k port-forward deploy/grpcstore-demo 8080`

- List services using `grpc_cli`: 

```bash
grpc_cli ls localhost:8080
```

```
grpc.reflection.v1alpha.ServerReflection
solo.examples.v1.StoreService
```

- List StoreService details:

```bash
grpc_cli ls localhost:8080 solo.examples.v1.StoreService -l
```

```
filename: proto/file.proto
package: solo.examples.v1;
service StoreService {
  rpc CreateItem(solo.examples.v1.CreateItemRequest) returns (solo.examples.v1.CreateItemResponse) {}
  rpc ListItems(solo.examples.v1.ListItemsRequest) returns (solo.examples.v1.ListItemsResponse) {}
  rpc DeleteItem(solo.examples.v1.DeleteItemRequest) returns (solo.examples.v1.DeleteItemResponse) {}
  rpc GetItem(solo.examples.v1.GetItemRequest) returns (solo.examples.v1.GetItemResponse) {}
}
```

- Inspect types:

```bash
grpc_cli type localhost:8080 solo.examples.v1.CreateItemRequest
```

```
message CreateItemRequest {
  .solo.examples.v1.Item item = 1[json_name = "item"];
}

```

- Curl `CreateItem` endpoint:

```bash
echo '{"item":{"name":"item1"}}' | grpcurl -plaintext -d @ localhost:8080 solo.examples.v1.StoreService/CreateItem
```
```json
{
  "item": {
    "name": "item1"
  }
}
```

- Curl `GetItem` endpoint:

```bash
echo '{"name":"item1"}' | grpcurl -plaintext -d @ localhost:8080 solo.examples.v1.StoreService/GetItem
```
```json
{
  "item": {
    "name": "item1"
  }
}
```

- Curl `ListItems` endpoint:

```bash
echo '{"item":{"name":"item2"}}' | grpcurl -plaintext -d @ localhost:8080 solo.examples.v1.StoreService/CreateItem
echo "" | grpcurl -plaintext -d @ localhost:8080 solo.examples.v1.StoreService/ListItems
```
```json
{
  "items": [
    {
      "name": "item1"
    },
    {
      "name": "item2"
    }
  ]
}
```

## Enabling TCP proxy 

`k get us -n gloo-system default-grpcstore-demo-80`

`k apply -f tcp-gateway.yaml`

```bash
echo "" | grpcurl -plaintext -d @ localhost:8000 solo.examples.v1.StoreService/ListItems
```

```json
{
  "items": [
    {
      "name": "item1"
    },
    {
      "name": "item2"
    }
  ]
}
```

## Enabling HTTP proxy with function routing

`kubectl label namespace default discovery.solo.io/function_discovery=enabled`

 `k get us -n gloo-system default-grpcstore-demo-80`
 
 `k apply -f rest-vs.yaml`
 
 - Get the proxy URL:
 
 `glooctl proxy url`
 
 `http://192.168.64.15:32274`
 
 - Curl the route:
 
 `curl http://192.168.64.15:32274`
 
 `{"items":[{"name":"item1"},{"name":"item2"}]}`