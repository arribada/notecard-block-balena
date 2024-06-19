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
	card *notecard.Context
}

func handleError(w http.ResponseWriter, err error, msg string) {
	err_str := fmt.Sprintf("%s: %v", msg, err)
	http.Error(w, err_str, http.StatusInternalServerError)
	log.Print(err_str)
}

func (s *server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if s.card == nil {
		log.Fatal("notecard not initialized")
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

	card, err := notecard.OpenI2C("/dev/i2c-1", 0x17)
	if err != nil {
		return nil, fmt.Errorf("error opening Notecard: %v", err)
	}
	return card, nil
}

func main() {
	transport := os.Getenv("NOTECARD_TRANSPORT")
	if transport == "" {
		log.Printf("transport protocol not provided, defaulting to I2C...")
		transport = I2C
	}

	card, err := setupNotecard(transport)
	if err != nil {
		log.Fatalf("while setting up notecard: %v", err)
	}
	defer card.Close()

	http.Handle("/", &server{card: card})
	http.ListenAndServe(":3434", nil)
}
