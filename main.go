package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/eugene-bert/immich-auto-albums/immich"
	"github.com/eugene-bert/immich-auto-albums/rules"
	"github.com/eugene-bert/immich-auto-albums/sync"
)

//go:embed templates/*.html
var templateFS embed.FS

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
		URL:    os.Getenv("IMMICH_URL"),
		APIKey: os.Getenv("IMMICH_API_KEY"),
	}
	if client.URL == "" || client.APIKey == "" {
		log.Fatal("IMMICH_URL and IMMICH_API_KEY are required")
	}

	tmpl = template.Must(template.ParseFS(templateFS, "templates/*.html"))

	go runScheduler(store, client)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		list, _ := store.List()
		tmpl.ExecuteTemplate(w, "layout", map[string]any{"Rules": list})
	})

	mux.HandleFunc("GET /rules/new", func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, "rule-form", rules.Rule{})
	})

	mux.HandleFunc("POST /rules", func(w http.ResponseWriter, r *http.Request) {
		rule := parseForm(r)
		if err := store.Create(&rule); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		tmpl.ExecuteTemplate(w, "rule-row", rule)
	})

	mux.HandleFunc("GET /rules/{id}/edit", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
		rule, err := store.Get(id)
		if err != nil {
			http.Error(w, "not found", 404)
			return
		}
		tmpl.ExecuteTemplate(w, "rule-form", rule)
	})

	mux.HandleFunc("PUT /rules/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
		rule := parseForm(r)
		rule.ID = id
		if err := store.Update(&rule); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		rule, _ = store.Get(id)
		tmpl.ExecuteTemplate(w, "rule-row", rule)
	})

	mux.HandleFunc("DELETE /rules/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
		store.Delete(id)
		w.WriteHeader(200)
	})

	mux.HandleFunc("POST /rules/{id}/sync", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
		rule, err := store.Get(id)
		if err != nil {
			http.Error(w, "not found", 404)
			return
		}
		if err := sync.Run(client, store, rule); err != nil {
			log.Printf("sync error [%s]: %v", rule.Name, err)
		}
		rule, _ = store.Get(id)
		tmpl.ExecuteTemplate(w, "rule-row", rule)
	})

	port := envOr("PORT", "8095")
	log.Printf("immich-auto-albums listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func parseForm(r *http.Request) rules.Rule {
	r.ParseForm()
	interval, _ := strconv.Atoi(r.FormValue("interval"))
	if interval < 60 {
		interval = 3600
	}
	return rules.Rule{
		Name:        r.FormValue("name"),
		AlbumName:   r.FormValue("album_name"),
		CameraMake:  r.FormValue("camera_make"),
		CameraModel: r.FormValue("camera_model"),
		LensModel:   r.FormValue("lens_model"),
		MediaType:   r.FormValue("media_type"),
		City:        r.FormValue("city"),
		State:       r.FormValue("state"),
		Country:     r.FormValue("country"),
		TakenAfter:  r.FormValue("taken_after"),
		TakenBefore: r.FormValue("taken_before"),
		Interval:    interval,
		Enabled:     r.FormValue("enabled") == "1",
	}
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
