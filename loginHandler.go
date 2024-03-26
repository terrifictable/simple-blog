package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
)

func PostLoginPageHandler(w http.ResponseWriter, r *http.Request) {
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
}

func GetLoginPageHandler(w http.ResponseWriter, r *http.Request) {
	if config.DevMode {
		templates = LoadTemplates(config, funcs)
	}
	err := templates["website/login"].Execute(w, LoginPageData{
		AllowLogin: config.AllowLogin,
	})
	if err != nil {
		fmt.Printf("%v\n", err)
	}
}
