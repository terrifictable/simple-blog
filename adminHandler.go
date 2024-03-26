package main

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

func AdminHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	is_authenticated, user := AuthenticateUser(w, r)
	if !is_authenticated {
		http.Error(w, "401 unauthorized", http.StatusUnauthorized)
		return
	}

	posts, err = UpdatePosts()
	if err != nil {
		fmt.Printf("%v", err)
	}

	data := AdminPageData{
		IsAuthenticated: is_authenticated,
		User:            user,

		Posts: posts,
		Links: links,
	}

	trimmed := strings.TrimPrefix(r.URL.Path, "/admin/")
	if trimmed == "" {
		if config.DevMode {
			templates = LoadTemplates(config, funcs)
		}
		err := templates["admin/index"].Execute(w, data)
		if err != nil {
			fmt.Printf("%v\n", err)
		}
		return
	} else if trimmed == "posts" {
		data.Links.Set("b posts")
		if config.DevMode {
			templates = LoadTemplates(config, funcs)
		}
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

		data.Links.Set("c users")
		if config.DevMode {
			templates = LoadTemplates(config, funcs)
		}
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

			if config.DevMode {
				templates = LoadTemplates(config, funcs)
			}
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
			posts = slices.DeleteFunc(posts, func(post Post) bool {
				return post.ID == id
			})

			w.Header().Set("HX-Refresh", "true")
			w.WriteHeader(http.StatusOK)
		} else if strings.HasPrefix(trimmed, "edit/") {
			id, err := strconv.Atoi(strings.TrimPrefix(trimmed, "edit/"))
			if err != nil {
				http.Error(w, "500 internal error", http.StatusInternalServerError)
				fmt.Printf("Atoi - %v\n", err)
				return
			}

			// posts, err = UpdatePosts()
			// if err != nil {
			// 	http.Error(w, "500 internal error", http.StatusInternalServerError)
			// 	fmt.Printf("Update Posts - %v\n", err)
			// 	return
			// }

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

			if config.DevMode {
				templates = LoadTemplates(config, funcs)
			}
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
			if config.DevMode {
				templates = LoadTemplates(config, funcs)
			}
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
}
