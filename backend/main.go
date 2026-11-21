package main

import (
    "encoding/json"
    "log"
    "net/http"
    "os"
)

type Response struct {
    Message string `json:"message"`
    Version string `json:"version"`
}

func main() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        response := Response{
            Message: "Hello from Go running on Kubernetes!",
            Version: "1.0.0",
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    })

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    log.Println("Server running on port", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}