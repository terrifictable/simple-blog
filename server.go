package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"
)

var db *sql.DB

func NewTemplate(funcs template.FuncMap, files ...string) (*template.Template, error) {
	return template.New(path.Base(files[0])).Funcs(funcs).ParseFiles(files...)
}

func File(name string) string { return config.Prefix + name }
func LoadTemplates(config Config, funcs template.FuncMap) map[string]*template.Template {
	nav := File("website/admin/nav.templ")
	return map[string]*template.Template{
		"admin/index":     template.Must(NewTemplate(funcs, File("website/admin/index.html"), nav)),
		"admin/posts":     template.Must(NewTemplate(funcs, File("website/admin/posts.html"), nav)),
		"admin/users":     template.Must(NewTemplate(funcs, File("website/admin/users.html"), nav)),
		"admin/edit":      template.Must(NewTemplate(funcs, File("website/admin/edit.html"), nav)),
		"admin/user_edit": template.Must(NewTemplate(funcs, File("website/admin/user_edit.html"), nav)),

		"website/index": template.Must(NewTemplate(funcs, File("website/index.html"))),
		"website/post":  template.Must(NewTemplate(funcs, File("website/post.html"))),
		"website/login": template.Must(NewTemplate(funcs, File("website/login.html"))),
	}
}

var (
	posts     []Post
	config    Config
	templates map[string]*template.Template
)

var (
	xor_key = 0x42069420

	links = Links{
		"a home":  {Active: false, HREF: "/", Name: "home"},
		"b posts": {Active: false, HREF: "/admin/posts", Name: "posts"},
		"c users": {Active: false, HREF: "/admin/users", Name: "users"},

		"x logout": {Active: false, HREF: "/logout", Name: "logout", Before: "<br>"},
	}
	funcs = template.FuncMap{
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
)

func main() {
	var err error

	MustEmpty(config.ParseConfig())
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

	templates = LoadTemplates(config, funcs)
	posts = Must(UpdatePosts())

	router := http.NewServeMux()
	middlewares := CreateMiddlewares(LoggingMiddleware)

	port := "8080"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	router.Handle("/s/", http.StripPrefix("/s/", http.FileServer(http.Dir(config.Prefix+"static/"))))

	router.HandleFunc("/", IndexPageHandler)

	router.HandleFunc("POST /login", PostLoginPageHandler)
	router.HandleFunc("GET /login", GetLoginPageHandler)

	router.HandleFunc("GET /p/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "404 not found", http.StatusNotFound)
			return
		}

		posts, err = UpdatePosts()
		if err != nil {
			fmt.Printf("%v", err)
		}

		if config.DevMode {
			templates = LoadTemplates(config, funcs)
		}
		err = templates["website/post"].Execute(w, PostPageData{
			Post: GetPostByID(posts, id),
		})
		if err != nil {
			fmt.Printf("%v\n", err)
		}
	})

	router.HandleFunc("GET /logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "token", Value: "", Expires: time.Unix(0, 0)})
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		http.Redirect(w, r, "/", http.StatusPermanentRedirect)
	})

	router.HandleFunc("/admin/", AdminHandler)

	router.HandleFunc("GET /export/posts", func(w http.ResponseWriter, r *http.Request) {
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

	router.HandleFunc("/export/users", func(w http.ResponseWriter, r *http.Request) {
		is_authenticated, user := AuthenticateUser(w, r)
		if user.Equals(EmptyUser) || !is_authenticated || !user.CanManageUsers() {
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
	})

	server := http.Server{
		Addr:    ":" + port,
		Handler: middlewares(router),
	}
	fmt.Printf("Starting server at http://localhost:%s/\n", port)
	Must(server.ListenAndServe(), nil)
}
