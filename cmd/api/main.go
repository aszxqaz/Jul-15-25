package main

import (
	"github.com/aszxqaz/fetch-zip/internal/domain"
	"github.com/aszxqaz/fetch-zip/internal/handler"
	"github.com/aszxqaz/fetch-zip/internal/operator"
	"github.com/aszxqaz/fetch-zip/internal/zipper"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.Static("/static/archives", "./static/archives")

	api := r.Group("/api")

	repo := domain.NewRepository()
	zipper := zipper.New()
	oper := operator.New(operator.Config{
		SupportedTypes:        []string{"image/jpeg", "application/pdf"},
		MaxArchivesProcessing: 3,
		MaxFilesPerArchive:    3,
		ArchivesDir:           "./static/archives",
	}, repo, zipper)
	h := handler.New(repo, oper)

	api.POST("/archives", h.HandleCreateArchive)
	api.GET("/archives/:id", h.HandleGetArchive)
	api.POST("/archives/:id/files", h.HandleAddFiles)

	r.Run(":8080")
}
