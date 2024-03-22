package main

import (
	"time"
)

func GetUser(username string, password string, salt []byte) (User, error) {
	hashed, err := HashPassword(password, salt)
	if err != nil {
		return User{}, nil
	}

	var user User
	err = db.QueryRow("SELECT * FROM users WHERE username = ? AND password = ? AND salt = ?;", username, hashed, salt).Scan(&user)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func IsValidUser(username string, password string) (bool, User, error) {
	var user User
	err := db.QueryRow("SELECT * FROM users WHERE username = ?;", username).Scan(&user.ID, &user.Username, &user.Password, &user.Salt, &user.Access)
	if err != nil {
		return false, User{}, err
	}

	valid, err := VerifyPassword(user.Password, password, user.Salt)
	if err != nil {
		return false, User{}, err
	}
	return valid, user, nil
}

func UpdatePosts() ([]Post, error) {
	var posts []Post

	rows, err := db.Query("SELECT * FROM posts;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	i := 0
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.ID, &post.Date, &post.Title, &post.Content); err != nil {
			return nil, err
		}
		posts = append(posts, post)
		i++
	}

	return posts, rows.Err()
}

func GetPostByID(posts []Post, id int) Post {
	for _, post := range posts {
		if post.ID == id {
			return post
		}
	}
	return Post{-1, time.Unix(0, 0), "", ""}
}

func DeletePost(id int) error {
	_, err := db.Exec("DELETE FROM posts WHERE id = ?;", id)
	return err
}

func UpdatePost(post Post) error {
	_, err := db.Exec("UPDATE posts SET date = ?, title = ?, content = ? WHERE id = ?;", post.Date, post.Title, post.Content, post.ID)
	return err
}

func CreatePost(post Post) error {
	_, err := db.Exec("INSERT INTO posts (date, title, content) VALUES (?, ?, ?);", post.Date, post.Title, post.Content)
	return err
}
