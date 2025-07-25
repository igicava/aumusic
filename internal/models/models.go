package models

import "time"

type Track struct {
	Id      string    `json:"id"`
	Name    string    `json:"name"`
	Artist  string    `json:"artist"`
	Album   string    `json:"album"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}

type TrackDB struct {
	Id      string
	UserId  string
	Name    string
	Artist  string
	Album   string
	Size    int64
	Path    string
	ModTime time.Time
}

type Playlist struct {
	Id     string `json:"id"`
	UserId string `json:"user_id"`
	Name   string `json:"name"`
}

type User struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Pass  string `json:"pass"`
	Email string `json:"email"`
}
