package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

var router = mux.NewRouter()
var db *sql.DB

func initDB() {

	var err error
	config := mysql.Config{
		User:                 "root",
		Passwd:               "secret",
		Addr:                 "127.0.0.1:3306",
		Net:                  "tcp",
		DBName:               "go_simple_blog",
		AllowNativePasswords: true,
	}

	// 准备数据库连接池
	db, err = sql.Open("mysql", config.FormatDSN())
	checkError(err)

	// set the maximum number of connections.
	db.SetMaxOpenConns(25)
	// set the maximum number of idle connections.
	db.SetMaxIdleConns(25)
	// set expiration time for each link.
	db.SetConnMaxLifetime(5 * time.Minute)

	// try to connect, an error will be  reported if it fails.
	err = db.Ping()
	checkError(err)
	fmt.Println("mysql connected!")
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "<h1>Hello, welcome to my goblog!</h1>")
}

func aboutHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "This blog is used to record programming notes. If you have feedback or suggestions, please contact"+
		"<a href=\"mailto:3267666759@qq.com\">3267666759@qq.com</a>")
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, "<h1>Requested page not found :(</h1><p>If you have questions, please contact us.</p>")
}

func articlesShowHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	fmt.Fprint(w, "article ID:"+id)
}

func articlesIndexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Access article list.")
}

// ArticlesFormData 创建博文表单数据
type ArticlesFormData struct {
	Title, Body string
	URL         *url.URL
	Errors      map[string]string
}

func articlesStoreHandler(w http.ResponseWriter, r *http.Request) {

	title := r.PostFormValue("title")
	body := r.PostFormValue("body")

	errors := make(map[string]string)

	// 验证标题
	if title == "" {
		errors["title"] = "The title can not be blank"
	} else if utf8.RuneCountInString(title) < 3 || utf8.RuneCountInString(title) > 40 {
		errors["title"] = "Title lenght needs to be between 3-40"
	}

	// 验证内容
	if body == "" {
		errors["body"] = "The content can not be blank"
	} else if utf8.RuneCountInString(body) < 10 {
		errors["body"] = "Content lenght needs to be greater than or equal to 10 bytes"
	}

	// 检查是否有错误
	if len(errors) == 0 {
		fmt.Fprint(w, "verifiction passed!<br>")
		fmt.Fprintf(w, "value of title: %v <br>", title)
		fmt.Fprintf(w, "lenght of title: %v <br>", utf8.RuneCountInString(title))
		fmt.Fprintf(w, "value of body: %v <br>", body)
		fmt.Fprintf(w, "lenght of body: %v <br>", utf8.RuneCountInString(body))
	} else {
		storeURL, _ := router.Get("articles.store").URL()

		data := ArticlesFormData{
			Title:  title,
			Body:   body,
			URL:    storeURL,
			Errors: errors,
		}
		tmpl, err := template.ParseFiles("resources/views/articles/create.gohtml")
		if err != nil {
			panic(err)
		}

		err = tmpl.Execute(w, data)
		if err != nil {
			panic(err)
		}
	}
}

func forceHTMLMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. set header
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// 2. continue processing the request
		next.ServeHTTP(w, r)
	})
}
func removeTrailingSlash(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. remove the diagonal bar behind all request paths except the homepage
		if r.URL.Path != "/" {
			r.URL.Path = strings.TrimSuffix(r.URL.Path, "/")
		}

		// 2. pass the request on 
		next.ServeHTTP(w, r)
	})
}

func articlesCreateHandler(w http.ResponseWriter, r *http.Request) {
	storeURL, _ := router.Get("articles.store").URL()
	data := ArticlesFormData{
		Title:  "",
		Body:   "",
		URL:    storeURL,
		Errors: nil,
	}
	tmpl, err := template.ParseFiles("resources/views/articles/create.gohtml")
	if err != nil {
		panic(err)
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		panic(err)
	}
}

func main() {
	initDB()

	router.HandleFunc("/", homeHandler).Methods("GET").Name("home")
	router.HandleFunc("/about", aboutHandler).Methods("GET").Name("about")

	router.HandleFunc("/articles/{id:[0-9]+}", articlesShowHandler).Methods("GET").Name("articles.show")
	router.HandleFunc("/articles", articlesIndexHandler).Methods("GET").Name("articles.index")
	router.HandleFunc("/articles", articlesStoreHandler).Methods("POST").Name("articles.store")
	router.HandleFunc("/articles/create", articlesCreateHandler).Methods("GET").Name("articles.create")

	// custom 404 page
	router.NotFoundHandler = http.HandlerFunc(notFoundHandler)

	// middleware: force content type to HTML
	router.Use(forceHTMLMiddleware)

	http.ListenAndServe(":3000", removeTrailingSlash(router))
}
