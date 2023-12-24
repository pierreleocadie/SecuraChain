package api_package

import (
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type ApiEndpointData struct {
	AnnouncementID        uuid.UUID `json:"announcementID"`        // Announcement ID - UUID
	OwnerAddress          []byte    `json:"ownerAddress"`          // Owner address - ECDSA public key
	Filename              []byte    `json:"filename"`              // Encrypted filename
	Extension             []byte    `json:"extension"`             // Encrypted extension
	FileSize              uint64    `json:"fileSize"`              // File size
	Checksum              []byte    `json:"checksum"`              // Checksum - SHA256
	OwnerSignature        []byte    `json:"ownerSignature"`        // Owner signature - ECDSA signature
	AnnouncementTimestamp int64     `json:"announcementTimestamp"` // Announcement timestamp - Unix timestamp
}

type Data []ApiEndpointData

// func allRequests(w http.ResponseWriter, r *http.Request) {

// 	StringChecksum := sha256.Sum256([]byte("Y2hlY2tzdW0="))
// 	checksum := StringChecksum[:]

// 	data := Data{
// 		ApiEndpointData{
// 			AnnouncementID:        uuid.New(),
// 			OwnerAddress:          []byte("MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEA0vjhgxf5S1ZVDx4Li7Q4F6WqWFNpocBK8V/sB/J7SOHyan0G83cPapq0tGmznP+68ur+C4Rtzoa+002vHOIag=="),
// 			Filename:              []byte("example_file"),
// 			Extension:             []byte(".txt"),
// 			FileSize:              uint64(1024),
// 			Checksum:              checksum,
// 			OwnerSignature:        []byte("MEYCIQDP7XYADNQQYwW9qWk6L54JWzKdHAxDzpqSVEThUiOnBgIhAJsXLE+CwZMI8C8lwxnXS+ddxsJivE6CqBTURFB8rql/"),
// 			AnnouncementTimestamp: 1701783508,
// 		},
// 	}

// 	fmt.Println("Endpoint Hit: All data Endpoint ")
// 	json.NewEncoder(w).Encode(data)
// }

func homePage(w http.ResponseWriter, r *http.Request) {
	// Ce qui est affiché sur la page
	fmt.Fprintf(w, "Homepage Endpoint Hit")

}

func testPostRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "test POST endpoint worked")
}

func HandleRequests() {

	myrouter := mux.NewRouter().StrictSlash(true)

	myrouter.HandleFunc("/", homePage)
	// myrouter.HandleFunc("/data", allRequests).Methods("GET")
	myrouter.HandleFunc("/data", testPostRequest).Methods("POST")
	log.Fatal(http.ListenAndServe(":8081", myrouter))
}
