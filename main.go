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

var albums []Album // Almacena los álbumes en memoria
var nextID int     // Chequea el siguiente ID disponible (para POST)

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

	for _, a := range albums {
		if a.ID >= nextID {
			nextID = a.ID + 1
		}
	}
}

func saveAlbums() error {
	data, err := json.MarshalIndent(albums, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile("data/albums.json", data, 0644)
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
	switch r.Method {
	case http.MethodGet:
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

	case http.MethodPost:
		var album Album
		if err := json.NewDecoder(r.Body).Decode(&album); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		album.ID = nextID
		nextID++
		if album.Status == "" {
			album.Status = "active"
		}
		albums = append(albums, album)

		if err := saveAlbums(); err != nil {
			http.Error(w, "Failed to persist data", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, album)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func albumByIDHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Album ID must be an integer", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		for _, a := range albums {
			if a.ID == id {
				writeJSON(w, http.StatusOK, a)
				return
			}
		}
		http.Error(w, "Album not found", http.StatusNotFound)

	case http.MethodPut:
		var updated Album
		if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		for i, a := range albums {
			if a.ID == id {
				updated.ID = id
				if updated.Status == "" {
					updated.Status = "active"
				}
				albums[i] = updated
				if err := saveAlbums(); err != nil {
					http.Error(w, "Failed to persist data", http.StatusInternalServerError)
					return
				}
				writeJSON(w, http.StatusOK, updated)
				return
			}
		}
		http.Error(w, "Album not found", http.StatusNotFound)

	case http.MethodDelete:
		for i, a := range albums {
			if a.ID == id {
				albums = append(albums[:i], albums[i+1:]...)
				if err := saveAlbums(); err != nil {
					http.Error(w, "Failed to persist data", http.StatusInternalServerError)
					return
				}
				writeJSON(w, http.StatusOK, Message{Message: "Album deleted successfully"})
				return
			}
		}
		http.Error(w, "Album not found", http.StatusNotFound)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func main() {
	loadAlbums()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/ping", pingHandler)
	mux.HandleFunc("/api/albums", albumsHandler)
	mux.HandleFunc("/api/albums/{id}", albumByIDHandler)

	log.Println("Albums API running on :24732")
	log.Fatal(http.ListenAndServe(":24732", mux))
}
