package main

import (
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type (
	// Структура для хранения данных о загруженном файле
	dirFiles struct {
		Name, Extension, Adress string
		Size                    int64
		IsDir                   bool
	}
	TemplateData struct {
		Files []dirFiles
		Title string
	}
	Router struct{}
)

const (
	ip             string = "192.168.101.87"
	uploadPath     string = "upload"
	fileServerPort string = ":5460"
)

var (
	templatesPaths = []string{
		"./templates/template.html",
		"./templates/tempData.html",
	}
	srv      = &http.Server{
		Addr:         ":8080",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
)

//Фильтр содержимого файлсервера
func filter(fileArr []dirFiles, pattern string){

}

// Функция находит файлы и папки в директории; возвращает: []dirFiles{}, error
func listFtpFiles(directory string) ([]dirFiles, error) {
	files, err := os.ReadDir(directory)
	if err != nil {
		return []dirFiles{}, err
	}
	thisDirFiles := make([]dirFiles, len(files))
	for i, file := range files {
		if !file.IsDir() {
			entity, err := checkFileInfo(directory + "/" + file.Name())
			if err != nil {
				return []dirFiles{}, err
			} else {
				thisDirFiles[i] = entity
			}
		} else {
			thisDirFiles[i] = dirFiles{file.Name(), "Папка", "", 0, true}
		}
	}
	return thisDirFiles, nil
}

// Функция получает информацию о файле; взвращает: структуру типа dirFiles{}, error
func checkFileInfo(fileName string) (dirFiles, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return dirFiles{}, err
	}
	defer file.Close()
	fi, err := file.Stat()
	if err != nil {
		return dirFiles{}, err
	}
	fileEntity := dirFiles{fi.Name(), filepath.Ext(fi.Name()), "http://" + ip + fileServerPort + "/" + fi.Name(), fi.Size(), false}
	return fileEntity, nil
}

// Маршрутизатор для http-запросов
func (e Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
			templ, err := template.ParseFiles(templatesPaths...)
			if err!=nil{
				log.Println(err)
				return
			}

		fileStack, err := listFtpFiles(uploadPath)
		if err != nil {
			log.Println(err)
			return
		}
		td := TemplateData{fileStack, "Облако"}
		if r.Method==http.MethodGet && r.FormValue("filter")!=""{
			filterKey:=r.FormValue("filter")
			filterVal:=r.FormValue("pattrn")
			log.Println(filterKey, filterVal)
			templ.ExecuteTemplate(w, "fileData",td)
		} else{
			err=templ.ExecuteTemplate(w, "main",td)
			if err!=nil{
				log.Println(err)
				return
			}
		}
	case "/exchange":
		w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-type", "multipart/form-data")
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
		filepath := uploadPath + "/" + header.Filename
		err = ioutil.WriteFile(filepath, data, 0777)
		if err != nil {
			w.Write([]byte("Не удалось сохранить файл на сервер"))
			return
		}
		w.Write([]byte("Файл загружен на сервер\n"))
	default:
		w.Write([]byte("Запрошенной страницы не существует"))
	}
}

func main() {
	http.Handle("/", Router{})
	go srv.ListenAndServe()

	fs := &http.Server{
		Addr:         fileServerPort,
		Handler:      http.FileServer(http.Dir(uploadPath)),
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}
	fs.ListenAndServe()
}
