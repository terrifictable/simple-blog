package main

import (
	"encoding/base64"
	"net/http"
	"strings"
)

func AuthenticateUser(w http.ResponseWriter, r *http.Request) (bool, User) {
	var user User
	var is_authenticated bool = false
	cookie, err := r.Cookie("token")
	if err == nil {
		split := strings.Split(cookie.Value, " ")
		raw_username, err := base64.StdEncoding.DecodeString(split[0])
		if err != nil {
			is_authenticated = false
			http.Error(w, "401 unauthorized", http.StatusUnauthorized)
			return false, EmptyUser
		}
		raw_password, err := base64.StdEncoding.DecodeString(split[1])
		if err != nil {
			is_authenticated = false
			http.Error(w, "401 unauthorized", http.StatusUnauthorized)
			return false, EmptyUser
		}
		username := string(xor(raw_username, xor_key))
		password := string(xor(raw_password, xor_key))

		is_authenticated, user, err = IsValidUser(username, password)
		if err != nil {
			is_authenticated = false
		}
	}
	return is_authenticated, user
}
