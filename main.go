package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
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

	// prepare database connection pool
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

// Article  Corresponding to an article data
type Article struct {
	Title, Body string
	ID          int64
}

func articlesShowHandler(w http.ResponseWriter, r *http.Request) {
	// 1. get URL parameters
	id := getRouteVariable("id", r)

	// 2. read the corresponding article data
	article, err := getArticleByID(id)

	// 3. if an error occurs
	if err != nil {
		if err == sql.ErrNoRows {
			// 3.1 data not found
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "404 article not found!")
		} else {
			// 3.2 database error
			checkError(err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "500 server internal error!")
		}
	} else {
		// 4. read successfully
		tmpl, err := template.ParseFiles("resources/views/articles/show.gohtml")
		checkError(err)

		err = tmpl.Execute(w, article)
		checkError(err)
	}
}

func getRouteVariable(parameterName string, r *http.Request) string {
	vars := mux.Vars(r)
	return vars[parameterName]
}

func getArticleByID(id string) (Article, error) {
	article := Article{}
	query := "SELECT * FROM articles WHERE id = ?"
	err := db.QueryRow(query, id).Scan(&article.ID, &article.Title, &article.Body)
	return article, err
}

func articlesEditHandler(w http.ResponseWriter, r *http.Request) {

	// 1. get URL parameters
	id := getRouteVariable("id", r)

	// 2. read the corresponding article data
	article, err := getArticleByID(id)

	// 3. If there have an error.
	if err != nil {
		if err == sql.ErrNoRows {
			// 3.1 Data not found.
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "404 article not found!")
		} else {
			// 3.2 Database error.
			checkError(err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "500 internal server error!")
		}
	} else {
		// 4. Read successful, display the form.
		updateURL, _ := router.Get("articles.update").URL("id", id)
		data := ArticlesFormData{
			Title:  article.Title,
			Body:   article.Body,
			URL:    updateURL,
			Errors: nil,
		}
		tmpl, err := template.ParseFiles("resources/views/articles/edit.gohtml")
		checkError(err)

		err = tmpl.Execute(w, data)
		checkError(err)
	}
}

func articlesUpdateHandler(w http.ResponseWriter, r *http.Request) {
	// 1. get article ID.
	id := getRouteVariable("id", r)
	// 2. get article content by ID
	_, err := getArticleByID(id)

	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "404 article not found!")
		} else {
			// database error
			checkError(err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "500 internal server error.")
		}
	} else {
		// No errors.

		// Check the form.
		title := r.PostFormValue("title")
		body := r.PostFormValue("body")
		errors := validateArticleFormData(title, body)

		if len(errors) == 0 {
			// No errors

			// Form validation passed, updateing data.
			query := "UPDATE articles SET title = ?, body = ? WHERE id = ?"
			rs, err := db.Exec(query, title, body, id)

			if err != nil {
				checkError(err)
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, "500 Internal server error!")
			}

			// Update successful, redirecting to the article details page.
			if n, _ := rs.RowsAffected(); n > 0 {
				showURL, _ := router.Get("articles.show").URL("id", id)
				http.Redirect(w, r, showURL.String(), http.StatusFound)
			} else {
				fmt.Fprint(w, "You haven't made any changes!")
			}
		} else {
			// 4.3 Form validation failed, displaying reasons.

			updateURL, _ := router.Get("articles.update").URL("id", id)
			data := ArticlesFormData{
				Title:  title,
				Body:   body,
				URL:    updateURL,
				Errors: errors,
			}
			tmpl, err := template.ParseFiles("resources/views/articles/edit.gohtml")
			checkError(err)

			err = tmpl.Execute(w, data)
			checkError(err)
		}
	}
}

func articlesIndexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Access article list.")
}

// ArticlesFormData create blog post form data.
type ArticlesFormData struct {
	Title, Body string
	URL         *url.URL
	Errors      map[string]string
}

func validateArticleFormData(title string, body string) map[string]string {

	errors := make(map[string]string)

	if title == "" {
		errors["title"] = "The title can not be blank"
	} else if utf8.RuneCountInString(title) < 3 || utf8.RuneCountInString(title) > 40 {
		errors["title"] = "Title lenght needs to be between 3-40"
	}

	// validate content
	if body == "" {
		errors["body"] = "The content can not be blank"
	} else if utf8.RuneCountInString(body) < 10 {
		errors["body"] = "Content lenght needs to be greater than or equal to 10 bytes"
	}

	return errors
}

func articlesStoreHandler(w http.ResponseWriter, r *http.Request) {

	title := r.PostFormValue("title")
	body := r.PostFormValue("body")

	errors := validateArticleFormData(title, body)

	// check for errors
	if len(errors) == 0 {
		lastInsertID, err := saveArticleToDB(title, body)
		if lastInsertID > 0 {
			fmt.Fprint(w, "insertion successful, ID is "+strconv.FormatInt(lastInsertID, 10))
		} else {
			checkError(err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "500 server internal error")
		}
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

func saveArticleToDB(title string, body string) (int64, error) {

	// Variable initialization
	var (
		id   int64
		err  error
		rs   sql.Result
		stmt *sql.Stmt
	)

	// 1.get a prepare statement
	stmt, err = db.Prepare("INSERT INTO articles (title, body) VALUES(?,?)")
	// routine error checking
	if err != nil {
		return 0, err
	}

	// 2. close this statement after this function is finished
	// running to prevent it from  occupying the SQL connection
	defer stmt.Close()

	// 3. excute the request and pass parameters into the bound content
	rs, err = stmt.Exec(title, body)
	if err != nil {
		return 0, err
	}

	// 4. if the insertion is successful, the auto-increment ID will be returned
	if id, err = rs.LastInsertId(); id > 0 {
		return id, nil
	}

	return 0, err
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

func createTables() {
	createArticlesSQL := `CREATE TABLE IF NOT EXISTS articles(
    id bigint(20) PRIMARY KEY AUTO_INCREMENT NOT NULL,
    title varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
    body longtext COLLATE utf8mb4_unicode_ci
); `

	_, err := db.Exec(createArticlesSQL)
	checkError(err)
}

func main() {
	initDB()
	createTables()

	router.HandleFunc("/", homeHandler).Methods("GET").Name("home")
	router.HandleFunc("/about", aboutHandler).Methods("GET").Name("about")

	router.HandleFunc("/articles/{id:[0-9]+}", articlesShowHandler).Methods("GET").Name("articles.show")
	router.HandleFunc("/articles", articlesIndexHandler).Methods("GET").Name("articles.index")
	router.HandleFunc("/articles", articlesStoreHandler).Methods("POST").Name("articles.store")
	router.HandleFunc("/articles/create", articlesCreateHandler).Methods("GET").Name("articles.create")
	router.HandleFunc("/articles/{id:[0-9]+}/edit", articlesEditHandler).Methods("GET").Name("articles.edit")
	router.HandleFunc("/articles/{id:[0-9]+}", articlesUpdateHandler).Methods("POST").Name("articles.update")

	// custom 404 page
	router.NotFoundHandler = http.HandlerFunc(notFoundHandler)

	// middleware: force content type to HTML
	router.Use(forceHTMLMiddleware)

	http.ListenAndServe(":3000", removeTrailingSlash(router))
}
