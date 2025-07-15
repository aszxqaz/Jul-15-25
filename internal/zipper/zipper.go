package zipper

import (
	"archive/zip"
	"io"
	"os"
	"path"

	"github.com/google/uuid"
)

type Zipper interface {
	Upsert(zipFilename string, path string, src []byte) error
}

type zipper struct{}

func New() Zipper {
	return &zipper{}
}

func (z *zipper) Upsert(zipFilename string, filename string, src []byte) error {
	zf, err := os.OpenFile(zipFilename, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0777)
	if err != nil {
		return err
	}
	defer zf.Close()

	stat, err := zf.Stat()
	if err != nil {
		return err
	}

	if stat.Size() == 0 {
		zw := zip.NewWriter(zf)
		defer zw.Close()
		w, err := zw.Create("./" + filename)
		if err != nil {
			return err
		}
		_, err = w.Write(src)
		if err != nil {
			return err
		}
		return err
	}

	zr, err := zip.NewReader(zf, stat.Size())
	if err != nil {
		return err
	}

	outZipFilename := path.Join(zipFilename, "..", uuid.NewString()+".zip")
	outZipFile, err := os.Create(outZipFilename)
	if err != nil {
		return err
	}
	defer outZipFile.Close()

	zipWriter := zip.NewWriter(outZipFile)
	defer zipWriter.Close()

	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		w, err := zipWriter.CreateHeader(&f.FileHeader)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, rc)
		if err != nil {
			return err
		}
	}

	newFileWriter, err := zipWriter.Create("./" + filename)
	if err != nil {
		return err
	}
	_, err = newFileWriter.Write(src)
	if err != nil {
		return err
	}

	os.Rename(outZipFilename, zipFilename)
	return nil
}
