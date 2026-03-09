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

// Manejo de errores JSON
type ErrorResponse struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}

var albums []Album // Almacena los álbumes en memoria

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

// Calcula el siguiente ID disponible basado en el estado actual del slice
func getNextID() int {
	max := 0
	for _, a := range albums {
		if a.ID > max {
			max = a.ID
		}
	}
	return max + 1
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

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message, Code: status})
}

func validateAlbum(album Album) string {
	if album.Name == "" {
		return "Field 'name' is required"
	}
	if album.Artist == "" {
		return "Field 'artist' is required"
	}
	if album.Genre == "" {
		return "Field 'genre' is required"
	}
	if album.Year < 1900 || album.Year > 2100 {
		return "Field 'year' must be between 1900 and 2100"
	}
	if album.Length <= 0 {
		return "Field 'length' must be a positive number"
	}
	return ""
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

		hasFilters := idStr != "" || genre != "" || artist != "" || yearStr != ""

		// Validar tipos antes de filtrar
		var filterID, filterYear int
		if idStr != "" {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				writeError(w, http.StatusBadRequest, "Query parameter 'id' must be an integer")
				return
			}
			filterID = id
		}
		if yearStr != "" {
			year, err := strconv.Atoi(yearStr)
			if err != nil {
				writeError(w, http.StatusBadRequest, "Query parameter 'year' must be an integer")
				return
			}
			filterYear = year
		}

		result := make([]Album, 0)
		for _, a := range albums {
			if idStr != "" && a.ID != filterID {
				continue
			}
			if genre != "" && strings.ToLower(a.Genre) != genre {
				continue
			}
			if artist != "" && !strings.Contains(strings.ToLower(a.Artist), artist) {
				continue
			}
			if yearStr != "" && a.Year != filterYear {
				continue
			}
			result = append(result, a)
		}

		if hasFilters && len(result) == 0 {
			writeError(w, http.StatusNotFound, "No albums found matching the given filters")
			return
		}

		writeJSON(w, http.StatusOK, result)

	case http.MethodPost:
		var album Album
		if err := json.NewDecoder(r.Body).Decode(&album); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		if msg := validateAlbum(album); msg != "" {
			writeError(w, http.StatusBadRequest, msg)
			return
		}

		album.ID = getNextID()
		if album.Status == "" {
			album.Status = "active"
		}
		albums = append(albums, album)

		if err := saveAlbums(); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to persist data")
			return
		}

		writeJSON(w, http.StatusCreated, album)

	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func albumByIDHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Album ID must be an integer")
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
		writeError(w, http.StatusNotFound, "Album not found")

	case http.MethodPut:
		var updated Album
		if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		if msg := validateAlbum(updated); msg != "" {
			writeError(w, http.StatusBadRequest, msg)
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
					writeError(w, http.StatusInternalServerError, "Failed to persist data")
					return
				}
				writeJSON(w, http.StatusOK, updated)
				return
			}
		}
		writeError(w, http.StatusNotFound, "Album not found")

	case http.MethodPatch:
		var patch map[string]any
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON body")
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
					writeError(w, http.StatusInternalServerError, "Failed to persist data")
					return
				}
				writeJSON(w, http.StatusOK, albums[i])
				return
			}
		}
		writeError(w, http.StatusNotFound, "Album not found")

	case http.MethodDelete:
		for i, a := range albums {
			if a.ID == id {
				albums = append(albums[:i], albums[i+1:]...)
				if err := saveAlbums(); err != nil {
					writeError(w, http.StatusInternalServerError, "Failed to persist data")
					return
				}
				writeJSON(w, http.StatusOK, Message{Message: "Album deleted successfully"})
				return
			}
		}
		writeError(w, http.StatusNotFound, "Album not found")

	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
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
