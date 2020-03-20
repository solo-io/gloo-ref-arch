```
k apply -f upstream.yaml
k apply -f vs-1.yaml
curl -v -H "Content-Type: application/json" $(glooctl proxy url)/post -d @data.json | jq

=> json response with 200 and a body containing the error message

k apply -f vs-2.yaml
curl -v -H "Content-Type: application/json" $(glooctl proxy url)/post -d @data.json | jq
=> 400


curl -v -H "Content-Type: application/json" $(glooctl proxy url)/post -d @data-2.json | jq
=> 200

