package domain

type File struct {
	ID         string `json:"id"`
	Url        string `json:"url"`
	Downloaded bool   `json:"downloaded"`
	Archived   bool   `json:"archived"`
	Error      string `json:"error,omitempty"`
	ArchiveID  string `json:"-"`
}

const (
	ArchiveStatusProcessing = "В обработке"
	ArchiveStatusDone       = "Готово"
)

type Archive struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Files       []File `json:"files,omitempty"`
	DownloadUrl string `json:"download_url,omitempty"`
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
