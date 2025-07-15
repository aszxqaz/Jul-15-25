package domain

import (
	"time"
)

type File struct {
	ID         string    `json:"id"`
	Url        string    `json:"url"`
	Downloaded bool      `json:"downloaded"`
	Archived   bool      `json:"archived"`
	Error      string    `json:"error,omitempty"`
	AddedAt    time.Time `json:"added_at"`
	ArchiveID  string    `json:"-"`
}

const (
	ArchiveStatusProcessing = "В обработке"
	ArchiveStatusDone       = "Готово"
)

type Archive struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	Files     []File    `json:"files,omitempty"`
	Url       string    `json:"url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func (a Archive) ValidFileUrlsCount() int {
	count := 0
	for _, file := range a.Files {
		if file.Error == "" {
			count++
		}
	}
	return count
}

func (a Archive) ArchivedFilesCount() int {
	count := 0
	for _, file := range a.Files {
		if file.Archived {
			count++
		}
	}
	return count
}
