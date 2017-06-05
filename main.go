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
	Id            int64
	Book_name     string
	Isbn          string
	Rent_time     int64
	Complete_time int64
	Status        bool
}

type Book_entry struct {
	Id    int64
	Name  string
	Isbn  string
	Price float32
	Count int
	Left  int
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
			cookie := http.Cookie{Name: "uid", Value: strconv.FormatInt(uid, 10), Path: "/", MaxAge: 3600}
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
		if sql.ErrNoRows != err {
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

func handl_logout(w http.ResponseWriter, r *http.Request) {
	_, err := get_cur_user_id(r)
	if err != nil {
		fmt.Println("not logged into the sys")
		http.Redirect(w, r, "/", 302)
		return
	}

	// expires cookie
	cookie := http.Cookie{Name: "uid", Path: "/", MaxAge: -1}
	http.SetCookie(w, &cookie)

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

	rows, err := db.Query("SELECT t_rent.id, t_book.book_name, t_book.isbn, status, rent_time, complete_time FROM t_rent join t_book on t_book.id= t_rent.book_id where user_id=?", id)
	checkErr(err)
	db.Close()

	data := []Rent_entry{}
	tRes := Rent_entry{}

	for rows.Next() {
		var rid int64
		var name, isbn string
		var status bool
		var rent_time, complete_time int64
		rows.Scan(&rid, &name, &isbn, &status, &rent_time, &complete_time)
		tRes.Id = rid
		tRes.Book_name = name
		tRes.Isbn = isbn
		tRes.Rent_time = rent_time
		tRes.Complete_time = complete_time
		tRes.Status = status
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
	defer db.Close()
	checkErr(err)

	r.ParseForm()
	keyword := r.FormValue("keyword")
	fmt.Println("keyword:", keyword)
	isbn := keyword
	keyword = "%" + keyword + "%"
	fmt.Println("keyword changed: ", keyword)
	rows, err := db.Query("SELECT id, book_name, isbn, price, count, left_count FROM t_book where book_name like ? or isbn=?", keyword, isbn)
	checkErr(err)

	data := []Book_entry{}
	tRes := Book_entry{}

	for rows.Next() {
		var id int64
		var name, isbn string
		var count, left int
		var price float32
		rows.Scan(&id, &name, &isbn, &price, &count, &left)
		tRes.Id = id
		tRes.Name = name
		tRes.Isbn = isbn
		tRes.Count = count
		tRes.Price = price
		tRes.Left = left
		fmt.Printf("%d %s %s %f %d %d\n", id, name, isbn, price, count, left)
		data = append(data, tRes)
	}
	t, _ := template.ParseFiles("pages/result.html")
	log.Println(t.Execute(w, data))
}

func handle_manage(w http.ResponseWriter, r *http.Request) {
	id, err := get_cur_user_id(r)
	if err != nil {
		fmt.Println("not logged into the sys")
		failedPage(w)
		return
	}

	// get user
	db, err := sql.Open("mysql", "root:@tcp(localhost:3306)/book_rent?charset=utf8")
	checkErr(err)
	defer db.Close()

	//check user exist
	var type_id int
	err = db.QueryRow("SELECT type_id FROM t_user where id=?", id).Scan(&type_id)
	checkErr(err)

	if type_id != 2 {
		failedPage(w)
		return
	}

	t, _ := template.ParseFiles("pages/manage.html")
	log.Println(t.Execute(w, nil))

}

func handle_add_book(w http.ResponseWriter, r *http.Request) {
	fmt.Println("method:", r.Method)
	if r.Method == "POST" {
		r.ParseForm()

		price := r.FormValue("price")
		count := r.FormValue("count")
		isbn := r.FormValue("isbn")
		book_name := r.FormValue("book_name")

		db, err := sql.Open("mysql", "root:@tcp(localhost:3306)/book_rent?charset=utf8")
		checkErr(err)
		defer db.Close()

		//check user exist
		var is_exist int
		err = db.QueryRow("SELECT 1 FROM t_book where isbn=?", isbn).Scan(&is_exist)
		//checkErr(err)
		if sql.ErrNoRows != err {
			t, _ := template.ParseFiles("pages/manage.html")
			log.Println(t.Execute(w, nil))
			return
		}

		//插入数据
		stmt, err := db.Prepare("INSERT t_book SET book_name=?,isbn=?,price=?,count=?,left_count=?")
		checkErr(err)

		res, err := stmt.Exec(book_name, isbn, price, count, count)
		checkErr(err)

		affected, err := res.RowsAffected()
		checkErr(err)

		fmt.Println(affected)
		db.Close()
		if affected == 1 {
			t, _ := template.ParseFiles("pages/add_success.html")
			log.Println(t.Execute(w, nil))
		}

	} else if r.Method == "GET" {
		t, _ := template.ParseFiles("pages/manage.html")
		log.Println(t.Execute(w, nil))
	}
}

func handle_return_book(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {

		rent_id := r.FormValue("rent_id")

		db, err := sql.Open("mysql", "root:@tcp(localhost:3306)/book_rent?charset=utf8")
		checkErr(err)
		defer db.Close()

		//check user exist
		var status bool
		err = db.QueryRow("SELECT status FROM t_rent where id=?", rent_id).Scan(&status)
		//checkErr(err)
		if sql.ErrNoRows == err || status == true {
			fmt.Fprint(w, "Wrong borrow id or the book has been returned\n")
			return
		}

		stmt, err := db.Prepare("update t_rent set complete_time=?,status=? where id=?")
		checkErr(err)

		complete_time := time.Now().Unix()
		res, err := stmt.Exec(complete_time, 1, rent_id)
		checkErr(err)

		affect, err := res.RowsAffected()
		checkErr(err)

		if affect != 1 {
			fmt.Fprint(w, "error updating database")
			return
		}

		stmt, err = db.Prepare("update t_book set left_count=left_count+1 where t_book.id=(select book_id from t_rent where t_rent.id=?)")
		checkErr(err)

		res, err = stmt.Exec(rent_id)
		checkErr(err)

		affect, err = res.RowsAffected()
		checkErr(err)

		if affect != 1 {
			fmt.Fprint(w, "error updating database")
			return
		}

		fmt.Fprint(w, "Success!")

	}
}

func handle_borrow_book(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		book_id := r.FormValue("book_id")
		user_id, err := get_cur_user_id(r)
		checkErr(err)

		var left_count int

		db, err := sql.Open("mysql", "root:@tcp(localhost:3306)/book_rent?charset=utf8")
		checkErr(err)
		defer db.Close()

		//check user exist
		err = db.QueryRow("SELECT left_count FROM t_book where id=?", book_id).Scan(&left_count)
		//checkErr(err)
		if sql.ErrNoRows == err || left_count <= 0 {
			fmt.Fprint(w, "Wrong book id or the book is not availible\n")
			return
		}

		stmt, err := db.Prepare("INSERT t_rent SET book_id=?,user_id=?,rent_time=?,status=?")
		checkErr(err)

		rent_time := time.Now().Unix()
		res, err := stmt.Exec(book_id, user_id, rent_time, 0)
		checkErr(err)

		affect, err := res.RowsAffected()
		checkErr(err)

		if affect != 1 {
			fmt.Fprint(w, "error updating database")
			return
		}

		stmt, err = db.Prepare("update t_book set left_count=left_count-1 where t_book.id=?")
		checkErr(err)

		res, err = stmt.Exec(book_id)
		checkErr(err)

		affect, err = res.RowsAffected()
		checkErr(err)

		if affect != 1 {
			fmt.Fprint(w, "error updating database")
			return
		}

		fmt.Fprint(w, "Success!")
	}
}

func main() {
	http.HandleFunc("/", index_page)
	http.HandleFunc("/register", handle_register)
	http.HandleFunc("/login", handle_login)
	http.HandleFunc("/my", handle_my)
	http.HandleFunc("/search", handle_search)
	http.HandleFunc("/manage", handle_manage)
	http.HandleFunc("/add_book", handle_add_book)

	http.HandleFunc("/return", handle_return_book)
	http.HandleFunc("/borrow", handle_borrow_book)

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
	if cookie_err != nil {
		return 0, cookie_err
	}
	id, _ := strconv.ParseInt(cookie.Value, 10, 64)
	return id, cookie_err
}

func failedPage(w http.ResponseWriter) {
	t, _ := template.ParseFiles("pages/failed.html")
	log.Println(t.Execute(w, nil))
}
