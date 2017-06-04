package main

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"
	//"strings"
)

type Rent_entry struct {
	id            int64
	name          string
	isbn          string
	rent_time     int64
	complete_time int64
	status        bool
}

type Book_entry struct {
	id    int64
	name  string
	isbn  string
	price float32
	count int
	left  int
}

func index_page(w http.ResponseWriter, r *http.Request) {
	fmt.Println("method:", r.Method) //获取请求的方法

	if r.Method == "GET" {
		t, _ := template.ParseFiles("pages/index.html")
		log.Println(t.Execute(w, nil))
	}
}

// func register_page(w http.ResponseWriter, r *http.Request) {
// 	fmt.Println("method:", r.Method) //获取请求的方法

// }

func handle_login(w http.ResponseWriter, r *http.Request) {
	fmt.Println("method:", r.Method)
	if r.Method == "POST" {
		r.ParseForm()
		username := r.FormValue("username")
		password := r.FormValue("password")

		db, err := sql.Open("mysql", "root:@tcp(localhost:3306)/book_rent?charset=utf8")
		checkErr(err)

		var queried_password string
		var queried_salt string
		var uid int64
		err = db.QueryRow("SELECT id, password, salt FROM t_user where username=? or stu_no=?", username, username).Scan(&uid, &queried_password, &queried_salt)
		checkErr(err)

		db.Close()

		if md5_hash(password+queried_salt) == queried_password {
			// t, _ := template.ParseFiles("pages/success.html")
			// log.Println(t.Execute(w, nil))

			//setting cookie
			fmt.Println("setting cookie")
			expiration := time.Now()
			expiration = expiration.AddDate(1, 0, 0)
			cookie := http.Cookie{Name: "uid", Value: strconv.FormatInt(uid, 10), Expires: expiration}
			http.SetCookie(w, &cookie)

			http.Redirect(w, r, "/my", 302)

		} else {
			t, _ := template.ParseFiles("pages/failed.html")
			log.Println(t.Execute(w, nil))
		}

	} else if r.Method == "GET" {
		t, _ := template.ParseFiles("pages/index.html")
		log.Println(t.Execute(w, nil))
	}
}

func handle_register(w http.ResponseWriter, r *http.Request) {
	fmt.Println("method:", r.Method)
	if r.Method == "POST" {
		r.ParseForm()

		stu_no := r.FormValue("stu_no")
		username := r.FormValue("username")

		db, err := sql.Open("mysql", "root:@tcp(localhost:3306)/book_rent?charset=utf8")
		checkErr(err)

		//check user exist
		var is_exist int
		err = db.QueryRow("SELECT 1 FROM t_user where stu_no=? or username=?", stu_no, username).Scan(&is_exist)
		checkErr(err)
		if is_exist == 1 {
			t, _ := template.ParseFiles("pages/failed.html")
			log.Println(t.Execute(w, nil))
			return
		}

		//插入数据
		stmt, err := db.Prepare("INSERT t_user SET name=?,username=?,password=?,stu_no=?,type_id=?,salt=?")
		checkErr(err)

		timestamp := time.Now().Unix()
		salt := md5_hash(strconv.FormatInt(timestamp, 10))
		password := md5_hash(r.FormValue("password") + salt)
		fmt.Println("salt:", salt)
		fmt.Println("password:", password)
		res, err := stmt.Exec(r.FormValue("name"), username, password, stu_no, 1, salt)
		checkErr(err)

		affected, err := res.RowsAffected()
		checkErr(err)

		fmt.Println(affected)
		db.Close()
		if affected == 1 {
			t, _ := template.ParseFiles("pages/success.html")
			log.Println(t.Execute(w, nil))
		}

	} else if r.Method == "GET" {
		t, _ := template.ParseFiles("pages/register.html")
		log.Println(t.Execute(w, nil))
	}
}

func handle_my(w http.ResponseWriter, r *http.Request) {
	id, err := get_cur_user_id(r)
	if err != nil {
		fmt.Println("not logged into the sys")
		failedPage(w)
		return
	}

	fmt.Println("user id:", id)

	db, err := sql.Open("mysql", "root:@tcp(localhost:3306)/book_rent?charset=utf8")
	checkErr(err)

	rows, err := db.Query("SELECT t_rent.id, t_book.name, t_book.isbn, status, rent_time, complete_time FROM t_rent join t_book on t_book.id= t_rent.book_id where user_id=?", id)
	checkErr(err)

	data := []Rent_entry{}
	tRes := Rent_entry{}

	for rows.Next() {
		var rid int64
		var name, isbn string
		var status bool
		var rent_time, complete_time int64
		rows.Scan(&rid, &name, &isbn, &status, &rent_time, &complete_time)
		tRes.id = rid
		tRes.name = name
		tRes.isbn = isbn
		tRes.rent_time = rent_time
		tRes.complete_time = complete_time
		tRes.status = status
		data = append(data, tRes)
	}
	t, _ := template.ParseFiles("pages/my.html")
	log.Println(t.Execute(w, data))
}

func handle_search(w http.ResponseWriter, r *http.Request) {
	id, err := get_cur_user_id(r)
	if err != nil {
		fmt.Println("not logged into the sys")
		failedPage(w)
		return
	}

	fmt.Println("user id:", id)

	db, err := sql.Open("mysql", "root:@tcp(localhost:3306)/book_rent?charset=utf8")
	checkErr(err)

	r.ParseForm()
	keyword := r.FormValue("keyword")
	fmt.Println("keyword:", keyword)
	rows, err := db.Query("SELECT id, name, isbn, price, count FROM t_book where contains(name, ?) or isbn=?", keyword, keyword)
	checkErr(err)

	data := []Book_entry{}
	tRes := Book_entry{}

	for rows.Next() {
		var id int64
		var name, isbn string
		var count int
		var price float32
		rows.Scan(&id, &name, &isbn, &price, &count)
		tRes.id = id
		tRes.name = name
		tRes.isbn = isbn
		tRes.count = count
		tRes.price = price
		data = append(data, tRes)
	}
	t, _ := template.ParseFiles("pages/result.html")
	log.Println(t.Execute(w, data))
}

func main() {
	http.HandleFunc("/", index_page)
	http.HandleFunc("/register", handle_register)
	http.HandleFunc("/login", handle_login)
	http.HandleFunc("/my", handle_my)
	http.HandleFunc("/search", handle_search)

	err := http.ListenAndServe(":9090", nil) //设置监听的端口
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func md5_hash(str string) string {
	data := []byte(str)
	s := fmt.Sprintf("%x", md5.Sum(data))
	return s
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

//return
func get_cur_user_id(r *http.Request) (int64, error) {
	cookie, cookie_err := r.Cookie("uid")
	id, _ := strconv.ParseInt(cookie.Value, 10, 64)
	return id, cookie_err
}

func failedPage(w http.ResponseWriter) {
	t, _ := template.ParseFiles("pages/failed.html")
	log.Println(t.Execute(w, nil))
}
