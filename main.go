package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
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
		q := r.URL.Query()
		idStr := q.Get("id")
		genre := strings.ToLower(q.Get("genre"))
		artist := strings.ToLower(q.Get("artist"))
		yearStr := q.Get("year")

		result := make([]Album, 0)
		for _, a := range albums {
			if idStr != "" {
				id, err := strconv.Atoi(idStr)
				if err != nil {
					http.Error(w, "Invalid id parameter", http.StatusBadRequest)
					return
				}
				if a.ID != id {
					continue
				}
			}
			if genre != "" && strings.ToLower(a.Genre) != genre {
				continue
			}
			if artist != "" && !strings.Contains(strings.ToLower(a.Artist), artist) {
				continue
			}
			if yearStr != "" {
				year, err := strconv.Atoi(yearStr)
				if err != nil {
					http.Error(w, "Invalid year parameter", http.StatusBadRequest)
					return
				}
				if a.Year != year {
					continue
				}
			}
			result = append(result, a)
		}
		writeJSON(w, http.StatusOK, result)

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

	case http.MethodPatch:
		var patch map[string]any
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		for i, a := range albums {
			if a.ID == id {
				if v, ok := patch["name"].(string); ok && v != "" {
					albums[i].Name = v
				}
				if v, ok := patch["artist"].(string); ok && v != "" {
					albums[i].Artist = v
				}
				if v, ok := patch["genre"].(string); ok && v != "" {
					albums[i].Genre = v
				}
				if v, ok := patch["status"].(string); ok && v != "" {
					albums[i].Status = v
				}
				if v, ok := patch["year"].(float64); ok && v >= 1900 {
					albums[i].Year = int(v)
				}
				if v, ok := patch["length"].(float64); ok && v > 0 {
					albums[i].Length = int(v)
				}
				if err := saveAlbums(); err != nil {
					http.Error(w, "Failed to persist data", http.StatusInternalServerError)
					return
				}
				writeJSON(w, http.StatusOK, albums[i])
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
