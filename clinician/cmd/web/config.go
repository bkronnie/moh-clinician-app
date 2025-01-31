package main

import (
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

type Config struct {
	Address      string `json:"Address"`
	ReadTimeout  int64  `json:"ReadTimeout"`
	WriteTimeout int64  `json:"WriteTimeout"`
	Static       string `json:"Static"`
	Ux           string `json:"Ux"`
	Px           string `json:"Px"`
	Dx           string `json:"Dx"`
}

func getConfig() (config Config) {
	file, err := os.Open("config.json")
	if err != nil {
		fmt.Println("Error opening config file:", err)
		return
	}
	defer file.Close()

	// Decode the JSON data into a Config struct
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		fmt.Println("Error decoding JSON:", err)
		return
	}

	return config
}
