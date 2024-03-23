package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

/*
 * TODO: implement caching
 */

func ConnectDB(cfg mysql.Config) (*sql.DB, error) {
	_db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		fmt.Printf("Failed to connect to database: \n\t> %v\n", err)
		fmt.Printf("Retrying in 10secs\n")
		time.Sleep(10 * time.Second)
		return ConnectDB(cfg)
	}
	return _db, nil
}

/* ==========
 * == USER ==
 * ========== */

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

func GetUsers(user User) ([]User, error) {
	if user.IsAdmin() || user.CanManageUsers() {
		var users []User
		rows, err := db.Query("SELECT id, username, password, salt, access FROM users;")
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var user User
			if err := rows.Scan(&user.ID, &user.Username, &user.Password, &user.Salt, &user.Access); err != nil {
				return nil, err
			}
			users = append(users, user)
		}

		return users, nil
	}
	return nil, fmt.Errorf("user does not have required permissions")
}

func GetUserById(id int) User {
	users, err := GetUsers(User{Access: UserAccess_Admin}) // hacky, idk should probably change this a lot
	if err != nil {
		return EmptyUser
	}
	for _, u := range users {
		if u.ID == id {
			return u
		}
	}
	return EmptyUser
}

func DeleteUser(user User, to_delete User) error {
	if user.IsAdmin() {
		_, err := db.Exec("DELETE FROM users WHERE ID = ?;", to_delete.ID)
		return err
	}
	if user.CanEditUsers() || user.ID == to_delete.ID {
		_, err := db.Exec("DELETE FROM users WHERE ID = ?;", to_delete.ID)
		return err
	}
	return fmt.Errorf("user does not have required permissions")
}

func EditUser(current User, to_edit User) error {
	if current.IsAdmin() {
		_, err := db.Exec("UPDATE users SET username = ?, password = ?, salt = ?, access = ? WHERE id = ?;", to_edit.Username, to_edit.Password, to_edit.Salt, to_edit.Access, to_edit.ID)
		return err
	}
	if current.CanEditUsers() || current.ID == to_edit.ID {
		_, err := db.Exec("UPDATE users SET username = ?, password = ?, salt = ? WHERE id = ?;", to_edit.Username, to_edit.Password, to_edit.Salt, to_edit.ID)
		return err
	}
	return fmt.Errorf("user does not have required permissions")
}

func CreaterUser(current User, new User) error {
	if current.IsAdmin() {
		_, err := db.Exec("INSERT INTO users (username, password, salt, access) VALUES (?, ?, ?, ?)", new.Username, new.Password, new.Salt, new.Access)
		return err
	}
	if current.CanCreateUsers() {
		_, err := db.Exec("INSERT INTO users (username, password, salt, access) VALUES (?, ?, ?, ?)", new.Username, new.Password, new.Salt, UserAccess_ManagePosts)
		return err
	}
	return fmt.Errorf("user does not have required permissions")
}

/* ==========
 * == PAGE ==
 * ========== */

func UpdatePosts() ([]Post, error) {
	var posts []Post

	rows, err := db.Query("SELECT id, author, date, title, content FROM posts;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.ID, &post.Author, &post.Date, &post.Title, &post.Content); err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	return posts, rows.Err()
}

func GetPostByID(posts []Post, id int) Post {
	for _, post := range posts {
		if post.ID == id {
			return post
		}
	}
	return Post{-1, -1, time.Unix(0, 0), "", ""}
}

func DeletePost(user User, id int) error {
	if user.IsAdmin() {
		_, err := db.Exec("DELETE FROM posts WHERE id = ?;", id)
		return err
	}
	if user.CanEditPosts() { // user not admin but can edit posts
		_, err := db.Exec("DELETE FROM posts WHERE id = ? AND author = ?;", id)
		return err
	}
	return fmt.Errorf("user is not allowed to update posts")
}

func UpdatePost(user User, post Post) error {
	if user.IsAdmin() {
		_, err := db.Exec("UPDATE posts SET date = ?, title = ?, content = ? WHERE id = ?;", post.Date, post.Title, post.Content, post.ID)
		return err
	}
	if user.CanEditPosts() { // user not admin but can edit posts
		_, err := db.Exec("UPDATE posts SET date = ?, title = ?, content = ? WHERE id = ? AND author = ?;", post.Date, post.Title, post.Content, post.ID, user.ID)
		return err
	}
	return fmt.Errorf("user is not allowed to update posts")
}

func CreatePost(user User, post Post) error {
	if user.CanCreatePosts() {
		_, err := db.Exec("INSERT INTO posts (date, title, content, author) VALUES (?, ?, ?, ?);", post.Date, post.Title, post.Content, user.ID)
		return err
	}
	return fmt.Errorf("user is not allowed to create posts")
}
