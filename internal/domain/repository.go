package domain

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

var ErrArchiveNotFound = errors.New("archive not found")
var ErrLinkNotFound = errors.New("link not found")

type Repository interface {
	CreateArchive() Archive
	GetArchive(archiveID string) (Archive, bool)
	AddLinks(archiveID string, urls []string) (Archive, bool)
	Transact(archiveID string, fn func(a Archive) (Archive, error)) (Archive, error)
	UpdateLink(l File, fn func(l File) File) (File, error)
}

type repository struct {
	mu    sync.RWMutex
	store map[string]Archive
}

func NewRepository() Repository {
	return &repository{
		store: make(map[string]Archive),
	}
}

func (r *repository) CreateArchive() Archive {
	r.mu.Lock()
	defer r.mu.Unlock()
	archiveID := uuid.NewString()
	archive := Archive{
		ID:        archiveID,
		CreatedAt: time.Now(),
		Status:    ArchiveStatusProcessing,
	}
	r.store[archiveID] = archive
	return archive
}

func (r *repository) GetArchive(archiveID string) (Archive, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	archive, ok := r.store[archiveID]
	if !ok {
		return Archive{}, false
	}
	return archive, true
}

func (r *repository) AddLinks(archiveID string, urls []string) (Archive, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	archive, ok := r.store[archiveID]
	if !ok {
		return Archive{}, false
	}

	links := make([]File, len(urls))
	for i, url := range urls {
		links[i] = File{
			ArchiveID: archiveID,
			Url:       url,
			AddedAt:   time.Now(),
		}
	}

	archive.Files = append(archive.Files, links...)
	r.store[archiveID] = archive
	return archive, true
}

func (r *repository) Transact(archiveID string, fn func(a Archive) (Archive, error)) (Archive, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	old, ok := r.store[archiveID]
	if !ok {
		return Archive{}, ErrArchiveNotFound
	}
	new, err := fn(old)
	if err != nil {
		return Archive{}, err
	}

	r.store[archiveID] = new
	return new, err
}

func (r *repository) UpdateLink(link File, fn func(l File) File) (File, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	archive, ok := r.store[link.ArchiveID]
	if !ok {
		return File{}, ErrArchiveNotFound
	}

	for i, l := range archive.Files {
		if l.ID == link.ID {
			link = fn(l)
			archive.Files[i] = link
			r.store[link.ArchiveID] = archive
			return link, nil
		}
	}

	return File{}, ErrLinkNotFound
}
