package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/blues/note-go/notecard"
)

type Transport string

const (
	Serial Transport = "serial"
	I2C    Transport = "i2c"
)

var card *notecard.Context

func serveNotecard(w http.ResponseWriter, req *http.Request) {

	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "Method not allowed")
		return
	}

	var data map[string]interface{}
	if err := json.NewDecoder(req.Body).Decode(&data); err != nil {
		http.Error(w, fmt.Sprintf("Error decoding JSON: %v", err), http.StatusBadRequest)
		return
	}

	for key, value := range data {
		fmt.Printf("%s: %v\n", key, value)
	}

	jsonString, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		w.WriteHeader(http.StatusInternalServerError)
		panic(err)
	}

	// Print the resulting JSON string
	fmt.Println(string(jsonString))

	note_rsp, err := card.Transaction(data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		panic(err)
	}

	note_rsp_json, err := json.Marshal(note_rsp)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		w.WriteHeader(http.StatusInternalServerError)
		panic(err)
	}

	// Set the Content-Type header to application/json
	w.Header().Set("Content-Type", "application/json")
	w.Write(note_rsp_json)

	w.WriteHeader(http.StatusOK)
}

func setupNotecard(protocol Transport) *notecard.Context {
	var card *notecard.Context
	var err error

	fmt.Printf("Setting up Notecard with protocol: %s\n", protocol)

	if protocol == Transport("serial") {
		card, err = notecard.OpenSerial("/dev/tty.usbmodemNOTE1", 9600)
		if err != nil {
			fmt.Printf("Error opening Notecard: %v\n", err)
			panic(err)
		}
	} else if protocol == Transport("i2c") {
		card, err = notecard.OpenI2C("/dev/i2c-1", 0x17)
		if err != nil {
			fmt.Printf("Error opening Notecard: %v\n", err)
			panic(err)
		}
	} else {
		fmt.Printf("Missing transport protocol\n")
	}
	return card
}

func main() {
	var i interface{} = os.Getenv("NOTECARD_TRANSPORT")
	transport, err := i.(Transport)

	if !err {
		fmt.Printf("Error getting transport protocol, defaulting to I2C...\n")
		transport = I2C
	}

	card = setupNotecard(transport)
	defer card.Close()

	http.HandleFunc("/", serveNotecard)
	http.ListenAndServe(":3434", nil)
}
