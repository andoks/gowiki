package main

import (
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"bytes"
	"bufio"
)

var (
	validPath     = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")
	linkPath      = regexp.MustCompile(`\[.+\]`)
	templates     = template.Must(template.ParseFiles("templates/edit.html", "templates/view.html"))
	dataDir       = "data"
)

type Page struct {
	Title string
	Body  []byte
}

func (p *Page) save() error {
	filename := dataDir + "/" + p.Title + ".txt"
	log.Printf("saving page '%s' as file '%s'", p.Title, filename)
	return ioutil.WriteFile(filename, p.Body, 0600)
}

func setup() {
	// create data directory if it doesn't exist
	info, err := os.Stat(dataDir)
	if err != nil {
		log.Printf("creating data directory '%s'", dataDir)
		err = os.Mkdir(dataDir, 0700)
		if err != nil {
			log.Panicln("unable to create data folder")
		}
	} else {
		if !info.IsDir() {
			log.Panicf("'%s' path exists, but is not a directory", dataDir)
		}
	}
}

func loadPage(title string) (*Page, error) {
	filename := dataDir + "/" + title + ".txt"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return &Page{Title: title, Body: body}, nil
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func renderTemplateWithWikiSyntax(w http.ResponseWriter, tmpl string, p *Page) {
	buffer := bytes.Buffer{}
	writer := bufio.NewWriter(&buffer)
	err := templates.ExecuteTemplate(writer, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	writer.Flush()
	s := buffer.String()
	s = linkPath.ReplaceAllStringFunc(s, func(m string) string {
		page := m[1 : len(m)-1]
		return "<a href=\"/view/" + page + "\">" + page + "</a>"
	})

	w.Write([]byte(s))
}

func makeHandle(handle func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
		}
		title := m[2]
		log.Printf("request started for action '%s' on page '%s' from %v received", m[1], title, r.RemoteAddr)
		defer func() {
			log.Printf("request finished for action '%s' on page '%s' from %v received", m[1], title, r.RemoteAddr)
		}()
		handle(w, r, title)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplateWithWikiSyntax(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/view/FrontPage", http.StatusPermanentRedirect)
}

func main() {
	setup()
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/view/", makeHandle(viewHandler))
	http.HandleFunc("/edit/", makeHandle(editHandler))
	http.HandleFunc("/save/", makeHandle(saveHandler))
	http.ListenAndServe(":8080", nil)
}
