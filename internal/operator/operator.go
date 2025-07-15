package operator

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"slices"
	"sync"
	"time"

	"github.com/aszxqaz/fetch-zip/internal/domain"
	"github.com/aszxqaz/fetch-zip/internal/zipper"
	"github.com/google/uuid"
)

type Operator interface {
	HandleAddLinks(archiveID string, urls []string) (domain.Archive, error)
	CreateArchive() (domain.Archive, error)
}

type Config struct {
	SupportedTypes        []string
	MaxArchivesProcessing int
	MaxFilesPerArchive    int
	ArchivesDir           string
}

type aJob struct {
	link     domain.File
	src      []byte
	filename string
}

type operator struct {
	config  Config
	repo    domain.Repository
	zipper  zipper.Zipper
	aJobs   map[string]chan aJob
	aJobsMu sync.Mutex
}

func New(config Config, repo domain.Repository, zipper zipper.Zipper) Operator {
	o := &operator{
		config: config,
		repo:   repo,
		zipper: zipper,
		aJobs:  make(map[string]chan aJob),
	}
	return o
}

func (o *operator) CreateArchive() (domain.Archive, error) {
	o.aJobsMu.Lock()
	defer o.aJobsMu.Unlock()
	if len(o.aJobs) == o.config.MaxArchivesProcessing {
		msg := fmt.Sprintf("Превышен лимит одновременно обрабатываемых задач (%d)", o.config.MaxArchivesProcessing)
		return domain.Archive{}, NewError(ErrCodeMaxArchivesProcessing, msg)
	}
	archive := o.repo.CreateArchive()
	job := make(chan aJob)
	go o.consumeArchiveJobs(job)
	o.aJobs[archive.ID] = job
	return archive, nil
}

func (o *operator) HandleAddLinks(archiveID string, urls []string) (domain.Archive, error) {
	archive, err := o.repo.Transact(
		archiveID,
		func(archive domain.Archive) (domain.Archive, error) {
			if archive.ValidFileUrlsCount() == o.config.MaxFilesPerArchive {
				msg := fmt.Sprintf("Превышен лимит объектов на задачу (%d)", o.config.MaxFilesPerArchive)
				return domain.Archive{}, NewError(ErrCodeMaxLinksPerArchive, msg)
			}

			var wg sync.WaitGroup
			var mu sync.Mutex
			for _, url := range urls {
				wg.Add(1)
				go func() {
					mu.Lock()
					defer mu.Unlock()
					link := domain.File{
						ID:        uuid.NewString(),
						ArchiveID: archiveID,
						Url:       url,
						AddedAt:   time.Now(),
					}
					_, err := o.fetchUrl(url, true)
					if err != nil {
						link.Error = err.Error()
					} else {
						if archive.ValidFileUrlsCount() < o.config.MaxFilesPerArchive {
							go o.download(link)
						} else {
							link.Error = fmt.Sprintf("Превышен лимит объектов на задачу (%d)", o.config.MaxFilesPerArchive)
						}
					}
					archive.Files = append(archive.Files, link)
					if archive.ValidFileUrlsCount() == o.config.MaxFilesPerArchive {
						archive.Url = path.Join(o.config.ArchivesDir, fmt.Sprintf("%s.zip", archive.ID))
					}
					wg.Done()
				}()
			}
			wg.Wait()
			return archive, nil
		},
	)

	if err != nil {
		if errors.Is(err, domain.ErrArchiveNotFound) {
			msg := fmt.Sprintf("Задача с id %s не существует", archiveID)
			return domain.Archive{}, NewError(ErrCodeArchiveNotFound, msg)
		}
		return domain.Archive{}, err
	}

	return archive, nil
}

func (o *operator) fetchUrl(url string, prefetch bool) (*http.Response, error) {
	var err error
	var rsp *http.Response
	if prefetch {
		rsp, err = http.Head(url)
	} else {
		rsp, err = http.Get(url)
	}

	if err != nil {
		msg := "Не удалось отправить запрос на удаленный ресурс"
		return nil, NewError(ErrCodeRemoteResource, msg)
	}

	if rsp.StatusCode != 200 {
		msg := fmt.Sprintf("Удаленный ресурс ответил ошибкой: status code %d", rsp.StatusCode)
		return nil, NewError(ErrCodeRemoteResource, msg)
	}

	contentTypes, ok := rsp.Header["Content-Type"]
	if !ok {
		msg := "Невозможно определить тип файла на удаленном ресурсе"
		return nil, NewError(ErrCodeContentTypeUnset, msg)
	}

	supported := false
	for _, ct := range contentTypes {
		if slices.Contains(o.config.SupportedTypes, ct) {
			supported = true
			break
		}
	}
	if !supported {
		msg := fmt.Sprintf("Неподдерживаемый тип файла на удаленном ресурсе: %s", contentTypes)
		return nil, NewError(ErrCodeContentTypeUnsupported, msg)
	}

	return rsp, nil
}

func (o *operator) download(link domain.File) {
	rsp, err := o.fetchUrl(link.Url, false)
	if err != nil {
		o.repo.UpdateLink(link, func(link domain.File) domain.File {
			link.Error = err.Error()
			return link
		})
		return
	}

	src, err := io.ReadAll(rsp.Body)
	if err != nil {
		o.repo.UpdateLink(link, func(link domain.File) domain.File {
			link.Error = err.Error()
			return link
		})
		return
	}

	o.repo.UpdateLink(link, func(link domain.File) domain.File {
		link.Downloaded = true
		return link
	})

	o.aJobs[link.ArchiveID] <- aJob{
		link:     link,
		src:      src,
		filename: path.Base(rsp.Request.URL.Path),
	}

}

func (o *operator) consumeArchiveJobs(aJobs chan aJob) {
	for job := range aJobs {
		zipFileName := fmt.Sprintf("%s.zip", job.link.ArchiveID)
		zipFilePath := path.Join(o.config.ArchivesDir, zipFileName)
		err := o.zipper.Upsert(zipFilePath, job.filename, job.src)
		if err != nil {
			o.repo.UpdateLink(job.link, func(l domain.File) domain.File {
				l.Error = "Не удалось запаковать файл в архив"
				return l
			})
		}
		o.repo.UpdateLink(job.link, func(l domain.File) domain.File {
			l.Archived = true
			return l
		})

		archive, _ := o.repo.GetArchive(job.link.ArchiveID)
		if archive.ArchivedFilesCount() == o.config.MaxFilesPerArchive {
			close(aJobs)
			delete(o.aJobs, job.link.ArchiveID)
			o.repo.Transact(archive.ID, func(a domain.Archive) (domain.Archive, error) {
				a.Status = domain.ArchiveStatusDone
				return a, nil
			})
		}
	}
}
