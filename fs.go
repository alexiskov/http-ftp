package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Router struct{}

// Структура для хранения данных о загруженном файле
type dirFiles struct {
	Name, Extension string
	Size            int64
	IsDir           bool
}

const(
  ip string = "192.168.101.87"
  uploadPath = "upload"
)

// Функция находит файлы и папки в директории; возвращает: []dirFiles{}, error
func listFtpFiles(directory string) ([]dirFiles,error) {
	files, err := os.ReadDir(directory)
	if err != nil {
		return []dirFiles{}, err
	}
	thisDirFiles:=make([]dirFiles, len(files))
	for i, file := range files {
		if !file.IsDir() {
			entity, err := checkFileInfo(file.Name())
			if err != nil {
				return []dirFiles{}, err
			} else{
        thisDirFiles[i]=entity
      }
		} else {
			thisDirFiles[i]=dirFiles{file.Name(),"Папка", 0, true,}
		}
	}
	return thisDirFiles,nil
}

// Функция получает информацию о файле; взвращает: структуру типа dirFiles{}, error
func checkFileInfo(directory string) (dirFiles, error) {
	file, err := os.Open(directory)
	if err != nil {
		return dirFiles{}, err
	}
	defer file.Close()
  fi, err:=file.Stat()
  if err!=nil{
    return dirFiles{}, err
  }
	fileEntity := dirFiles{fi.Name(), filepath.Ext(fi.Name()), fi.Size(), false}
	return fileEntity, nil
}

//Маршрутизатор для http-запросов
func (e Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		w.Write([]byte("Главная"))
	case "/filter":
		w.Write([]byte("Поиск"))
	case "/ftp":
		fileStack, err := listFtpFiles(uploadPath)
		if err != nil {
			log.Println(err)
			return
		}
    log.Println(fileStack)
	case "/upload":
		file, header, err := r.FormFile("fileupload")
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

	srv := &http.Server{
		Addr:         ":8011",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	srv.ListenAndServe()

}
