package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"gopkg.in/yaml.v3"
)

type Post struct {
	ID      int       `json:"id"`
	Date    time.Time `json:"date"`
	Title   string    `json:"title"`
	Content string    `json:"content"`
}

var db *sql.DB

func Must[V any](v V, e error) V {
	if e != nil {
		log.Fatal(e)
	}
	return v
}

func xor(str string, key int) []byte {
	k1 := key & 0xff
	k2 := (key >> 8) & 0xff
	k3 := (key >> 16) & 0xff
	k4 := (key >> 24) & 0xff

	res := make([]byte, 0)
	for _, c := range str {
		res = append(res, byte((((int(c)^k1)^k2)^k3)^k4))
	}
	return res
}

var EncodedUsernamePassword string

func main() {
	var config Config
	var err error

	config_file := "config.yml"
	if len(os.Args) > 1 {
		config_file = os.Args[1]
	}

	Must(yaml.Unmarshal(Must(os.ReadFile(config_file)), &config), nil)

	EncodedUsernamePassword = base64.StdEncoding.EncodeToString(xor(config.Username+config.Password, config.XorKey))

	cfg := mysql.Config{
		User:      config.DB.User,     // User:      "user",
		Passwd:    config.DB.Password, // Passwd:    "gmLv93bhAtn8U5ss",
		Net:       "tcp",
		Addr:      config.DB.Address, // Addr:      "127.0.0.1:3306",
		DBName:    config.DB.DBName,  // DBName:    "data",
		ParseTime: true,
	}

	db = Must(sql.Open("mysql", cfg.FormatDSN()))
	Must(db.Ping(), nil)
	fmt.Println("DB Connected")

	var posts []Post
	posts = Must(UpdatePosts())

	defer func() {
		out := Must(yaml.Marshal(config))
		os.WriteFile(config_file, out, 0644)
	}()

	http.Handle("/s/", http.StripPrefix("/s/", http.FileServer(http.Dir("./static/"))))

	http.HandleFunc("/c", func(w http.ResponseWriter, r *http.Request) {
		out, err := yaml.Marshal(config)
		if err != nil {
			fmt.Fprintf(w, "Error marshalling config, %v", err)
		}
		fmt.Fprintf(w, "%s", out)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.Error(w, "404 not found.", http.StatusNotFound)
			return
		}

		posts, err = UpdatePosts()
		if err != nil {
			fmt.Printf("%v", err)
		}

		var login bool = false
		login_cookie, err := r.Cookie("token")
		if err != nil {
			if err != http.ErrNoCookie {
				fmt.Printf("Error while getting cookie 'token': %v\n", err)
			}
		} else {
			err = login_cookie.Valid()
			if err == nil {
				login = login_cookie.Value == EncodedUsernamePassword
			} else {
				fmt.Printf("Cookie 'token' is invalid: %v\n", err)
			}
		}
		templ := template.Must(template.ParseFiles("website/index.html"))
		templ.Execute(w, IndexPageData{
			IsLoggedin: login,
			AllowLogin: config.AllowLogin,
			Posts:      posts,
		})
	})

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			r.ParseForm()

			username := r.PostForm.Get("username")
			password := r.PostForm.Get("password")

			if username != config.Username || password != config.Password {
				fmt.Fprintf(w, "Invalid login")
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:  "token",
				Value: base64.StdEncoding.EncodeToString(xor(config.Username+config.Password, config.XorKey)),
			})
			w.Header().Set("HX-Redirect", "/")
			// w.WriteHeader(http.StatusOK)
		} else if r.Method == "GET" {
			templ := template.Must(template.ParseFiles("website/login.html"))
			templ.Execute(w, LoginPageData{
				AllowLogin: config.AllowLogin,
			})
		}
	})

	http.HandleFunc("/p/", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/p/"))
		if err != nil {
			http.Error(w, "500 internal server error", http.StatusInternalServerError)
			return
		}

		posts, err = UpdatePosts()
		if err != nil {
			fmt.Printf("%v", err)
		}
		templ := template.Must(template.ParseFiles("website/post.html"))
		templ.Execute(w, PostPageData{
			Post: GetPostByID(posts, id),
		})
	})

	http.HandleFunc("/admin/", func(w http.ResponseWriter, r *http.Request) {
		posts, err = UpdatePosts()
		if err != nil {
			fmt.Printf("%v", err)
		}

		var is_authenticated bool = false
		cookie, err := r.Cookie("token")
		if err == nil {
			is_authenticated = cookie.Value == EncodedUsernamePassword
		} else {
			if err != http.ErrNoCookie {
				fmt.Printf("Error while getting cookie 'token': %v\n", err)
			}
		}

		if !is_authenticated {
			http.Error(w, "401 unauthorized", http.StatusUnauthorized)
			return
		}

		data := AdminPageData{
			IsAuthenticated: is_authenticated,
			Posts:           posts,
			Links: map[string]Link{
				"home":  {Active: false, HREF: "/", Name: "home"},
				"posts": {Active: false, HREF: "/admin/posts", Name: "posts"},
			},
		}
		trimmed := strings.TrimPrefix(r.URL.Path, "/admin/")
		if trimmed == "" {
			templ := template.Must(template.ParseFiles("website/admin/index.html"))
			templ.Execute(w, data)
			return
		} else if trimmed == "posts" {
			templ := template.Must(template.ParseFiles("website/admin/posts.html"))
			if l, ok := data.Links["posts"]; ok {
				l.Active = true
				data.Links["posts"] = l
			}
			templ.Execute(w, data)
			return
		} else if strings.HasPrefix(trimmed, "delete/") {
			if r.Method != "POST" || !data.IsAuthenticated {
				http.Error(w, "404 not found", http.StatusNotFound)
				return
			}

			id, err := strconv.Atoi(strings.TrimPrefix(trimmed, "delete/"))
			if err != nil {
				http.Error(w, "500 internal error", http.StatusInternalServerError)
				return
			}

			err = DeletePost(id)
			if err != nil {
				http.Error(w, "failed to delete post", http.StatusInternalServerError)
				return
			}
			posts, err = UpdatePosts()
			if err != nil {
				http.Error(w, "failed to update posts", http.StatusInternalServerError)
				return
			}

			w.Header().Set("HX-Refresh", "true")
			w.WriteHeader(http.StatusOK)
		} else if strings.HasPrefix(trimmed, "edit/") {
			if !data.IsAuthenticated {
				http.Error(w, "404 not found", http.StatusNotFound)
				return
			}

			id, err := strconv.Atoi(strings.TrimPrefix(trimmed, "edit/"))
			if err != nil {
				http.Error(w, "500 internal error - Atoi", http.StatusInternalServerError)
				fmt.Printf("Atoi - %v\n", err)
				return
			}

			posts, err = UpdatePosts()
			if err != nil {
				http.Error(w, "500 internal error", http.StatusInternalServerError)
				fmt.Printf("Update Posts - %v\n", err)
				return
			}

			if r.Method == "POST" {
				r.ParseForm()

				title := strings.Trim(r.PostForm.Get("title"), " \t\n")
				content := strings.Trim(r.PostForm.Get("content"), " \t\n")
				if content == "" || title == "" {
					http.Error(w, "title and content cannot be empty", http.StatusBadRequest)
					return
				}
				post := Post{
					ID:      id,
					Date:    time.Now(),
					Title:   title,
					Content: content,
				}
				if r.PostForm.Get("new") == "true" || id == -1 {
					if err := CreatePost(post); err != nil {
						http.Error(w, "500 internal error", http.StatusInternalServerError)
						fmt.Printf("CreatePost - %v\n", err)
						return
					}
				} else {
					if err := UpdatePost(post); err != nil {
						http.Error(w, "500 internal error", http.StatusInternalServerError)
						fmt.Printf("UpdatePost - %v\n", err)
						return
					}
				}

				posts, err = UpdatePosts()
				if err != nil {
					http.Error(w, "500 internal error", http.StatusInternalServerError)
					fmt.Printf("Update Posts - %v\n", err)
					return
				}

				w.Header().Set("HX-Redirect", "/admin/posts")
				w.WriteHeader(http.StatusOK)
				return
			}

			templ := template.Must(template.ParseFiles("website/admin/edit.html"))
			templ.Execute(w, EditPageData{
				IsAuthenticated: data.IsAuthenticated,
				Links:           data.Links,
				Post:            GetPostByID(posts, id),
				NewPost:         false,
			})
		} else if trimmed == "new" {
			templ := template.Must(template.ParseFiles("website/admin/edit.html"))
			templ.Execute(w, EditPageData{
				IsAuthenticated: data.IsAuthenticated,
				Links:           data.Links,
				Post:            Post{-1, time.Now(), "", ""},
				NewPost:         true,
			})
		} else {
			http.Error(w, "404 not found", http.StatusNotFound)
		}
	})

	http.HandleFunc("/export", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()

		posts, err = UpdatePosts()
		if err != nil {
			http.Error(w, "500 internal error", http.StatusInternalServerError)
			return
		}
		if r.Form.Has("nice") {
			data, err := json.MarshalIndent(posts, "", "  ")
			if err != nil {
				http.Error(w, "500 internal error", http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(w, "%s", data)
			return
		}
		data, err := json.Marshal(posts)
		if err != nil {
			http.Error(w, "500 internal error", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "%s", data)
	})

	fmt.Printf("Starting server at http://localhost:8000/\n")
	Must(http.ListenAndServe(":8000", nil), nil)
}
