package main

import (
	"log"
	"net/http"
	"runtime"
)

func Handle(err error) {
	if err != nil {
		pc, _, _, _ := runtime.Caller(1)
		details := runtime.FuncForPC(pc)
		log.Println("Fatal: " + err.Error() + " - " + details.Name())
	}
}

func WriteError(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	log.Println("http error:" + message)
	_, err := w.Write([]byte(message))
	Handle(err)
}
