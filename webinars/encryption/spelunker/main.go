package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"

	// "fmt"
	// "io"
	"net/http"
)

func main() {
	certFile := os.Getenv("CERT_FILE")
	if certFile == "" {
		certFile = "localhost.crt"
	}
	keyFile := os.Getenv("KEY_FILE")
	if keyFile == "" {
		keyFile = "localhost.key"
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
	response := ""
	errRateStr := request.Header.Get("x-error-rate")
	if errRateStr != "" {
		errRate, err := strconv.Atoi(errRateStr)
		if err != nil {
			response = fmt.Sprintf("Error parsing error rate %s: %s", errRateStr, err)
		} else if errRate < 0 || errRate > 100 {
			response = fmt.Sprintf("Invalid error rate: %d", errRate)
		} else {
			randVal := rand.Intn(100)
			if randVal <= errRate {
				w.WriteHeader(http.StatusInternalServerError)
				response = "Server error!"
			}
		}
	}

	if response == "" {
		response = fmt.Sprintf("This is an example http server.\n\n%v\n", request)
	}
	w.Write([]byte(response))
}

type HttpsHandler struct {

}

func (h *HttpsHandler) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	response := fmt.Sprintf("This is an example https server.\n\n%v\n", request)
	w.Write([]byte(response))
}