package main

type Config struct {
	AllowLogin bool `yaml:"allow_login,omitempty"`

	Prefix string `yaml:"prefix,omitempty"`

	DB struct {
		User     string `yaml:"user,omitempty"`
		Password string `yaml:"password,omitempty"`
		Address  string `yaml:"address,omitempty"`
		DBName   string `yaml:"db_name,omitempty"`
	} `yaml:"db,omitempty"`
}

type User struct {
	ID       int
	Username string
	Password string // 64 long
	Salt     []byte // 16 long
	Access   int
}

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

type AdminPageData struct {
	Posts []Post

	IsAuthenticated bool
	User            User
	Links           map[string]Link
}

type EditPageData struct {
	IsAuthenticated bool
	Links           map[string]Link

	Post    Post
	NewPost bool
}
