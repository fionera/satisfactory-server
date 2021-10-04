package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

var gameRoot string

func init() {
	flag.StringVar(&gameRoot, "gameroot", "/config/gamefiles", "")
}

func main() {
	flag.Parse()

	modsPath := path.Join(gameRoot, "FactoryGame/Mods")
	mkdirAllIfNotExists(modsPath)

	mods := parseMods()
	if len(mods) == 0 {
		log.Println("No Mod IDs given. Deleting all!")
		if err := os.RemoveAll(modsPath); err != nil {
			log.Fatal(err)
		}
		return
	}

	binariesPath := path.Join(gameRoot, "FactoryGame/Binaries/Win64/")
	err := downloadLatestFile("satisfactorymodding", "SatisfactoryModBootstrapper", "msdia140.dll", binariesPath)
	if err != nil {
		log.Fatal(err)
	}

	err = downloadLatestFile("satisfactorymodding", "SatisfactoryModBootstrapper", "xinput1_3.dll", binariesPath)
	if err != nil {
		log.Fatal(err)
	}

	err = downloadLatestZipFile("satisfactorymodding", "SatisfactoryModLoader", "SML.zip", path.Join(modsPath, "SML"))
	if err != nil {
		log.Fatal(err)
	}

	for _, mod := range mods {
		if err := downloadZipFile(mod.downloadURL(), mod.Name(), path.Join(modsPath, mod.Name())); err != nil {
			log.Fatal(err)
		}
	}
}

type modInfoResp struct {
	Success bool `json:"success"`
	Data    struct {
		ID               string    `json:"id"`
		Name             string    `json:"name"`
		ShortDescription string    `json:"short_description"`
		FullDescription  string    `json:"full_description"`
		Logo             string    `json:"logo"`
		SourceURL        string    `json:"source_url"`
		CreatorID        string    `json:"creator_id"`
		Approved         bool      `json:"approved"`
		Views            int       `json:"views"`
		Downloads        int       `json:"downloads"`
		Hotness          int       `json:"hotness"`
		Popularity       int       `json:"popularity"`
		UpdatedAt        time.Time `json:"updated_at"`
		CreatedAt        time.Time `json:"created_at"`
	} `json:"data"`
}

type Mod struct {
	id      string
	version string
}

func (m Mod) Name() string {
	res, err := http.Get(fmt.Sprintf("https://api.ficsit.app/v1/mod/%s", m.id))
	if err != nil {
		return m.id
	}
	defer res.Body.Close()

	var respData modInfoResp
	if err := json.NewDecoder(res.Body).Decode(&respData); err != nil {
		return m.id
	}

	if respData.Data.Name == "" {
		return m.id
	}

	return respData.Data.Name
}

func (m Mod) downloadURL() string {
	return fmt.Sprintf("https://api.ficsit.app/v1/mod/%s/versions/%s/download", m.id, m.version)
}

func parseMods() (m []Mod) {
	modIDs := os.Getenv("MOD_IDS")
	if modIDs == "" {
		return nil
	}

	for _, mod := range strings.Split(modIDs, ",") {
		modParts := strings.SplitN(mod, ":", 2)
		if len(modParts) != 2 {
			log.Fatal("Invalid modID given")
		}

		modID, modVersion := modParts[0], modParts[1]
		m = append(m, Mod{modID, modVersion})
	}
	return m
}

func mkdirAllIfNotExists(name string) {
	if _, err := os.Stat(name); errors.Is(err, os.ErrNotExist) {
		err := os.MkdirAll(name, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func downloadLatestZipFile(owner, repo, fileName, dst string) error {
	return downloadZipFile(latestReleaseUrl(owner, repo, fileName), fileName, dst)
}

func downloadZipFile(url, fileName, dst string) error {
	log.Printf("fetching %q to %q", fileName, dst)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("invalid status: %q: %s", url, resp.Status)
	}

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	r, err := zip.NewReader(bytes.NewReader(buf), int64(len(buf)))
	if err != nil {
		return err
	}

	writeZipFile := func(zipFile *zip.File) error {
		dir, _ := path.Split(zipFile.Name)
		if dir != "" {
			mkdirAllIfNotExists(path.Join(dst, dir))
		}

		zipReader, err := zipFile.Open()
		if err != nil {
			return err
		}
		defer zipReader.Close()

		filePath := path.Join(dst, zipFile.Name)
		log.Printf("writing file %q", filePath)
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(file, zipReader)
		if err != nil {
			return err
		}

		return nil
	}

	mkdirAllIfNotExists(dst)
	for _, entry := range r.File {
		if err := writeZipFile(entry); err != nil {
			return err
		}
	}

	return nil
}

func downloadLatestFile(owner, repo, fileName, dst string) error {
	url := latestReleaseUrl(owner, repo, fileName)
	filePath := path.Join(dst, fileName)
	log.Printf("fetching %q to %q", fileName, filePath)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("invalid status: %q: %s", url, resp.Status)
	}

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func latestReleaseUrl(owner, repo, fileName string) string {
	return fmt.Sprintf("https://github.com/%s/%s/releases/latest/download/%s", owner, repo, fileName)
}
