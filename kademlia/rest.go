package kademlia

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

func (k *Kademlia) StartRESTServer(addr string) {
	http.HandleFunc("/objects", k.handleObjects)
	http.HandleFunc("/objects/", k.handleObjectByHash)

	log.Printf("Starting REST server at %s\n", addr)
	http.ListenAndServe(addr, nil)
}

// POST /objects
func (k *Kademlia) handleObjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data, err := io.ReadAll(r.Body)

	if err != nil || len(data) == 0 {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	value := string(data)

	key, success := k.IterativeStore(value, true)
	if !success {
		http.Error(w, "Failed to store object in the DHT: "+key, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Location", "/objects/"+key)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(key))
}

func (k *Kademlia) handleObjectByHash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	hash := r.URL.Path[len("/objects/"):]
	if len(hash) != 40 {
		http.Error(w, "Invalid key length", http.StatusBadRequest)
		return
	}
	key := NewKademliaID(hash)

	contacts, value := k.IterativeFindValue(key, 3, 20)
	if value == nil {
		http.Error(w, fmt.Sprintf("Object %s not found, closest nodes: %v", key, contacts), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(*value))

}
