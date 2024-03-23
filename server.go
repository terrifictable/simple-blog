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
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

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

func NewTemplate(funcs template.FuncMap, files ...string) (*template.Template, error) {
	return template.New(path.Base(files[0])).Funcs(funcs).ParseFiles(files...)
}

func main() {
	var config Config
	var err error

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

	funcs := template.FuncMap{
		"html": func(html string) template.HTML {
			return template.HTML(html)
		},
		"author": func(id int) string {
			by_id := GetUserById(id)
			if by_id.Equals(EmptyUser) {
				return ""
			}
			return by_id.Username
		},
	}

	templates := map[string]*template.Template{
		"admin/index":     template.Must(NewTemplate(funcs, config.Prefix+"website/admin/index.html", config.Prefix+"website/admin/nav.templ")),
		"admin/posts":     template.Must(NewTemplate(funcs, config.Prefix+"website/admin/posts.html", config.Prefix+"website/admin/nav.templ")),
		"admin/users":     template.Must(NewTemplate(funcs, config.Prefix+"website/admin/users.html", config.Prefix+"website/admin/nav.templ")),
		"admin/edit":      template.Must(NewTemplate(funcs, config.Prefix+"website/admin/edit.html", config.Prefix+"website/admin/nav.templ")),
		"admin/user_edit": template.Must(NewTemplate(funcs, config.Prefix+"website/admin/user_edit.html", config.Prefix+"website/admin/nav.templ")),

		"website/index": template.Must(NewTemplate(funcs, config.Prefix+"website/index.html")),
		"website/post":  template.Must(NewTemplate(funcs, config.Prefix+"website/post.html")),
		"website/login": template.Must(NewTemplate(funcs, config.Prefix+"website/login.html")),
	}

	var posts []Post
	posts = Must(UpdatePosts())

	http.Handle("/s/", http.StripPrefix("/s/", http.FileServer(http.Dir(config.Prefix+"static/"))))

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
				split := strings.Split(login_cookie.Value, " ")
				raw_username, err := base64.StdEncoding.DecodeString(split[0])
				if err == nil {
					raw_password, err := base64.StdEncoding.DecodeString(split[1])
					if err == nil {
						username := string(xor(raw_username, xor_key))
						password := string(xor(raw_password, xor_key))

						login, _, err = IsValidUser(username, password)
						if err != nil {
							login = false
						}
					}
				}
			} else {
				fmt.Printf("Cookie 'token' is invalid: %v\n", err)
			}
		}
		// templ := template.Must(template.ParseFiles(config.Prefix + "website/index.html"))
		err = templates["website/index"].Execute(w, IndexPageData{
			IsLoggedin: login,
			AllowLogin: config.AllowLogin,
			Posts:      posts,
		})
		if err != nil {
			fmt.Printf("%v\n", err)
		}
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
			// templ := template.Must(template.ParseFiles(config.Prefix + "website/login.html"))
			err := templates["website/login"].Execute(w, LoginPageData{
				AllowLogin: config.AllowLogin,
			})
			if err != nil {
				fmt.Printf("%v\n", err)
			}
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
		// templ := template.Must(template.ParseFiles(config.Prefix + "website/post.html"))
		err = templates["website/post"].Execute(w, PostPageData{
			Post: GetPostByID(posts, id),
		})
		if err != nil {
			fmt.Printf("%v\n", err)
		}
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

		trimmed := strings.TrimPrefix(r.URL.Path, "/admin/")
		if trimmed == "" {
			// templ := template.Must(NewTemplate(funcs, config.Prefix+"website/admin/index.html", templates[0]))
			err := templates["admin/index"].Execute(w, data)
			if err != nil {
				fmt.Printf("%v\n", err)
			}
			return
		} else if trimmed == "posts" {
			if l, ok := data.Links["b posts"]; ok {
				l.Active = true
				data.Links["b posts"] = l
			}
			// templ := template.Must(NewTemplate(funcs, config.Prefix+"website/admin/posts.html", templates[0]))
			err = templates["admin/posts"].Execute(w, data)
			if err != nil {
				fmt.Printf("%v\n", err)
			}
		} else if trimmed == "users" {
			users, err := GetUsers(user)
			if err != nil {
				http.Error(w, "401 unauthorized", http.StatusUnauthorized)
				return
			}

			if l, ok := data.Links["c users"]; ok {
				l.Active = true
				data.Links["c users"] = l
			}
			// templ := template.Must(NewTemplate(funcs, config.Prefix+"website/admin/users.html", templates[0]))
			err = templates["admin/users"].Execute(w, UsersPageData{
				IsAuthenticated: data.IsAuthenticated,
				Links:           data.Links,
				User:            user,
				Users:           users,
			})
			if err != nil {
				fmt.Printf("%v\n", err)
			}
		} else if strings.HasPrefix(trimmed, "users/") {
			trimmed = strings.TrimPrefix(trimmed, "users/")
			if strings.HasPrefix(trimmed, "delete/") {
				if r.Method != "POST" {
					http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
					return
				}

				id, err := strconv.Atoi(strings.TrimPrefix(trimmed, "delete/"))
				if err != nil {
					http.Error(w, "500 internal server error", http.StatusInternalServerError)
					return
				}

				err = DeleteUser(user, User{ID: id})
				if err != nil {
					http.Error(w, "500 internal server error", http.StatusInternalServerError)
					return
				}

				w.Header().Set("HX-Refresh", "true")
				w.WriteHeader(http.StatusOK)
			} else if strings.HasPrefix(trimmed, "edit/") {
				id, err := strconv.Atoi(strings.TrimPrefix(trimmed, "edit/"))
				if err != nil {
					http.Error(w, "500 internal error", http.StatusInternalServerError)
					return
				}

				to_edit := GetUserById(id)
				if to_edit.Equals(EmptyUser) {
					http.Error(w, fmt.Sprintf("404 user (%d) does not exist", id), http.StatusNotFound)
					return
				}

				if r.Method == "POST" {
					fmt.Fprintf(w, "TODO:")
					// w.Header().Set("HX-Redirect", "/admin/users")
					// w.WriteHeader(http.StatusOK)
					return
				}

				// templ := template.Must(NewTemplate(funcs, config.Prefix+"website/admin/user_edit.html", templates[0]))
				err = templates["admin/user_edit"].Execute(w, UserEditPageData{
					IsAuthenticated: data.IsAuthenticated,
					User:            user,
					Links:           data.Links,
					ToEdit:          to_edit,
					NewUser:         false,
				})
				if err != nil {
					fmt.Printf("%v\n", err)
				}
			} else if trimmed == "new" {
				fmt.Fprintf(w, "TODO:")
			} else {
				http.Error(w, "404 not found", http.StatusNotFound)
			}
		} else if strings.HasPrefix(trimmed, "posts/") {
			trimmed = strings.TrimPrefix(trimmed, "posts/")
			if strings.HasPrefix(trimmed, "delete/") {
				if r.Method != "POST" {
					http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
					return
				}

				id, err := strconv.Atoi(strings.TrimPrefix(trimmed, "delete/"))
				if err != nil {
					http.Error(w, "500 internal error", http.StatusInternalServerError)
					return
				}

				err = DeletePost(user, id)
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
				id, err := strconv.Atoi(strings.TrimPrefix(trimmed, "edit/"))
				if err != nil {
					http.Error(w, "500 internal error", http.StatusInternalServerError)
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
						if err := CreatePost(user, post); err != nil {
							http.Error(w, "500 internal error", http.StatusInternalServerError)
							fmt.Printf("CreatePost - %v\n", err)
							return
						}
					} else {
						if err := UpdatePost(user, post); err != nil {
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

				// templ := template.Must(NewTemplate(funcs, config.Prefix+"website/admin/edit.html", templates[0]))
				err = templates["admin/edit"].Execute(w, PostEditPageData{
					IsAuthenticated: data.IsAuthenticated,
					User:            user,
					Links:           data.Links,
					Post:            GetPostByID(posts, id),
					NewPost:         false,
				})
				if err != nil {
					fmt.Printf("%v\n", err)
				}
			} else if trimmed == "new" {
				// templ := template.Must(NewTemplate(funcs, config.Prefix+"website/admin/edit.html", templates[0]))
				err := templates["admin/edit"].Execute(w, PostEditPageData{
					IsAuthenticated: data.IsAuthenticated,
					User:            user,
					Links:           data.Links,
					Post:            Post{-1, user.ID, time.Now(), "", ""},
					NewPost:         true,
				})
				if err != nil {
					fmt.Printf("%v\n", err)
				}
			} else {
				http.Error(w, "404 not found", http.StatusNotFound)
			}
		} else {
			http.Error(w, "404 not found", http.StatusNotFound)
		}
	})

	http.HandleFunc("/export/", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()

		if strings.HasPrefix(r.URL.Path, "/export/posts") {
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
		} else if strings.HasPrefix(r.URL.Path, "/export/users") {
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

			if !is_authenticated || !user.CanManageUsers() {
				http.Error(w, "401 unauthorized", http.StatusUnauthorized)
				return
			}

			users, err := GetUsers(user)
			if err != nil {
				http.Error(w, "500 internal error", http.StatusInternalServerError)
				return
			}
			if r.Form.Has("nice") {
				data, err := json.MarshalIndent(users, "", "  ")
				if err != nil {
					http.Error(w, "500 internal error", http.StatusInternalServerError)
					return
				}
				fmt.Fprintf(w, "%s", data)
				return
			}
			data, err := json.Marshal(users)
			if err != nil {
				http.Error(w, "500 internal error", http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(w, "%s", data)
		} else {
			http.Error(w, "404 not found", http.StatusNotFound)
		}
	})

	fmt.Printf("Starting server at http://localhost:8000/\n")
	Must(http.ListenAndServe(":8000", nil), nil)
}
