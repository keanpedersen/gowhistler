package gowhistler

import (
	"bytes"
	"github.com/beevik/etree"
	"github.com/pkg/errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func init() {
	if !checkFileExists("cache") {
		os.Mkdir("cache", 0775)
	}
}

func getWSDL(url string) (doc *etree.Document, err error) {

	var raw io.ReadCloser

	if strings.HasPrefix(url, "http") {

		cacheFile := "cache/" + strings.ReplaceAll(url, "/", "_")
		if !strings.HasSuffix(cacheFile, ".wsdl") {
			cacheFile += ".wsdl"
		}
		if !checkFileExists(cacheFile) {

			log.Printf("Downloading %v\n", url)
			client := &http.Client{
				Timeout: time.Second * 30,
			}

			resp, err := client.Get(url)
			if err != nil {
				return nil, err
			}

			if resp.StatusCode != http.StatusOK {
				return nil, errors.New("Could not download")
			}

			content, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, errors.WithStack(err)
			}

			if err := os.WriteFile(cacheFile, content, 0664); err != nil {
				return nil, errors.WithStack(err)
			}
			if err := resp.Body.Close(); err != nil {
				return nil, errors.WithStack(err)
			}

			raw = io.NopCloser(bytes.NewReader(content))
		} else {
			//log.Printf("Using cache for %v\n", url)
		}

		f, err := os.Open(cacheFile)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		raw = f

	} else {
		log.Printf("Opening %v\n", url)
		f, err := os.Open(url)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		raw = f
	}

	doc = etree.NewDocument()
	if _, err := doc.ReadFrom(raw); err != nil {
		return nil, errors.WithStack(err)
	}

	if err := raw.Close(); err != nil {
		return nil, errors.WithStack(err)
	}

	return doc, nil
}

func checkFileExists(filePath string) bool {
	_, error := os.Stat(filePath)
	//return !os.IsNotExist(err)
	return !errors.Is(error, os.ErrNotExist)
}
