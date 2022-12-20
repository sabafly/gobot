// Copyright (C) 2022  ikafly144

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
package api

import (
	"io"
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
