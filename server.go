package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gorilla/mux"
	"google.golang.org/api/option"
)

var ctx = context.Background()
var client, err = firestore.NewClient(ctx, "tracer-f96fe", option.WithCredentialsFile("credentials.json"))

type noteStruct struct {
	Section    string    `firestore:"section"`
	DatePosted time.Time `firestore:"datePosted"`
	// Section type? Chapter vs page vs headline
}
type progressStruct struct {
	ProgressValue float32   `firestore:"progressValue"`
	LastUpdate    time.Time `firestore:"lastUpdate"`
}
type book struct {
	Title             string         `firestore:"title"`
	Author            string         `firestore:"author"`
	CoverURL          string         `firestore:"coverURL"`
	Summary           string         `firestore:"summary"`
	CurrentPageNumber int            `firestore:"currentPageNumber"`
	TotalPageNumbers  int            `firestore:"totalPageNumbers"`
	Progress          progressStruct `firestore:"progress"`
	Notes             []noteStruct   `firestore:"notes"`
}
type server struct{}

// Adds book
func addBook() {

}

func getAllBooks() []byte {
	log.Println("Attempting to GET all books in this collection")
	books := client.Collection("books")
	docsnaps, err := books.DocumentRefs(ctx).GetAll()
	var booksData []book
	if err != nil {
		log.Panic("Error getting doc refs:", err)
	}
	for _, ds := range docsnaps {
		docsnap, err := ds.Get(ctx)
		if err != nil {
			log.Fatal("Error looping through document data", err)
		}
		var bookData book
		if err := docsnap.DataTo(&bookData); err != nil {
			log.Panic("Could not read data from book in", err)
			continue
		}
		log.Print(bookData)
		booksData = append(booksData, bookData)
	}
	jsonData, err := json.Marshal(booksData)
	if err != nil {
		log.Fatal("cannot convert to json")
	}
	return jsonData
}

func getBook(bookID string) {
	log.Printf("Attempting to GET %s books in this collection", bookID)

}

func updateBook() {

}

func deleteBook() {

}

// Calls Google Books API when user is adding a book
// to see if there is a match in metadata
func suggestBook() {

}
func books(w http.ResponseWriter, r *http.Request) {
	log.Printf("books endpoint called")
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		// get a specific ID
		if r.Header.Get("bookID") != "" {
			bookID := r.Header.Get("bookID")
			getBook(bookID)
		} else {
			// Get all books
			booksJsonData := getAllBooks()
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(booksJsonData))
		}

	case "POST":
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"message": "post called"}`))
	case "PUT":
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"message": "put called"}`))
	case "DELETE":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "delete called"}`))
	default:
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "not found"}`))
	}
}
func health(w http.ResponseWriter, r *http.Request) {
	log.Printf("Reached the health endpoint")

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "UP"}`))
}

func main() {
	if err != nil {
		log.Fatal("Failed to connect to firestore due to: ", err)
	}

	log.Printf("Starting the application server")
	r := mux.NewRouter()
	log.Printf("Started the application server")
	r.HandleFunc("/health", health)
	r.HandleFunc("/books", books)

	log.Printf("Serving on localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))

}
