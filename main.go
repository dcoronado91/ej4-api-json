package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
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
	defer file.Close()

	err = json.NewDecoder(file).Decode(&albums)
	if err != nil {
		log.Fatal("Error parsing JSON:", err)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, Message{Message: "pong"})
}

func albumsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		writeJSON(w, http.StatusOK, albums)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid id parameter", http.StatusBadRequest)
		return
	}

	for _, a := range albums {
		if a.ID == id {
			writeJSON(w, http.StatusOK, a)
			return
		}
	}

	http.Error(w, "Album not found", http.StatusNotFound)
}

func main() {
	loadAlbums()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/ping", pingHandler)
	mux.HandleFunc("/api/albums", albumsHandler)

	log.Println("Albums API running on :24732")
	log.Fatal(http.ListenAndServe(":24732", mux))
}
