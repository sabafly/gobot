package api

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

var APIserver string

func init() {
	godotenv.Load()
	APIserver = os.Getenv("API_SERVER")
}

func GetApi(URI string) (*http.Response, error) {
	req, err := http.NewRequest("GET", "http://"+APIserver+URI, http.NoBody)
	if err != nil {
		log.Printf("error on api: %v", err)
		return &http.Response{}, err
	}
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("error on api: %v", err)
		return resp, err
	}
	return resp, nil
}
