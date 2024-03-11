package main

type Config struct {
	Username   string `yaml:"username,omitempty"`
	Password   string `yaml:"password,omitempty"`
	XorKey     int    `yaml:"xor_key,omitempty"`
	AllowLogin bool   `yaml:"allow_login,omitempty"`

	DB struct {
		User     string `yaml:"user,omitempty"`
		Password string `yaml:"password,omitempty"`
		Address  string `yaml:"address,omitempty"`
		DBName   string `yaml:"db_name,omitempty"`
	} `yaml:"db,omitempty"`
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
}

type AdminPageData struct {
	Posts []Post

	IsAuthenticated bool
	Links           map[string]Link
}

type EditPageData struct {
	IsAuthenticated bool
	Links           map[string]Link

	Post    Post
	NewPost bool
}
