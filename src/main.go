package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"google.golang.org/api/option"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

var ctx = context.Background()
var client *firestore.Client
var err error

type noteStruct struct {
	Section    string    `firestore:"section" json:"section"`
	DatePosted time.Time `firestore:"datePosted" json:"datePosted"`
	// Section type? Chapter vs page vs headline
}
type bookStruct struct {
	ID                  string       `firestore:"id" json:"id"`
	Title               string       `firestore:"title" json:"title"`
	Author              string       `firestore:"author" json:"author"`
	CoverURL            string       `firestore:"coverURL" json:"coverUrl"`
	Summary             string       `firestore:"summary" json:"summary"`
	CurrentPageNumber   int          `firestore:"currentPageNumber" json:"currentPageNumber"`
	TotalPageNumbers    int          `firestore:"totalPageNumbers" json:"totalPageNumbers"`
	LastUpdatedProgress time.Time    `firestore:"lastUpdatedProgress" json:"lastUpdatedProgress"`
	Notes               []noteStruct `firestore:"notes" json:"notes"`
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

	return result.Payload.Data
}

// Adds book
func addBook(bookData bookStruct, userID string) (bool, error) {
	log.Print("Attempint to add", bookData)
	var err error
	added := true
	book := client.Collection("users/" + userID + "/books").NewDoc()
	bookData.ID = book.ID
	bookData.LastUpdatedProgress = time.Now()
	_, err = book.Set(ctx, bookData)

	if err != nil {
		log.Panic("Failed to create book", err)
		added = false
	}
	return added, err
}

func getAllBooks(userID string) []byte {
	log.Println("Attempting to GET all books in this collection")
	books := client.Collection("users/" + userID + "/books")
	docsnaps, err := books.DocumentRefs(ctx).GetAll()
	booksData := []bookStruct{}
	if err != nil {
		log.Print("Error getting doc refs:", err)
		jsonData, _ := json.Marshal(booksData)
		return jsonData
	}
	for _, ds := range docsnaps {
		docsnap, err := ds.Get(ctx)
		if err != nil {
			log.Fatal("Error looping through document data", err)
		}
		var bookData bookStruct
		if err := docsnap.DataTo(&bookData); err != nil {
			log.Panic("Could not read data from book in", err)
			continue
		}
		log.Print(bookData)
		booksData = append(booksData, bookData)
	}
	response := make(map[string][]bookStruct)
	response["books"] = booksData
	jsonData, err := json.Marshal(response)
	if err != nil {
		log.Fatal("cannot convert to json")
	}
	return jsonData
}

func getBook(bookID string, userID string) ([]byte, error) {
	log.Printf("Attempting to GET %s books in this collection", bookID)
	var err error
	books := client.Collection("users/" + userID + "/books")
	docsnaps, err := books.DocumentRefs(ctx).GetAll()
	booksData := []bookStruct{}
	if err != nil {
		log.Panic("Error getting doc refs:", err)
		jsonData, _ := json.Marshal(booksData)
		return jsonData, err
	}
	for _, ds := range docsnaps {
		docsnap, err := ds.Get(ctx)
		if err != nil {
			log.Fatal("Error looping through document data", err)

		}
		var bookData bookStruct
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

func updateBook(bookData bookStruct, userID string) (bool, error) {
	var err error
	updated := true
	books := client.Collection("users/" + userID + "/books")
	docsnaps := books.Doc(bookData.ID)
	bookData.LastUpdatedProgress = time.Now()
	_, err = docsnaps.Set(ctx, bookData)
	if err != nil {
		log.Panic("Failed to update", err)
		updated = false
	}

	return updated, err
}

func deleteBook(bookID string, userID string) (bool, error) {
	var err error
	deleted := true
	books := client.Collection("users/" + userID + "/books")
	_, err = books.Doc(bookID).Delete(ctx)

	if err != nil {
		deleted = false
	}

	return deleted, err
}

// Calls Google Books API when user is adding a book
// to see if there is a match in metadata
func suggestBook(searchString string) [5]bookStruct {
	apiEndpoint := "https://www.googleapis.com/books/v1/volumes?q="

	var bookSuggestions [5]bookStruct
	fmt.Print(apiEndpoint)
	return bookSuggestions

}
func books(w http.ResponseWriter, r *http.Request) {
	log.Printf("books endpoint called")
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		// get a specific ID
		userID := r.Header.Get("UserID")
		if r.Header.Get("bookID") != "" {
			bookID := r.Header.Get("bookID")
			thisBook, err := getBook(bookID, userID)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusOK)
			}
			w.Write([]byte(thisBook))
		} else {
			// Get all books
			booksJSONData := getAllBooks(userID)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(booksJSONData))
		}

	case "POST":
		var bookData bookStruct
		log.Println("POST called")
		err := json.NewDecoder(r.Body).Decode(&bookData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			log.Panic("Failed to do POST", err)
			return
		}
		log.Printf(bookData.Author)
		userID := r.Header.Get("UserID")
		postAttemped, err := addBook(bookData, userID)
		log.Printf("ADD book called")
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
		var bookData bookStruct
		err := json.NewDecoder(r.Body).Decode(&bookData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		userID := r.Header.Get("UserID")

		putAttempted, err := updateBook(bookData, userID)
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
		if r.Header.Get("BookID") != "" {
			userID := r.Header.Get("UserID")
			bookID := r.Header.Get("BookID")
			deleteAttempted, err := deleteBook(bookID, userID)
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
func users(w http.ResponseWriter, r *http.Request) {
	log.Printf("Reached the users endpoint")

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "UP"}`))
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
	r.HandleFunc("/users", users)
	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "UserID", "BookID", "Content-Type"})
	originsOk := handlers.AllowedOrigins([]string{"http://localhost:3000"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "DELETE", "HEAD", "POST", "PUT", "OPTIONS"})

	log.Printf("Serving on localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", handlers.CORS(originsOk, headersOk, methodsOk)(r)))

}
