package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

var initialAlbumsJSON = []byte(`[
    {"id":1,"name":"Bocanada","artist":"Gustavo Cerati","length":63,"year":1999,"genre":"Alternative Rock","status":"active"},
    {"id":2,"name":"Dynamo","artist":"Soda Stereo","length":68,"year":1992,"genre":"Alternative Rock","status":"active"},
    {"id":3,"name":"Debut","artist":"Björk","length":54,"year":1993,"genre":"Art Pop","status":"active"},
    {"id":4,"name":"OK Computer","artist":"Radiohead","length":53,"year":1997,"genre":"Alternative Rock","status":"active"},
    {"id":5,"name":"LP!","artist":"JPEGMAFIA","length":45,"year":2021,"genre":"Experimental Hip Hop","status":"active"},
    {"id":6,"name":"The Powers That B","artist":"Death Grips","length":81,"year":2015,"genre":"Experimental Hip Hop","status":"active"},
    {"id":7,"name":"De Todas las Flores","artist":"Natalia Lafourcade","length":66,"year":2022,"genre":"Latin Pop","status":"active"},
    {"id":8,"name":"YHLQMDLG","artist":"Bad Bunny","length":65,"year":2020,"genre":"Reggaeton","status":"active"},
    {"id":9,"name":"Wish You Were Here","artist":"Pink Floyd","length":44,"year":1975,"genre":"Progressive Rock","status":"active"},
    {"id":10,"name":"The New Abnormal","artist":"The Strokes","length":45,"year":2020,"genre":"Alternative Rock","status":"active"}
]`)

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

type ErrorResponse struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}

var (
	albums []Album
	once   sync.Once
	mu     sync.RWMutex
)

func initAlbums() {
	once.Do(func() {
		json.Unmarshal(initialAlbumsJSON, &albums)
	})
}

func getNextID() int {
	max := 0
	for _, a := range albums {
		if a.ID > max {
			max = a.ID
		}
	}
	return max + 1
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

func Handler(w http.ResponseWriter, r *http.Request) {
	initAlbums()

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	case path == "/api/ping":
		writeJSON(w, http.StatusOK, Message{Message: "pong"})

	case path == "/api/albums":
		handleAlbums(w, r)

	case strings.HasPrefix(path, "/api/albums/"):
		idStr := strings.TrimPrefix(path, "/api/albums/")
		if strings.Contains(idStr, "/") {
			writeError(w, http.StatusNotFound, "Route not found")
			return
		}
		handleAlbumByID(w, r, idStr)

	default:
		writeError(w, http.StatusNotFound, "Route not found")
	}
}

func handleAlbums(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		q := r.URL.Query()
		idStr := q.Get("id")
		genre := strings.ToLower(q.Get("genre"))
		artist := strings.ToLower(q.Get("artist"))
		yearStr := q.Get("year")

		hasFilters := idStr != "" || genre != "" || artist != "" || yearStr != ""

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

		mu.RLock()
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
		mu.RUnlock()

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

		mu.Lock()
		album.ID = getNextID()
		if album.Status == "" {
			album.Status = "active"
		}
		albums = append(albums, album)
		mu.Unlock()

		writeJSON(w, http.StatusCreated, album)

	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func handleAlbumByID(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Album ID must be an integer")
		return
	}

	switch r.Method {
	case http.MethodGet:
		mu.RLock()
		defer mu.RUnlock()
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

		mu.Lock()
		defer mu.Unlock()
		for i, a := range albums {
			if a.ID == id {
				updated.ID = id
				if updated.Status == "" {
					updated.Status = "active"
				}
				albums[i] = updated
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

		mu.Lock()
		defer mu.Unlock()
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
				writeJSON(w, http.StatusOK, albums[i])
				return
			}
		}
		writeError(w, http.StatusNotFound, "Album not found")

	case http.MethodDelete:
		mu.Lock()
		defer mu.Unlock()
		for i, a := range albums {
			if a.ID == id {
				albums = append(albums[:i], albums[i+1:]...)
				writeJSON(w, http.StatusOK, Message{Message: "Album deleted successfully"})
				return
			}
		}
		writeError(w, http.StatusNotFound, "Album not found")

	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}
