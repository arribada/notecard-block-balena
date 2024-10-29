package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/blues/note-go/notecard"
)

const (
	serial = "serial"
	i2c    = "i2c"

	defaultSerialDevice = "/dev/tty.usbmodemNOTE1"
	defaultSerialBaud   = "9600"
)

type server struct {
	// guards transaction
	muCard sync.Mutex
	card   *notecard.Context

	initError error
}

func getEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

const debug = false

func handleError(w http.ResponseWriter, err error, msg string) {
	err_str := fmt.Sprintf("%s: %v", msg, err)
	http.Error(w, err_str, http.StatusInternalServerError)
	log.Print(err_str)
}

func (s *server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if s.initError != nil {
		handleError(w, s.initError, "while initialising notecard")
		log.Fatal("Notecard not initialised, exiting...")
	}

	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		log.Printf("Method not allowed: %s", req.Method)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		handleError(w, err, "while reading request body")
		return
	}
	req.Body.Close()

	if debug {
		log.Printf("notecard request: %q", body)
	}

	s.muCard.Lock()
	note_rsp, err := s.card.TransactionJSON(body)
	if err != nil {
		handleError(w, err, "while performing a card transaction")
		s.muCard.Unlock()
		return
	}
	s.muCard.Unlock()

	if debug {
		log.Printf("notecard response: %q", note_rsp)
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

	if protocol != serial && protocol != i2c {
		return nil, fmt.Errorf("unsupported transport protocol: %v", protocol)
	}

	if protocol == serial {
		serialPort := getEnv("NOTECARD_SERIAL_DEVICE", defaultSerialDevice)

		baud, err := strconv.Atoi(getEnv("NOTECARD_SERIAL_BAUD", defaultSerialBaud))
		if err != nil {
			return nil, fmt.Errorf("error parsing NOTECARD_SERIAL_BAUD: %v", err)
		}

		card, err := notecard.OpenSerial(serialPort, baud)
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
	debug, err := strconv.ParseBool(getEnv("NOTECARD_DEBUG", "false"))
	if err != nil {
		log.Printf("Error parsing NOTECARD_DEBUG: %v", err)
	}

	if debug {
		log.Printf("Debug mode enabled")
	}

	transport := getEnv("NOTECARD_TRANSPORT", i2c)

	card, err := setupNotecard(transport)
	if err != nil {
		log.Printf("while setting up notecard: %v", err)
	} else {
		defer card.Close()
	}

	http.Handle("/", &server{card: card, initError: err})
	http.ListenAndServe(":3434", nil)
}
