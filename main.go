package main

import(
	"archive/zip"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const DEFAULT_SWAGGER_VERSION = "v3.43.0"

func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Println("usage: local-swagger-ui filename.json")
		os.Exit(1)
	}
	apiPath := args[1]
	apiName := filepath.Base(apiPath)
	fmt.Println("show swagger-ui for:", apiPath, apiName, "at http://localhost:8080/")
	endpoint := ":8080"
	path, err := cacheSwaggerDist()
	if err != nil {
		log.Fatalf("Cannot get swagger dist: %w\n", err)
	}
	z, err := zip.OpenReader(path)
	if err != nil {
		log.Fatalf("Cannot read zip file: %v\n", err)
	}
	prefix := "swagger-ui-3.43.0/dist/"
	for _, f := range z.File {
		tmp := strings.Split(f.Name, "/")
		if len(tmp) >= 2 && tmp[1] == "dist" {
			prefix = strings.Join(tmp[:2], "/")
			break
		}
	}
	prefixLen := len(prefix)
	files := make(map[string]*zip.File, 0)
	for _, f := range z.File {
		if strings.HasPrefix(f.Name, prefix) {
			files[f.Name[prefixLen:]] = f
		}
	}
	
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		} else if strings.HasSuffix(path, ".json") {// == "/" + apiName {
         http.ServeFile(w, r, apiPath)
			return
		}
		if f, ok := files[path]; ok {
			rc, err := f.Open()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			data, err := ioutil.ReadAll(rc)
			if path == "/index.html" {
				data = []byte(strings.Replace(string(data), "https://petstore.swagger.io/v2/swagger.json", apiName, -1))
			}
			rc.Close()
			http.ServeContent(w, r, path, f.Modified, bytes.NewReader(data))
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "Not Found: %q\n", path)
		}
	})
	err = http.ListenAndServe(endpoint, nil)
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func cacheSwaggerDist() (string, error) {
	version := os.Getenv("SWAGGER_RELEASE")
	if version == "" {
		version = DEFAULT_SWAGGER_VERSION
	}
	dir := os.Getenv("DOWNLOAD_DIRECTORY")
	if dir == "" {
		dir = os.Getenv("HOME") + "/Downloads"
	}
	path := dir + "/" + version + ".zip"
	if fileExists(path) {
		return path, nil
	}
	resp, err := http.Get(swaggerUrl(version))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	err = ioutil.WriteFile(path, body, 0644)
	return path, err
}

func swaggerUrl(version string) string {
	return "https://github.com/swagger-api/swagger-ui/archive/" + version + ".zip"
}
