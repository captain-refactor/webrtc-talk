package main

import (
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
)

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/connect", func(writer http.ResponseWriter, request *http.Request) {
		offer, err := ioutil.ReadAll(request.Body)
		defer request.Body.Close()
		if err != nil {
			panic(err)
		}
		player := NewPlayer()
		_, err = writer.Write(player.AcceptOffer(offer))
		if err != nil {
			panic(err)
		}
	})
	router.Handle("/", http.FileServer(http.Dir("media-player/public")))
	http.Handle("/", router)
	log.Fatalln(http.ListenAndServe(":4321", nil))
}
