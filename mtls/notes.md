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