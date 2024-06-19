package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/blues/note-go/notecard"
)

const (
	Serial = "serial"
	I2C    = "i2c"
)

type server struct {
	card      *notecard.Context
	initError error
}

var transport = os.Getenv("NOTECARD_TRANSPORT")

func handleError(w http.ResponseWriter, err error, msg string) {
	err_str := fmt.Sprintf("%s: %v", msg, err)
	http.Error(w, err_str, http.StatusInternalServerError)
	log.Print(err_str)
}

func (s *server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if s.card == nil || s.initError != nil {
		handleError(w, s.initError, "while initialising notecard")
		log.Fatal("Notecard not initialised, exiting...")
	}

	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		log.Printf("%s: Method not allowed", w)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		handleError(w, err, "while reading request body")
		return
	}
	req.Body.Close()

	note_rsp, err := s.card.TransactionJSON(body)
	if err != nil {
		handleError(w, err, "while performing a card transaction")
		return
	}
	log.Printf("notecard response: %s", note_rsp)

	// Set the Content-Type header to application/json
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(note_rsp)
	if err != nil {
		log.Printf("error writing response: %v", err)
		return
	}
}

func setupNotecard(protocol string) (*notecard.Context, error) {
	log.Printf("Setting up Notecard with protocol: %s\n", protocol)

	if protocol != Serial && protocol != I2C {
		return nil, fmt.Errorf("unsupported transport protocol: %v", protocol)
	}

	if protocol == Serial {
		card, err := notecard.OpenSerial("/dev/tty.usbmodemNOTE1", 9600)
		if err != nil {
			return nil, fmt.Errorf("error opening Notecard: %v", err)
		}
		return card, nil
	}

	card, err := notecard.OpenI2C("", 0x17)
	if err != nil {
		return nil, fmt.Errorf("error opening Notecard: %v", err)
	}

	status := map[string]interface{}{
		"req": "card.status",
	}
	if _, err = card.Transaction(status); err != nil {
		return nil, fmt.Errorf("error querying Notecard status: %v", err)
	}
	return card, nil
}

func main() {
	if transport == "" {
		log.Printf("transport protocol not provided, defaulting to I2C...")
		transport = I2C
	}

	card, err := setupNotecard(transport)
	if err != nil {
		log.Printf("while setting up notecard: %v", err)
	}
	defer card.Close()

	http.Handle("/", &server{card: card, initError: err})
	http.ListenAndServe(":3434", nil)
}
