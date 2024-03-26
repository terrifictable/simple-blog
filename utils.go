package main

import "log"

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
