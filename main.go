package main

import (
	"crypto/subtle"
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/eugene-bert/immich-auto-albums/immich"
	"github.com/eugene-bert/immich-auto-albums/rules"
	"github.com/eugene-bert/immich-auto-albums/sync"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

var tmpl *template.Template

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	store, err := rules.Open(envOr("DB_PATH", "rules.db"))
	if err != nil {
		log.Fatal(err)
	}

	client := &immich.Client{
		URL:    strings.TrimRight(os.Getenv("IMMICH_URL"), "/"),
		APIKey: os.Getenv("IMMICH_API_KEY"),
	}
	if client.URL == "" || client.APIKey == "" {
		log.Fatal("IMMICH_URL and IMMICH_API_KEY are required")
	}

	tmpl = template.Must(template.ParseFS(templateFS, "templates/*.html"))

	go runScheduler(store, client)

	mux := http.NewServeMux()
	static, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		list, err := store.List()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		connected := client.Ping() == nil
		tmpl.ExecuteTemplate(w, "layout", map[string]any{"Rules": list, "Connected": connected})
	})

	mux.HandleFunc("GET /rules/new", func(w http.ResponseWriter, r *http.Request) {
		explore, err := client.Explore()
		if err != nil {
			log.Printf("explore: %v", err)
		}
		explore.AlbumNames, err = client.ListAlbumNames()
		if err != nil {
			log.Printf("albums: %v", err)
		}
		tmpl.ExecuteTemplate(w, "rule-form", map[string]any{"Rule": rules.Rule{}, "Explore": explore})
	})

	mux.HandleFunc("POST /rules", func(w http.ResponseWriter, r *http.Request) {
		rule, err := parseForm(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := rule.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := store.Create(&rule); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.ExecuteTemplate(w, "rule-row", rule)
	})

	mux.HandleFunc("GET /rules/{id}/edit", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		rule, err := store.Get(id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		explore, err := client.Explore()
		if err != nil {
			log.Printf("explore: %v", err)
		}
		explore.AlbumNames, err = client.ListAlbumNames()
		if err != nil {
			log.Printf("albums: %v", err)
		}
		tmpl.ExecuteTemplate(w, "rule-form", map[string]any{"Rule": rule, "Explore": explore})
	})

	mux.HandleFunc("PUT /rules/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		rule, err := parseForm(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		rule.ID = id
		if err := rule.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := store.Update(&rule); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rule, err = store.Get(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.ExecuteTemplate(w, "rule-row", rule)
	})

	mux.HandleFunc("DELETE /rules/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err := store.Delete(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("POST /rules/{id}/sync", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		rule, err := store.Get(id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err := sync.Run(client, store, rule); err != nil {
			log.Printf("sync error [%s]: %v", rule.Name, err)
		}
		rule, err = store.Get(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.ExecuteTemplate(w, "rule-row", rule)
	})

	port := envOr("PORT", "8095")
	handler := maybeBasicAuth(mux)
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("immich-auto-albums listening on :%s", port)
	log.Fatal(srv.ListenAndServe())
}

func maybeBasicAuth(next http.Handler) http.Handler {
	user := os.Getenv("UI_USER")
	pass := os.Getenv("UI_PASSWORD")
	if user == "" && pass == "" {
		return next
	}
	if user == "" || pass == "" {
		log.Fatal("UI_USER and UI_PASSWORD must be set together")
	}
	userB := []byte(user)
	passB := []byte(pass)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(u), userB) != 1 || subtle.ConstantTimeCompare([]byte(p), passB) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="immich-auto-albums"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func parseForm(r *http.Request) (rules.Rule, error) {
	if err := r.ParseForm(); err != nil {
		return rules.Rule{}, err
	}
	interval, _ := strconv.Atoi(r.FormValue("interval"))
	if interval < 60 {
		interval = 3600
	}
	return rules.Rule{
		Name:             r.FormValue("name"),
		AlbumName:        r.FormValue("album_name"),
		CameraMake:       r.FormValue("camera_make"),
		CameraModel:      r.FormValue("camera_model"),
		LensModel:        r.FormValue("lens_model"),
		MediaType:        r.FormValue("media_type"),
		City:             r.FormValue("city"),
		State:            r.FormValue("state"),
		Country:          r.FormValue("country"),
		TakenAfter:       r.FormValue("taken_after"),
		TakenBefore:      r.FormValue("taken_before"),
		OriginalFileName: r.FormValue("original_file_name"),
		Description:      r.FormValue("description"),
		Interval:         interval,
		Enabled:          r.FormValue("enabled") == "1",
	}, nil
}

func runScheduler(store *rules.Store, client *immich.Client) {
	for {
		list, err := store.List()
		if err != nil {
			log.Printf("scheduler: %v", err)
			time.Sleep(60 * time.Second)
			continue
		}

		for _, rule := range list {
			if !rule.Enabled {
				continue
			}
			if time.Since(rule.LastSync) < time.Duration(rule.Interval)*time.Second {
				continue
			}
			if err := sync.Run(client, store, rule); err != nil {
				log.Printf("scheduler sync error [%s]: %v", rule.Name, err)
			}
		}

		time.Sleep(30 * time.Second)
	}
}
