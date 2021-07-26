package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/rs/cors"

	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Post struct {
	Id        bson.ObjectId `bson:"_id" json:"id"`
	Text      string        `json:"text" bson:"text"`
	CreatedAt time.Time     `json:"createdAt" bson:"created_at"`
}

var posts *mgo.Collection

func main() {
	// Connect to mongo
	session, err := mgo.Dial("mongo:27017")
	if err != nil {
		log.Fatalln(err)
		log.Fatalln("mongo err")
		os.Exit(1)
	}
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)

	// Get posts collection
	posts = session.DB("app").C("posts2")

	// Set up routes
	r := mux.NewRouter()
	r.HandleFunc("/posts", createPost).
		Methods("POST")
	r.HandleFunc("/posts", readPosts).
		Methods("GET")
	r.HandleFunc("/posts/{id}", updatePost).
		Methods("PUT")
	r.HandleFunc("/posts/{id}", deletePost).
		Methods("DELETE")

	http.ListenAndServe(":8080", cors.AllowAll().Handler(r))
	log.Println("Listening on port 8080...")
}

func createPost(w http.ResponseWriter, r *http.Request) {
	// Read body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		responseError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Read post
	post := &Post{}
	err = json.Unmarshal(data, post)
	if err != nil {
		responseError(w, err.Error(), http.StatusBadRequest)
		return
	}
	post.Id = bson.NewObjectId()
	post.CreatedAt = time.Now().UTC()

	// Insert new post
	if err := posts.Insert(post); err != nil {
		responseError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	responseJSON(w, post)
}

func readPosts(w http.ResponseWriter, r *http.Request) {
	result := []Post{}
	if err := posts.Find(nil).Sort("-created_at").All(&result); err != nil {
		responseError(w, err.Error(), http.StatusInternalServerError)
	} else {
		responseJSON(w, result)
	}
}

func responseError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func responseJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func updatePost(w http.ResponseWriter, r *http.Request) {
	var err error

	//get id from incoming url
	vars := mux.Vars(r)
	id := bson.ObjectIdHex(vars["id"])

	//decode the incoming Note into json
	var postResource Post
	err = json.NewDecoder(r.Body).Decode(&postResource)
	if err != nil {
		panic(err)
	}

	//partial update on mongodb
	err = posts.Update(bson.M{"_id": id},
		bson.M{"$set": bson.M{
			"text": postResource.Text,
		}})

	if err == nil {
		log.Printf("Updated Post : %s", id, postResource.Text)
	} else {
		panic(err)
	}

	w.WriteHeader(http.StatusNoContent)
}

func deletePost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	err := posts.Remove(bson.M{"_id": bson.ObjectIdHex(id)})
	if err != nil {
		log.Printf("Could not find the Note %s to delete", id)
	}

	w.WriteHeader(http.StatusNoContent)
}
