package api

import (
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

type ImagePngHash struct {
	gorm.Model
	Hash string `gorm:"primarykey"`
	Data string `gorm:"primarykey"`
}

var APIserver string

func init() {
	godotenv.Load()
	APIserver = os.Getenv("API_SERVER")
}

func GetApi(URI string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("GET", "http://"+APIserver+URI, body)
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
