package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/go-ego/riot"
	"github.com/go-ego/riot/types"
	"github.com/gorilla/mux"
	yaml "gopkg.in/yaml.v2"
)

var searcher = riot.New("en")
var AllPosts []*Post

type Post struct {
	Title              string   `json:"title" yaml:"title"`
	TitleEnglish       string   `json:"title_english" yaml:"title_english"`
	Description        string   `json:"description" yaml:"description"`
	DescriptionEnglish string   `json:"description_english" yaml:"description_english"`
	EntryURL           string   `json:"entry_url" yaml:"entry_url"`
	ImageURL           string   `json:"image_url" yaml:"image_url"`
	ImageALT           string   `json:"image_alt" yaml:"image_alt"`
	ImageALTEnglish    string   `json:"image_alt_english" yaml:"image_alt_english"`
	PublishedAt        string   `json:"published_at" yaml:"published_at"`
	Categories         []string `json:"categories" yaml:"categories"`
	CategoriesEnglish  []string `json:"categories_english" yaml:"categories_english"`
	AllEnglish         string   `json:"all_english" yaml:"all_english"`
}

func ParsePost(path string) (*Post, error) {
	var post Post
	text, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(text, &post); err != nil {
		return nil, err
	}
	return &post, nil

}

func indexEntries(folderPath string) error {

	files, err := ioutil.ReadDir(folderPath)
	if err != nil {
		return err
	}
	for i, f := range files {
		docID := uint64(i)
		fullPathSource := fmt.Sprintf("%s%s", folderPath, f.Name())
		post, err := ParsePost(fullPathSource)
		if err != nil {
			fmt.Println(fmt.Sprintf("error while indexing post: %s", fullPathSource))
			return err
		}
		searcher.IndexDoc(docID, types.DocIndexData{Content: post.TitleEnglish})
		searcher.IndexDoc(docID, types.DocIndexData{Content: post.DescriptionEnglish})
		searcher.IndexDoc(docID, types.DocIndexData{Content: post.ImageALTEnglish})
		searcher.IndexDoc(docID, types.DocIndexData{Content: post.AllEnglish})
		for _, cat := range post.CategoriesEnglish {
			searcher.IndexDoc(docID, types.DocIndexData{Content: cat})
		}

		AllPosts = append(AllPosts, post)

	}
	return nil
}

func search(query string) []Post {
	req := types.SearchReq{Text: query}
	posts := []Post{}
	searchResult := searcher.Search(req)
	for _, doc := range searchResult.Docs.(types.ScoredDocs) {
		posts = append(posts, *AllPosts[doc.DocId])
	}
	return posts
}

func SearchHandler(w http.ResponseWriter, r *http.Request) {
	words := r.URL.Query().Get("query")
	posts := search(words)
	jsonBytes, err := json.Marshal(posts)
	if err != nil {
		fmt.Println("JSON Marshal error:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, string(jsonBytes))
}

func main() {
	entriesFolder := os.Args[1]
	fmt.Println("indexing..")
	err := indexEntries(entriesFolder)
	if err != nil {
		fmt.Println("error occurred while indexing entries..")
		panic(err)
	}
	searcher.Flush()
	fmt.Println("finished indexing..")
	r := mux.NewRouter().Host("irasutoya.alejandro.pictures").Subrouter()
	r.Handle("/", http.FileServer(http.Dir("./static")))
	r.HandleFunc("/search", SearchHandler)
	http.ListenAndServe(":8000", r)
}
