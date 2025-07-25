package repo

import (
	"aumusic/internal/models"
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

func AddTrack(ctx context.Context, pool *pgxpool.Pool, track models.TrackDB) error {
	sql := "INSERT INTO tracks (user_id, name, artist, album, size, mod_time, path) VALUES ($1, $2, $3, $4, $5, $6, $7)"
	_, err := pool.Exec(ctx, sql, track.UserId, track.Name, track.Artist, track.Album, track.Size, track.ModTime, track.Path)
	if err != nil {
		return err
	}
	return nil
}

func GetTrack(ctx context.Context, pool *pgxpool.Pool, trackId string) (models.TrackDB, error) {
	sql := "SELECT user_id, name, artist, album, size, mod_time, path FROM tracks WHERE id = $1"
	var track models.TrackDB
	err := pool.QueryRow(ctx, sql, trackId).Scan(
		&track.UserId,
		&track.Name,
		&track.Artist,
		&track.Album,
		&track.Size,
		&track.ModTime,
		&track.Path,
	)
	if err != nil {
		return models.TrackDB{}, err
	}
	return track, nil
}

func GetTracksByUser(ctx context.Context, pool *pgxpool.Pool, userId string) ([]models.Track, error) {
	sql := "SELECT id, name, artist, album, size, mod_time FROM tracks WHERE user_id = $1"
	var tracks []models.Track
	rows, err := pool.Query(ctx, sql, userId)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var track models.Track
		err := rows.Scan(&track.Id, &track.Name, &track.Artist, &track.Album, &track.Size, &track.ModTime)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, track)
	}
	return tracks, nil
}

func DeleteTrack(ctx context.Context, pool *pgxpool.Pool, trackId string) error {
	sql := "DELETE FROM tracks WHERE id = $1"
	_, err := pool.Exec(ctx, sql, trackId)
	if err != nil {
		return err
	}
	return nil
}

func CreatePlaylist(ctx context.Context, pool *pgxpool.Pool, playlist models.Playlist) error {
	sql := "INSERT INTO playlists (user_id, name) VALUES ($1, $2)"
	_, err := pool.Exec(ctx, sql, playlist.UserId, playlist.Name)
	if err != nil {
		return err
	}
	return nil
}

func GetPlaylists(ctx context.Context, pool *pgxpool.Pool, userId string) ([]models.Playlist, error) {
	sql := "SELECT id, name FROM playlists WHERE user_id = $1"
	var playlists []models.Playlist
	rows, err := pool.Query(ctx, sql, userId)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var playlist models.Playlist
		err := rows.Scan(&playlist.Id, &playlist.Name)
		if err != nil {
			return nil, err
		}
		playlists = append(playlists, playlist)
	}
	return playlists, nil
}

func AddTrackToPlaylist(ctx context.Context, pool *pgxpool.Pool, playlistId string, trackId string) error {
	sql := "INSERT INTO playlist_tracks (playlist_id, track_id) VALUES ($1, $2)"
	_, err := pool.Exec(ctx, sql, playlistId, trackId)
	if err != nil {
		return err
	}
	return nil
}

func RemoveTrackFromPlaylist(ctx context.Context, pool *pgxpool.Pool, playlistId string, trackId string) error {
	sql := "DELETE FROM playlist_tracks WHERE playlist_id = $1 AND track_id = $2"
	_, err := pool.Exec(ctx, sql, playlistId, trackId)
	if err != nil {
		return err
	}
	return nil
}

func NewUser(ctx context.Context, pool *pgxpool.Pool, user models.User) error {
	sql := "INSERT INTO users (name, password, email) VALUES ($1, $2, $3)"
	_, err := pool.Exec(ctx, sql, user.Name, user.Pass, user.Email)
	if err != nil {
		return err
	}
	return nil
}

func GetUser(ctx context.Context, pool *pgxpool.Pool, username string) (models.User, error) {
	sql := "SELECT id, name, password, email FROM users WHERE name = $1"
	var user models.User
	err := pool.QueryRow(ctx, sql, username).Scan(&user.Id, &user.Name, &user.Pass, &user.Email)
	if err != nil {
		return models.User{}, err
	}
	return user, nil
}
