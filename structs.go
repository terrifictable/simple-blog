package main

import (
	"bytes"
	"time"
)

const (
	UserAccess_None        = 0b00000
	UserAccess_CreatePosts = 0b00001
	UserAccess_EditPosts   = 0b00010
	UserAccess_ManagePosts = UserAccess_CreatePosts | UserAccess_EditPosts
	UserAccess_CreateUsers = 0b00100
	UserAccess_EditUsers   = 0b01000
	UserAccess_ManageUsers = UserAccess_CreateUsers | UserAccess_EditUsers
	UserAccess_Admin       = 0b10000 | UserAccess_ManagePosts | UserAccess_ManageUsers
)

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"` // hashed
	Salt     []byte `json:"-"`
	/*
	 * 0b00001 -> can create posts
	 * 0b00010 -> can edit posts
	 * 0b00100 -> can create users
	 * 0b01000 -> can edit users
	 * 0b10000 -> admin
	 */
	Access int `json:"access"` // 0b1111
}

var EmptyUser = User{}

func (u User) IsAdmin() bool {
	return u.Access&UserAccess_Admin == UserAccess_Admin
}

func (u User) CanCreatePosts() bool {
	return u.Access&UserAccess_CreatePosts == UserAccess_CreatePosts
}
func (u User) CanEditPosts() bool {
	return u.Access&UserAccess_EditPosts == UserAccess_EditPosts
}
func (u User) CanManagePosts() bool {
	return (u.CanEditPosts() && u.CanCreatePosts()) || u.IsAdmin()
}

func (u User) CanCreateUsers() bool {
	return u.Access&UserAccess_CreateUsers == UserAccess_CreateUsers
}
func (u User) CanEditUsers() bool {
	return u.Access&UserAccess_EditUsers == UserAccess_EditUsers
}
func (u User) CanManageUsers() bool {
	return (u.CanEditUsers() && u.CanCreateUsers()) || u.IsAdmin()
}

func (u User) ToString() string {
	insert := func(str *string, should bool, val string) {
		if should {
			if len(*str) != 0 {
				*str += " - "
			}
			*str += val
		} else {
			*str += ""
		}
	}
	var str string

	insert(&str, u.IsAdmin(), "admin")
	insert(&str, u.CanManageUsers(), "users")
	insert(&str, u.CanManagePosts(), "posts")
	return str
}

func (u User) Equals(o User) bool {
	return u.ID == o.ID &&
		u.Username == o.Username &&
		u.Password == o.Password &&
		bytes.Equal(u.Salt, o.Salt) &&
		u.Access == o.Access
}

type Post struct {
	ID      int       `json:"id"`
	Author  int       `json:"author"`
	Date    time.Time `json:"date"`
	Title   string    `json:"title"`
	Content string    `json:"content"`
}

var EmptyPost = Post{}

/* ===========
 * == PAGES ==
 * =========== */

type IndexPageData struct {
	Posts      []Post
	IsLoggedin bool
	AllowLogin bool
}

type PostPageData struct {
	Post Post
}

type LoginPageData struct {
	AllowLogin bool
}

type Link struct {
	Active bool
	HREF   string
	Name   string
	Before string
}

type Links map[string]Link

func (l *Links) Set(key string) {
	for k, v := range *l {
		v.Active = k == key
		(*l)[k] = v
	}
}

type AdminPageData struct {
	Posts []Post

	IsAuthenticated bool
	User            User
	Links           Links
}

type PostEditPageData struct {
	IsAuthenticated bool
	User            User
	Links           Links

	Post    Post
	NewPost bool
}

type UserEditPageData struct {
	IsAuthenticated bool
	User            User
	Links           Links

	ToEdit  User
	NewUser bool
}

type UsersPageData struct {
	IsAuthenticated bool
	Links           Links
	User            User
	Users           []User
}
