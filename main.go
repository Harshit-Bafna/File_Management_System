package main

import (
	"authentication/config"
	"authentication/controllers"
	"authentication/utils"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	config.InitDB()

	go utils.DeleteExpiredFiles()

	port := 8080

	r := mux.NewRouter()

	r.HandleFunc("/register", controllers.RegisterHandler)
	r.HandleFunc("/login", controllers.LoginHandler)
	r.HandleFunc("/upload", controllers.UploadFileHandler)
	r.HandleFunc("/files", controllers.GetUserFilesHandler)
	r.HandleFunc("/search", controllers.SearchUserFilesHandler)
	r.HandleFunc("/share", controllers.ShareFileHandler)
	r.HandleFunc("/share/{file_id:[0-9]+}", controllers.AccessSharedFileHandler)

	fmt.Printf("Server started on port %d\n", port)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), r))
}
