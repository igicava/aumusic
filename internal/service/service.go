package service

import (
	"io"
	"log"
	"os"
	"time"
)

const MEDIA = "/mnt/d/Музыка/"

func GetTrack(fileName string) (io.ReadSeeker, int64, time.Time, error) {
	file, err := os.Open(MEDIA + fileName)
	if err != nil {
		log.Println(err)
		return nil, 0, time.Time{}, err
	}

	fileInfo, err := file.Stat()
	if err != nil {
		log.Println(err)
		return nil, 0, time.Time{}, err
	}
	fileSize := fileInfo.Size()

	return file, fileSize, fileInfo.ModTime(), nil
}

func ListTracks() ([]string, error) {
	dir, err := os.Open(MEDIA)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	files, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	return files, nil
}
