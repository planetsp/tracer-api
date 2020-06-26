package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/gorilla/mux"
	"google.golang.org/api/option"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

var ctx = context.Background()
var client *firestore.Client
var err error

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
	ID                string         `firestore:"id"`
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

func getCredentials() []byte {
	secretClient, err := secretmanager.NewClient(ctx)

	if err != nil {
		log.Fatalf("failed to setup client: %v", err)
	}
	name := "projects/tracer-f96fe/secrets/firestore_credentials_dev/versions/latest"

	// projectID := "tracer-f96fe"
	// Create the request to create the secret.

	// Build the request.
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	}

	// Call the API.
	result, err := secretClient.AccessSecretVersion(ctx, req)
	if err != nil {
		log.Panicf("failed to access secret version: %v", err)
		return nil
	}

	// WARNING: Do not print the secret in a production environment - this snippet
	// is showing how to access the secret material.
	log.Printf("Plaintext: %s\n", string(result.Payload.Data))
	return result.Payload.Data
}

// Adds book
func addBook(bookData book) (bool, error) {
	var err error
	added := true
	books := client.Collection("books")
	book := books.Doc(bookData.ID)

	_, err = book.Create(ctx, bookData)

	if err != nil {
		log.Panic("Failed to create book", err)
		added = false
	}
	return added, err
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

func getBook(bookID string) ([]byte, error) {
	log.Printf("Attempting to GET %s books in this collection", bookID)
	var err error
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
		if bookData.ID == bookID {
			booksData = append(booksData, bookData)
		}
	}

	jsonData, err := json.Marshal(booksData)
	if err != nil {
		log.Fatal("cannot convert to json")
	}
	if len(booksData) == 0 {
		err = mux.ErrNotFound
	}
	return jsonData, err
}

func updateBook(bookData book) (bool, error) {
	var err error
	updated := true
	books := client.Collection("books")
	docsnaps := books.Doc(bookData.ID)
	_, err = docsnaps.Set(ctx, bookData)
	if err != nil {
		log.Panic("Failed to update", err)
		updated = false
	}

	return updated, err
}

func deleteBook(bookID string) (bool, error) {
	var err error
	deleted := true
	books := client.Collection("books")
	_, err = books.Doc(bookID).Delete(ctx)

	if err != nil {
		deleted = false
	}

	return deleted, err
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
			thisBook, err := getBook(bookID)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusOK)
			}
			w.Write([]byte(thisBook))
		} else {
			// Get all books
			booksJSONData := getAllBooks()
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(booksJSONData))
		}

	case "POST":
		var bookData book
		err := json.NewDecoder(r.Body).Decode(&bookData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		postAttemped, err := addBook(bookData)

		if err != nil {
			log.Panic(err)
		} else if postAttemped {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"message": "post called"}`))
		} else {
			w.WriteHeader(http.StatusExpectationFailed)
			w.Write([]byte(`{"message": "post called"}`))
		}

	case "PUT":
		var bookData book
		err := json.NewDecoder(r.Body).Decode(&bookData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		putAttempted, err := updateBook(bookData)
		if err != nil {
			log.Panic(err)
		} else if putAttempted {
			w.WriteHeader(http.StatusAccepted)
			w.Write([]byte(`{"message": "put called"}`))
		} else {
			w.WriteHeader(http.StatusExpectationFailed)
			w.Write([]byte(`{"message": "put called and failed to update"}`))
		}

	case "DELETE":
		if r.Header.Get("bookID") != "" {
			deleteAttempted, err := deleteBook(r.Header.Get("bookID"))
			if err != nil {
				log.Panic(err)
			} else if deleteAttempted {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"message": "delete successful"}`))
			}
			w.WriteHeader(http.StatusExpectationFailed)
			w.Write([]byte(`{"message": "delete called"}`))
		} else {
			w.WriteHeader(http.StatusExpectationFailed)
			w.Write([]byte(`{"developerText": "failed to delete bookID"}`))
		}

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
	creds := getCredentials()

	client, err = firestore.NewClient(ctx, "tracer-f96fe", option.WithCredentialsJSON(creds))
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
