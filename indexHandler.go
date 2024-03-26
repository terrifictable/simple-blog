package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

func IndexPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}
	err := r.ParseForm()
	if err != nil {
		fmt.Printf("%v", err)
		return
	}

	posts, err = UpdatePosts()
	if err != nil {
		fmt.Printf("%v", err)
		return
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
	if config.DevMode {
		templates = LoadTemplates(config, funcs)
	}
	err = templates["website/index"].Execute(w, IndexPageData{
		IsLoggedin: login,
		AllowLogin: config.AllowLogin,
		Posts:      posts,
	})
	if err != nil {
		fmt.Printf("%v\n", err)
	}
}
