// The MIT License (MIT)

// Copyright (c) 2016 Fabian Wenzelmann

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package elowl

import (
	"bytes"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/nu7hatch/gouuid"
)

const OntologyDir = ".ontologies"

type OntologyLib struct {
	Db      *sql.DB
	BaseDir string
}

func NewOntologyLib(db *sql.DB, baseDir string) *OntologyLib {
	return &OntologyLib{Db: db,
		BaseDir: baseDir}
}

func (lib *OntologyLib) InitDatabase(driver string) error {
	// URI should be unique, but that isn't possible because URLs can
	// become longer than the SQL length limit for keys, so well we only
	// use the last result we get
	var query string
	switch driver {
	case "sqlite3":
		query = `
  	CREATE TABLE IF NOT EXISTS ontologylib (
  		id INTEGER PRIMARY KEY AUTOINCREMENT,
  		uri TEXT NOT NULL,
  		local_path VARCHAR(32),
      checksum TEXT NOT NULL,
      UNIQUE(local_path)
  	);
  	`
	case "mysql":
		query = `
  	CREATE TABLE IF NOT EXISTS ontologylib (
  		id INT NOT NULL AUTO_INCREMENT,
  		uri TEXT NOT NULL,
  		local_path VARCHAR(32),
      checksum TEXT NOT NULL,
  		PRIMARY KEY(id),
      UNIQUE(local_path)
  	);
  	`
	default:
		return fmt.Errorf("Unsupported database driver: %s", driver)
	}

	if _, err := lib.Db.Exec(query); err != nil {
		return err
	}
	return nil
}

func (lib *OntologyLib) InitLocal() error {
	info, err := os.Stat(lib.BaseDir)
	if err != nil {
		return err
	}
	err = os.MkdirAll(path.Join(lib.BaseDir, OntologyDir), info.Mode())
	if err != nil {
		return err
	}
	return nil
}

func md5Str(data []byte) string {
	hasher := md5.New()
	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))
}

func (lib *OntologyLib) readFromURL(url *url.URL) ([]byte, error) {
	response, err := http.Get(url.String())
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	allContent, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return allContent, nil
}

func (lib *OntologyLib) RetrieveFromUrl(url *url.URL) ([]byte, error) {
	allContent, err := lib.readFromURL(url)
	if err != nil {
		return nil, err
	}
	uuid, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	fileName := strings.Replace(uuid.String(), "-", "", -1)
	filePath := path.Join(lib.BaseDir, OntologyDir, fileName+".owl")
	out, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer out.Close()
	// create new reader for file
	reader := bytes.NewReader(allContent)
	_, err = io.Copy(out, reader)
	if err != nil {
		return nil, err
	}
	md5Str := md5Str(allContent)
	// insert into database
	query := "INSERT INTO ontologylib(uri, local_path, checksum) VALUES(?, ?, ?)"
	_, dbErr := lib.Db.Exec(query, url.String(), fileName, md5Str)
	if dbErr != nil {
		// now we already created a file... so remove it
		fileErr := os.Remove(filePath)
		if fileErr != nil {
			log.Println("Can't remove previously created file, now it's sleepting there forever!")
		}
		return nil, dbErr
	}
	return allContent, nil
}

func (lib *OntologyLib) GetFile(url *url.URL, addLocally bool) (*bytes.Reader, error) {
	// first check if we have stored this element locally
	query := "SELECT local_path FROM ontologylib WHERE uri = ?"
	rows, err := lib.Db.Query(query, url.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var fileName string
	for rows.Next() {
		err = rows.Scan(&fileName)
		if err != nil {
			return nil, err
		}
	}
	if len(fileName) == 0 {
		// read from web
		var content []byte
		if addLocally {
			content, err = lib.RetrieveFromUrl(url)
		} else {
			content, err = lib.readFromURL(url)
		}
		if err != nil {
			return nil, err
		}
		fmt.Println("GOT FROM WEB")
		return bytes.NewReader(content), nil
	} else {
		// read from local file
		filePath := path.Join(lib.BaseDir, OntologyDir, fileName+".owl")
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		fmt.Println("GOT FROM FILE")
		return bytes.NewReader(content), nil
	}
}
