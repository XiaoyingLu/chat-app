package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/joho/godotenv"
	"github.com/stretchr/gomniauth"
	"github.com/stretchr/gomniauth/providers/facebook"
	"github.com/stretchr/gomniauth/providers/github"
	"github.com/stretchr/gomniauth/providers/google"
	"github.com/stretchr/objx"
	"github.com/stretchr/signature"

	"trace"
)

// templ represents a single template
type templateHandler struct {
	once     sync.Once
	filename string
	templ    *template.Template
}

// ServeHTTP handles the HTTP request
func (t *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.once.Do(func() {
		t.templ = template.Must(template.ParseFiles(filepath.Join("templates", t.filename)))
	})
	data := map[string]interface{}{
		"Host": r.Host,
	}
	if authCookie, err := r.Cookie("auth"); err == nil {
		data["UserData"] = objx.MustFromBase64(authCookie.Value)
	}
	t.templ.Execute(w, data)
}

func main() {
	var addr = flag.String("addr", ":8080", "The address of the application.")
	flag.Parse() // parse the flags

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// setup gomniauth
	gomniauth.SetSecurityKey(signature.RandomKey(64))
	gomniauth.WithProviders(
		facebook.New(os.Getenv("FACEBOOK_CLIENT_ID"), os.Getenv("FACEBOOK_CLIENT_SECRET"),
       "http://localhost:8080/auth/callback/facebook"),
     github.New(os.Getenv("GITHUB_CLIENT_ID"), os.Getenv("GITHUB_CLIENT_SECRET"),
       "http://localhost:8080/auth/callback/github"),
     google.New(os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET"),
       "http://localhost:8080/auth/callback/google"),
	)

	r := newRoom()
	r.tracer = trace.New(os.Stdout)
	http.Handle("/assets/", http.StripPrefix("/assets", http.FileServer(http.Dir("assets"))))
	http.Handle("/chat", MustAuth(&templateHandler{filename: "chat.html"}))
	http.Handle("/login", &templateHandler{filename: "login.html"})
	http.HandleFunc("/auth/", loginHandler)
	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:   "auth",
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})
		w.Header().Set("Location", "/chat")
		w.WriteHeader(http.StatusTemporaryRedirect)
	})

	// Add handler for room path
	http.Handle("/room", r)
	// get the room going
	go r.run()

	// start the web server
	log.Println("Starting web server on", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
