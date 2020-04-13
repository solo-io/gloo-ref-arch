package main

import (
	"fmt"
	"log"
	"os"

	// "fmt"
	// "io"
	"net/http"
)

func main() {
	certFile := os.Getenv("CERT_FILE")
	if certFile == "" {
		certFile = "valet-test.com.crt"
	}
	keyFile := os.Getenv("KEY_FILE")
	if keyFile == "" {
		keyFile = "valet-test.com.key"
	}


	httpHandler := HttpHandler{}
	httpsHandler := HttpsHandler{}

	go func() {
		err := http.ListenAndServeTLS(":8443", certFile, keyFile, &httpsHandler)
		if err != nil {
			log.Fatal("ListenAndServeTLS: ", err)
		}
	}()
	err := http.ListenAndServe(":8080", &httpHandler)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

type HttpHandler struct {

}

func (h *HttpHandler) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	response := fmt.Sprintf("This is an example http server.\n\n%v\n", request)
	w.Write([]byte(response))
}

type HttpsHandler struct {

}

func (h *HttpsHandler) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	response := fmt.Sprintf("This is an example https server.\n\n%v\n", request)
	w.Write([]byte(response))
}