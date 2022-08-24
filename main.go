package main

import (
	"encoding/json"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// Лень подключать сюда конфиг-файл *.yaml, опишем константами)
const (
	ADDRES  = "192.168.101.87"
	WEBPORT = ":8011"
	FTPPORT = ":5460"
	FILEDIR = "upload"
)

type (
	Router     struct{}
	FileEntity struct {
		Name      string `json:"name"`
		Extension string `json:"extension"`
		Addres    string `json:"url"`
		Size      int64  `json:"size"`
		IsDir     bool   `json:"folder"`
	}
)

var (
	// Сюда кешируем содержимое файл-сервера
	fileCashe = make([]FileEntity, 0)
)

// Маршрутизация и обработка запросов
func (R Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		if r.Method == http.MethodGet && (r.FormValue("data") == "json" || r.FormValue("find") == "") {
			bufer, err := json.Marshal(fileCashe)
			if err != nil {
				log.Println(err)
			} else {
				w.Write(bufer)
			}
		} else if r.Method == http.MethodGet && r.FormValue("find") != "" {
			searchRes, err := fileSearch(fileCashe, r.FormValue("find"))
			if err != nil {
				log.Println(err)
				w.Write([]byte("Ошибка поиска"))
			}
			bufer, err := json.Marshal(searchRes)
			if err != nil {
				log.Println(err)
			} else {
				w.Write(bufer)
			}
		} else {
			templ, err := template.ParseFiles("templates/template.html")
			if err != nil {
				log.Println(err)
			}
			w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Add("Access-Control-Allow-Origin", "*")
			w.Header().Add("Content-Type", "*")
			templ.Execute(w, nil)
		}
	case "/upload":
		file, header, err := r.FormFile("fileUpload")
		if err != nil {
			w.Write([]byte("Ошибка загрузки файла"))
			return
		}
		defer file.Close()

		data, err := ioutil.ReadAll(file)
		if err != nil {
			w.Write([]byte("Ошибка обработк файла"))
			return
		}
		findInCashe(data, &header.Filename, &fileCashe) //--Чекнем в кеш...
		filepath := FILEDIR + "/" + header.Filename
		err = ioutil.WriteFile(filepath, data, 0777)
		if err != nil {
			w.Write([]byte("Не удалось сохранить файл на сервер"))
			return
		}

		w.Write([]byte("Файл загружен на сервер\n"))
	default:
		//...
	}
}

// Поиск файлов в кеше по фрагментам/имени
func fileSearch(fileList []FileEntity, pattern string) ([]FileEntity, error) {
	var fileBufer []FileEntity
	for _, fi := range fileList {
		matched, err := regexp.MatchString(pattern, fi.Name)
		if err != nil {
			return []FileEntity{}, err
		}
		if matched {
			fileBufer = append(fileBufer, FileEntity{fi.Name, fi.Extension, fi.Addres, fi.Size, fi.IsDir})
		}
	}
	return fileBufer, nil
}

// Добавляем файлы в кеш, проверка на наличие дубликатов файлов имен...
func findInCashe(data []byte, fileName *string, fileCashe *[]FileEntity) {
	pattern, _ := regexp.Compile(`\.\w+$`)
	match := pattern.FindAllStringSubmatch(*fileName, 1)
	for _, file := range *fileCashe {
		if file.Name == *fileName {
			*fileName = time.Now().Format(time.RFC1123) + *fileName
		}
	}
	*fileCashe = append(*fileCashe, FileEntity{*fileName, match[0][0], "http://" + ADDRES + FTPPORT + "/" + *fileName, int64(len(data)), false})
}

// Проверяем наличие файлов на сервере, кешируем...
func checkTheFiles(dirName string, fileCashe *[]FileEntity) error {
	files, err := os.ReadDir(dirName)
	if err != nil {
		return err
	}
	for _, file := range files {
		if !file.IsDir() {
			entity, err := os.Open(dirName + "/" + file.Name())
			if err != nil {
				return err
			}
			defer entity.Close()

			fi, err := entity.Stat()
			if err != nil {
				return err
			}
			*fileCashe = append(*fileCashe, FileEntity{fi.Name(), filepath.Ext(fi.Name()), "http://" + ADDRES + FTPPORT + "/" + fi.Name(), fi.Size(), false})
		}
	}
	return nil
}

func main() {
	// С низкого старта кешируем существующие файлы
	err := checkTheFiles(FILEDIR, &fileCashe)
	if err != nil {
		log.Println("Ошибка кеширования файлов сервера! Сервер Остановлен!", err)
		return
	}

	http.Handle("/", Router{})

	wserv := &http.Server{
		Addr:         WEBPORT,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	go wserv.ListenAndServe()

	ftpserv := &http.Server{
		Addr:         FTPPORT,
		Handler:      http.FileServer(http.Dir(FILEDIR)),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	ftpserv.ListenAndServe()
}
