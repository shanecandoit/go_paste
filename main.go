package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Page of pasted content
// we get the Body and Title from the user
// the hash is computed and becomes the url
// Title is a file that contains Hash
type Page struct {
	Title string // user given string
	Hash  string // base64 hash of Body
	Body  []byte
}

// TitleToHash has all titles and their Hash values
//var TitleToHash map[string]string = make(map[string]string)

func bytesToBase64Url(userText []byte) string {
	sum := sha256.Sum256(userText)
	var bts []byte = make([]byte, len(sum))
	for i := 0; i < len(sum); i++ {
		bts[i] = sum[i]
	}
	//copy(bts, sum)
	b64 := base64.URLEncoding.EncodeToString(bts)
	b64 = strings.ReplaceAll(b64, "=", "")
	return b64
}

func cleanFileName(path string) string {
	// https://stackoverflow.com/questions/1976007/what-characters-are-forbidden-in-windows-and-linux-directory-names
	path = strings.ReplaceAll(path, "/", "")
	path = strings.ReplaceAll(path, "\\", "")
	path = strings.ReplaceAll(path, "<", "")
	path = strings.ReplaceAll(path, ">", "")
	path = strings.ReplaceAll(path, ":", "")
	path = strings.ReplaceAll(path, "\"", "")
	path = strings.ReplaceAll(path, "|", "")
	path = strings.ReplaceAll(path, "?", "")
	path = strings.ReplaceAll(path, "*", "")
	path = filepath.Clean(path)
	return path
}

// save a Page
// save page
func (p *Page) save() error {
	fmt.Println("save")

	// limit text to 2mb
	twoMb := 2 * 1024 * 1024
	if len(p.Body) > twoMb {
		fmt.Println("big file being truncated", p.Title)
		p.Body = p.Body[:twoMb]
	}

	// get hash if not set
	if p.Hash == "" && p.Body != nil {
		p.Hash = bytesToBase64Url(p.Body)
	}

	// clean title, removes //..// but what about #?$
	//p.Title = filepath.Clean(p.Title)
	p.Title = cleanFileName(p.Title)

	fmt.Println("save", p.Title, p.Hash)

	// read from disk ?
	//TitleToHash[p.Title] = p.Hash

	start := time.Now().String()
	start = strings.ReplaceAll(start, " ", "-")
	start = start[:len("2019-11-14 12:22:45.860962")]
	fmt.Println("save", p.Title, "at", start)
	// 2019-11-14 12:22:45.860962

	// save hash in title
	ioutil.WriteFile("pages/"+p.Title+".hsh", []byte(p.Hash), 0600)
	fmt.Println("saved", p.Title+".hsh")

	titleName := p.Hash + ".txt"
	err := ioutil.WriteFile("pages/"+titleName, p.Body, 0600)
	fmt.Println("saved", titleName)
	return err
}

// loadPage
func loadPage(title string) (*Page, error) {
	fmt.Println("loadPage")

	title = cleanFileName(title)

	// is it in map?
	//v := TitleToHash[title]
	v := ""
	if v != "" {
		fmt.Println("load", title, v)

		filename := v + ".txt"
		body, err := ioutil.ReadFile("pages/" + filename)
		if err != nil {
			fmt.Println("Page:loadPage", title, "ERR", err)
			return nil, err
		}
		hash := bytesToBase64Url(body)

		// it was not so put it there
		//TitleToHash[title] = hash

		return &Page{Title: title, Hash: hash, Body: body}, nil
	}

	// not in map
	// does TestPage.hsh exist
	hashLookup := title + ".hsh"
	if _, err := os.Stat("pages/" + hashLookup); os.IsNotExist(err) {
		fmt.Println("not in mem, not on disk:", hashLookup)
		// path/to/whatever does not exist
		return nil, err
	}

	// read title file to get hash name
	b64, err := ioutil.ReadFile("pages/" + hashLookup)
	if err != nil {
		fmt.Println("err", err)
	}

	b64Filename := strings.TrimSpace(string(b64)) + ".txt"
	fmt.Print("load from disk", b64Filename)
	body, err := ioutil.ReadFile("pages/" + b64Filename)
	if err != nil {
		fmt.Println("Page:loadPage", title, "ERR", err)
		return nil, err
	}

	hash := bytesToBase64Url(body)

	return &Page{Title: title, Hash: hash, Body: body}, nil
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("defaultHandler")

	//fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
	//title := r.URL.Path[len("/view/"):]
	//p, _ := loadPage(title)
	//fmt.Fprintf(w, "<h1>%s</h1><div>%s</div>", p.Title, p.Body)
	tmpl := template.Must(template.ParseFiles("home.html"))

	// if GET send home.html
	if r.Method != http.MethodPost {
		fmt.Println("GET")
		tmpl.Execute(w, nil)
		return
	}

	fmt.Println("POST")
	userText := r.FormValue("text")
	urlHash := bytesToBase64Url([]byte(r.FormValue("message")))

	formDetails := Page{
		Title: r.FormValue("title"),
		//Subject: r.FormValue("subject"),
		//Message: r.FormValue("message"),
		Hash: urlHash,
		Body: []byte(userText),
	}
	formDetails.save()

	// go view that page
	http.Redirect(w, r, "/view/"+r.FormValue("title"), http.StatusFound)

	//fmt.Fprintf(w, homePage)
	//formCont := PageAndMap{page: &formDetails, TitlesToHashes: &TitleToHash}
	//tmpl.Execute(w, formDetails)
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("viewHandler")

	title := r.URL.Path[len("/view/"):]
	p, _ := loadPage(title)
	fmt.Println("view loaded, title", p.Title)
	fmt.Println("view loaded, hash ", p.Hash)
	fmt.Println("view loaded, body ", len(p.Body))

	tmpl := template.Must(template.ParseFiles("view.html"))
	tmpl.Execute(w, p)
}

func main() {
	fmt.Println("start")

	p1 := &Page{Title: "TestPage", Body: []byte("This is a sample Page.")}
	p1.save()
	p2, _ := loadPage("TestPage")
	fmt.Println(string(p2.Body))

	http.HandleFunc("/", defaultHandler)
	http.HandleFunc("/view/", viewHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
