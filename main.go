package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
)

type user struct {
	UserName string
	Password []byte
	First    string
	Last     string
	Role     string
}

type session struct {
	un           string
	lastActivity time.Time
}

type product struct {
	Product_id          string
	Product_name        string
	Image_path          string
	Product_price       int
	Available           int
	Product_description string
}

var tpl *template.Template

//var dbUsers = map[string]user{}       // user ID, user
var dbSessions = map[string]session{} // session ID, session
var dbSessionsCleaned time.Time
var db *sql.DB

const sessionLength int = 30

func init() {
	tpl = template.Must(template.ParseGlob("templates/*"))
	dbSessionsCleaned = time.Now()
}

func main() {
	db, _ = sql.Open("mysql", "root:avabbittu@/sys?charset=utf8")
	http.HandleFunc("/", index)
	//http.HandleFunc("/bar", bar)
	http.HandleFunc("/signup", signup)
	http.HandleFunc("/login", login)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/loggedin", loggedIn)
	http.Handle("/favicon.ico", http.NotFoundHandler())
	http.Handle("/resources/", http.StripPrefix("/resources", http.FileServer(http.Dir("./Assets"))))
	http.ListenAndServe(":80", nil)
}

func display(pro []product) {
	for _, prod := range pro {
		fmt.Println(prod)
	}
	fmt.Println("End")
}

func HandleDb(db *sql.DB) []product {
	rows, err := db.Query("SELECT * FROM product")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	var products []product
	for rows.Next() {
		var prod product
		if err := rows.Scan(&prod.Product_id, &prod.Product_name, &prod.Image_path, &prod.Product_price, &prod.Available, &prod.Product_description); err != nil {
			log.Fatal(err)
		}
		products = append(products, prod)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	return products
}

func loggedIn(w http.ResponseWriter, req *http.Request) {
	if !alreadyLoggedIn(w, req) {
		//http.Error(w, "You must login to view this page", http.StatusForbidden)
		http.Redirect(w, req, "/login", http.StatusSeeOther)
		return
	}
	pro := HandleDb(db)
	display(pro)
	err := tpl.ExecuteTemplate(w, "loggedInIndex.gohtml", pro)
	fmt.Println(err)
}

func index(w http.ResponseWriter, req *http.Request) {
	//u := getUser(w, req)
	//showSessions() // for demonstration purposes
	pro := HandleDb(db)
	display(pro)
	search := req.FormValue("search")
	err := tpl.ExecuteTemplate(w, "index.gohtml", pro)
	fmt.Println(err)
	fmt.Println(search)
}

/*
func bar(w http.ResponseWriter, req *http.Request) {
	u := getUser(w, req)
	if !alreadyLoggedIn(w, req) {
		http.Redirect(w, req, "/", http.StatusSeeOther)
		return
	}
	if u.Role != "007" {
		http.Error(w, "You must be 007 to enter the bar", http.StatusForbidden)
		return
	}
	showSessions() // for demonstration purposes
	tpl.ExecuteTemplate(w, "bar.gohtml", u)
}
*/
func signup(w http.ResponseWriter, req *http.Request) {
	if alreadyLoggedIn(w, req) {
		http.Redirect(w, req, "/", http.StatusSeeOther)
		return
	}
	var u user
	// process form submission
	if req.Method == http.MethodPost {
		// get form values
		un := req.FormValue("username")
		p := req.FormValue("password")
		f := req.FormValue("firstname")
		l := req.FormValue("lastname")
		r := req.FormValue("role")
		// username taken?
		if checkFormDb(un) {
			http.Error(w, "Username already taken", http.StatusForbidden)
			return
		}
		// create session
		sID := uuid.NewV4()
		c := &http.Cookie{
			Name:  "session",
			Value: sID.String(),
		}
		c.MaxAge = sessionLength
		http.SetCookie(w, c)
		dbSessions[c.Value] = session{un, time.Now()}
		// store user in dbUsers
		bs, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.MinCost)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		u = user{un, bs, f, l, r}
		res, err := db.Exec("INSERT INTO user VALUES (?,?,?,?,?);", un, bs, f, l, 0)
		fmt.Println(res)
		fmt.Println(err)
		//dbUsers[un] = u
		// redirect
		http.Redirect(w, req, "/loggedin", http.StatusSeeOther)
		return
	}
	showSessions() // for demonstration purposes
	tpl.ExecuteTemplate(w, "signup.gohtml", u)
}

