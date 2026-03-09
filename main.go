package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type Album struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Artist string `json:"artist"`
	Length int    `json:"length"`
	Year   int    `json:"year"`
	Genre  string `json:"genre"`
	Status string `json:"status"`
}

type Message struct {
	Message string `json:"message"`
}

var albums []Album

func loadAlbums() {
	file, err := os.Open("data/albums.json")
	if err != nil {
		log.Fatal("Error reading file:", err)
	}

	err = json.NewDecoder(file).Decode(&albums)
	if err != nil {
		log.Fatal("Error parsing JSON:", err)
	}
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	response := Message{Message: "pong"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func albumsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(albums)
}

func main() {
	loadAlbums()

	http.HandleFunc("api/ping", pingHandler)
	http.HandleFunc("api/albums", albumsHandler)

	log.Println("Server running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
