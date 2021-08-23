//1 вывод лога после, запуска сервера
//2 желательно разделить код по пакетам, service, handler, models, repository
//3 лучше создать его внутри функции и там же использовать, для каких то целей.
//4 * ссылка на метод структуры данные из структуры, метод его же
//5 waitgroup - для того что бы main gorutine дождалась, пока выполнится, backgroundTask
//6 не правильный запрос в базу, идет конкатенация, это опасно для sql injection

// т.к не работал  gorilla/mux router  и с postgres/pkg пакетами, могу что-то не заметить

package main

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/mux"

	"github.com/jackc/pgx/v4"
)

type BookModel struct {
	Title  string
	Author string
	Cost   int
}

type Service struct {
	Pool   []*pgx.Conn
	IsInit bool
}

var wg sync.WaitGroup //5

func (s *Service) initService(username, password string) { //4

	wg.Add(1)

	var backgroundTask = func() {
		var databaseUrl = "postgres://" + username + ":" + password + "@localhost:5432/books"
		for i := 1; i <= 10; i++ { // add dbConn in []Pool Db
			conn, err := pgx.Connect(context.Background(), databaseUrl)
			if err != nil {
				println("Ошибка при подключении к базе по URL = " + databaseUrl)
				panic(nil)
			}
			s.Pool = append(s.Pool, conn)
		}
		wg.Done()
	}
	go backgroundTask()
}

func (s *Service) getBooksByAuthor(username, password string, author *string) { //4
	var result = make([]BookModel, 10) //3

	if !s.IsInit { //singleton
		s.initService(username, password)
		s.IsInit = true
		wg.Wait()
	}
	//pool db conn -> for each client use this conn, if free, else wait poka will be free
	var conn *pgx.Conn
	for _, x := range s.Pool {
		if !x.IsClosed() {
			conn = x
			break
		}
	}
	//get data from db
	rows, err := conn.Query(context.Background(), "select title, cost from books where author=$1", *author) //6
	if err != nil {
		println("Не удалось получить книги по автору")
		panic(nil)
	}
	//write data
	for rows.Next() {
		var title string
		var cost int
		err = rows.Scan(&title, &cost)
		if err == nil {
			result = append(result, BookModel{title, *author, cost})
		}
	}
	log.Println(result, "result")

	println("Успешно выполнен запрос, заполнено записей: " + strconv.Itoa(len(result)))
}

func main() {
	var service = Service{}
	//2
	r := mux.NewRouter()
	r.HandleFunc("/GetBookByAuthor/{author}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		author := vars["author"]
		service.getBooksByAuthor("postgres", "password", &author) //2
	})
	http.ListenAndServe(":8083", r)
	println("Запуск сервера...") //1
}