func login(w http.ResponseWriter, req *http.Request) {
	/*if alreadyLoggedIn(w, req) {
		http.Redirect(w, req, "/", http.StatusSeeOther)
		return
	}*/
	var u user
	// process form submission
	if req.Method == http.MethodPost {
		un := req.FormValue("username")
		p := req.FormValue("password")
		// is there a username?
		//u, ok := dbUsers[un]
		row, err := db.Query("Select Username, Password from user")
		if err != nil {
			log.Fatal(err)
		}
		defer row.Close()
		var userna, pass string
		flag := 0
		for row.Next() {
			if err := row.Scan(&userna, &pass); err != nil {
				log.Fatal(err)
			}
			if userna == un {
				err := bcrypt.CompareHashAndPassword([]byte(pass), []byte(p))
				if err != nil {
					http.Error(w, "Username and/or password do not match", http.StatusForbidden)
					return
				}
				flag = 1
			}
		}
		if flag == 0 {
			http.Error(w, "Username does not exist", http.StatusForbidden)
			return
		}
		sID := uuid.NewV4()
		c := &http.Cookie{
			Name:  "session",
			Value: sID.String(),
		}
		c.MaxAge = sessionLength
		http.SetCookie(w, c)
		dbSessions[c.Value] = session{un, time.Now()}
		http.Redirect(w, req, "/loggedin", http.StatusSeeOther)
		return
	}
	showSessions() // for demonstration purposes
	tpl.ExecuteTemplate(w, "login.gohtml", u)
}

func logout(w http.ResponseWriter, req *http.Request) {
	if !alreadyLoggedIn(w, req) {
		http.Redirect(w, req, "/", http.StatusSeeOther)
		return
	}
	c, _ := req.Cookie("session")
	// delete the session
	delete(dbSessions, c.Value)
	// remove the cookie
	c = &http.Cookie{
		Name:   "session",
		Value:  "",
		MaxAge: -1,
	}
	http.SetCookie(w, c)

	// clean up dbSessions
	if time.Since(dbSessionsCleaned) > (time.Second * 30) {
		go cleanSessions()
	}

	http.Redirect(w, req, "/login", http.StatusSeeOther)
}

func alreadyLoggedIn(w http.ResponseWriter, req *http.Request) bool {
	c, err := req.Cookie("session")
	if err != nil {
		return false
	}
	s, ok := dbSessions[c.Value]
	if ok {
		s.lastActivity = time.Now()
		dbSessions[c.Value] = s
	}

	ok = checkFormDb(s.un)
	//_, ok = dbUsers[s.un]
	// refresh session
	c.MaxAge = sessionLength
	http.SetCookie(w, c)
	return ok
}

func checkFormDb(un string) bool {
	rows, err := db.Query("select username from user")
	if err != nil {
		return false
	}
	defer rows.Close()
	var username string
	for rows.Next() {
		if err := rows.Scan(&username); err != nil {
			log.Fatal(err)
			return false
		}
		if username == un {
			return true
		}
	}
	return false
}

func cleanSessions() {
	fmt.Println("BEFORE CLEAN")
	showSessions()
	for k, v := range dbSessions {
		if time.Since(v.lastActivity) > (time.Second * 30) {
			delete(dbSessions, k)
		}
	}
	dbSessionsCleaned = time.Now()
	fmt.Println("AFTER CLEAN")
	showSessions()
}

func showSessions() {
	fmt.Println("********")
	for k, v := range dbSessions {
		fmt.Println(k, v.un)
	}
	fmt.Println("")
}
