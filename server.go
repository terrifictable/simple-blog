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
func MustEmpty(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

var EncodedUsernamePassword string

func ConnectDB(cfg mysql.Config) (*sql.DB, error) {
	_db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		fmt.Printf("Failed to connect to database: \n\t> %v\n", err)
		fmt.Printf("Retrying in 10secs\n")
		time.Sleep(10 * time.Second)
		return ConnectDB(cfg)
	}
	return _db, nil
}

var defaults = struct {
	AllowLogin bool
	Prefix     string
	DB         struct {
		User     string
		Password string
		Address  string
		DBName   string
	}
}{
	AllowLogin: true,
	Prefix:     "./",
	DB: struct {
		User     string
		Password string
		Address  string
		DBName   string
	}{
		User:     string(rune(0x0)),
		Password: string(rune(0x0)),
		Address:  string(rune(0x0)),
		DBName:   string(rune(0x0)),
	},
}

func parseConfig(cfg *Config) error {
	parse := func(out *string, name string, def string) error {
		var exist bool
		*out, exist = os.LookupEnv(name)
		if !exist {
			fmt.Printf("Couldnt find '%s' in environment, using default '%s'\n", name, def)
			if def == string(rune(0x0)) {
				return fmt.Errorf("no default value for '%s'", name)
			}
			*out = def
		}
		return nil
	}

	var err error

	var allow_login string
	MustEmpty(parse(&allow_login, "ALLOW_LOGIN", strconv.FormatBool(defaults.AllowLogin)))
	cfg.AllowLogin, err = strconv.ParseBool(allow_login)
	if err != nil {
		return err
	}

	MustEmpty(parse(&cfg.Prefix, "PREFIX", defaults.Prefix))

	MustEmpty(parse(&cfg.DB.User, "DB_USER", defaults.DB.User))
	MustEmpty(parse(&cfg.DB.Password, "DB_PASSWORD", defaults.DB.Password))
	MustEmpty(parse(&cfg.DB.Address, "DB_ADDR", defaults.DB.Address))
	MustEmpty(parse(&cfg.DB.DBName, "DB_NAME", defaults.DB.DBName))

	return nil
}

func main() {
	var config Config
	var err error

	// xor_key := rand.Intn(0x7fffffff)
	xor_key := 0x42069420

	MustEmpty(parseConfig(&config))

	cfg := mysql.Config{
		User:      config.DB.User,
		Passwd:    config.DB.Password,
		Net:       "tcp",
		Addr:      config.DB.Address,
		DBName:    config.DB.DBName,
		ParseTime: true,
	}

	db = Must(ConnectDB(cfg))
	Must(db.Ping(), nil)
	fmt.Println("DB Connected")

	var posts []Post
	posts = Must(UpdatePosts())

	http.Handle("/s/", http.StripPrefix("/s/", http.FileServer(http.Dir(config.Prefix+"static/"))))

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
		templ := template.Must(template.ParseFiles(config.Prefix + "website/index.html"))
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

			valid, _, err := IsValidUser(username, password)
			if !valid || err != nil {
				fmt.Fprintf(w, "Invalid login, %t - %v", valid, err)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:  "token",
				Value: base64.StdEncoding.EncodeToString(xor([]byte(username), xor_key)) + " " + base64.StdEncoding.EncodeToString(xor([]byte(password), xor_key)),
			})
			w.Header().Set("HX-Redirect", "/admin/posts")
		} else if r.Method == "GET" {
			templ := template.Must(template.ParseFiles(config.Prefix + "website/login.html"))
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
		templ := template.Must(template.ParseFiles(config.Prefix + "website/post.html"))
		templ.Execute(w, PostPageData{
			Post: GetPostByID(posts, id),
		})
	})

	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "token", Value: "", Expires: time.Unix(0, 0)})
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		http.Redirect(w, r, "/", http.StatusPermanentRedirect)
	})

	http.HandleFunc("/admin/", func(w http.ResponseWriter, r *http.Request) {
		posts, err = UpdatePosts()
		if err != nil {
			fmt.Printf("%v", err)
		}

		var user User
		var is_authenticated bool = false
		cookie, err := r.Cookie("token")
		if err == nil {
			split := strings.Split(cookie.Value, " ")
			raw_username, err := base64.StdEncoding.DecodeString(split[0])
			if err != nil {
				is_authenticated = false
				http.Error(w, "401 unauthorized", http.StatusUnauthorized)
				return
			}
			raw_password, err := base64.StdEncoding.DecodeString(split[1])
			if err != nil {
				is_authenticated = false
				http.Error(w, "401 unauthorized", http.StatusUnauthorized)
				return
			}
			username := string(xor(raw_username, xor_key))
			password := string(xor(raw_password, xor_key))

			is_authenticated, user, err = IsValidUser(username, password)
			if err != nil {
				is_authenticated = false
			}
		}
		if !is_authenticated {
			http.Error(w, "401 unauthorized", http.StatusUnauthorized)
			return
		}

		data := AdminPageData{
			IsAuthenticated: is_authenticated,
			User:            user,

			Posts: posts,
			Links: map[string]Link{
				"a home":  {Active: false, HREF: "/", Name: "home"},
				"b posts": {Active: false, HREF: "/admin/posts", Name: "posts"},
				"c users": {Active: false, HREF: "/admin/users", Name: "users"},

				"x logout": {Active: false, HREF: "/logout", Name: "logout", Before: "<br>"},
			},
		}

		funcs := template.FuncMap{
			"html": func(html string) template.HTML {
				return template.HTML(html)
			},
		}

		trimmed := strings.TrimPrefix(r.URL.Path, "/admin/")
		if trimmed == "" {
			templ := template.Must(template.ParseFiles(config.Prefix + "website/admin/index.html"))
			templ.Funcs(funcs)
			templ.Execute(w, data)
			return
		} else if trimmed == "posts" {
			templ := template.Must(template.ParseFiles(config.Prefix + "website/admin/posts.html"))
			if l, ok := data.Links["b posts"]; ok {
				l.Active = true
				data.Links["b posts"] = l
			}
			templ.Funcs(funcs)
			templ.Execute(w, data)
			return
		} else if strings.HasPrefix(trimmed, "users") {
			// TODO:
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

			templ := template.Must(template.ParseFiles(config.Prefix + "website/admin/edit.html"))
			templ.Funcs(funcs)
			templ.Execute(w, EditPageData{
				IsAuthenticated: data.IsAuthenticated,
				Links:           data.Links,
				Post:            GetPostByID(posts, id),
				NewPost:         false,
			})
		} else if trimmed == "new" {
			templ := template.Must(template.ParseFiles(config.Prefix + "website/admin/edit.html"))
			templ.Funcs(funcs)
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
