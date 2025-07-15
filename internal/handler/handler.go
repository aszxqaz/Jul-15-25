package handler

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/aszxqaz/fetch-zip/internal/domain"
	"github.com/aszxqaz/fetch-zip/internal/operator"
	"github.com/gin-gonic/gin"
)

type handler struct {
	repo domain.Repository
	oper operator.Operator
}

func New(repo domain.Repository, oper operator.Operator) *handler {
	return &handler{repo, oper}
}

func (h *handler) HandleCreateArchive(c *gin.Context) {
	archive, err := h.oper.CreateArchive()
	if err != nil {
		if operErr, ok := err.(*operator.Error); ok {
			if operErr.Code() == operator.ErrCodeMaxArchivesProcessing {
				c.JSON(http.StatusConflict, NewError(operErr.Error()))
				return
			}
		}
		c.JSON(http.StatusInternalServerError, NewError("Неизвестная ошибка"))
		return
	}
	if archive.DownloadUrl != "" {
		archive.DownloadUrl = getPublicUrl(archive.DownloadUrl, c.Request)
	}
	c.JSON(http.StatusCreated, archive)
}

func (h *handler) HandleGetArchive(c *gin.Context) {
	archiveID := c.Param("id")
	archive, ok := h.repo.GetArchive(archiveID)
	if !ok {
		c.JSON(http.StatusNotFound, NewError("Задача не найдена"))
		return
	}
	if archive.DownloadUrl != "" {
		archive.DownloadUrl = getPublicUrl(archive.DownloadUrl, c.Request)
	}
	c.JSON(http.StatusOK, archive)
}

func (h *handler) HandleAddFiles(c *gin.Context) {
	archiveID := c.Param("id")

	var body struct {
		Urls []string `json:"urls"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusUnprocessableEntity, NewError("Invalid json body"))
		return
	}

	if len(body.Urls) == 0 {
		c.JSON(http.StatusBadRequest, NewError("Urls list is empty"))
		return
	}

	archive, err := h.oper.HandleAddLinks(string(archiveID), body.Urls)
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err.Error()))
		return
	}

	if archive.DownloadUrl != "" {
		archive.DownloadUrl = getPublicUrl(archive.DownloadUrl, c.Request)
	}

	c.JSON(http.StatusCreated, archive)
}

func getPublicUrl(path string, req *http.Request) string {
	base := fmt.Sprintf("http://%s", req.Host)
	url, _ := url.JoinPath(base, path)
	return url
}
