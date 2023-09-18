package main

import (
	"fmt"
	"net/http"
	"sync/atomic"

	uuid "github.com/twinj/uuid"

	"github.com/livefir/fir"
	"github.com/timshannon/bolthold"
)

func index() fir.RouteOptions {
	var count int32
	return fir.RouteOptions{
		fir.ID("counter"),
		fir.Content("counter.html"),

		fir.OnLoad(func(ctx fir.RouteContext) error {
			return ctx.KV("count", atomic.LoadInt32(&count))
		}),
		fir.OnEvent("inc", func(ctx fir.RouteContext) error {
			return ctx.KV("count", atomic.AddInt32(&count, 1))
		}),

		fir.OnEvent("dec", func(ctx fir.RouteContext) error {
			return ctx.KV("count", atomic.AddInt32(&count, -1))
		}),
	}

}

type TodoItem struct {
	Id     string `boltholdKey:"Id"`
	Text   string `json:"todo"`
	Status string
}

type deleteParams struct {
	TodoID []string `json:"todoID"`
}

func todo(db *bolthold.Store) fir.RouteFunc {
	return func() fir.RouteOptions {
		return fir.RouteOptions{
			fir.ID("todo"),
			fir.Content("todo.html"),
			fir.OnEvent("add-todo", func(ctx fir.RouteContext) error {
				todoItem := new(TodoItem)
				if err := ctx.Bind(todoItem); err != nil {
					return err
				}
				todoItem.Status = "not-complete"
				todoItem.Id = uuid.NewV4().String()
				if err := db.Insert(todoItem.Id, todoItem); err != nil {
					return err
				}
				return ctx.Data(todoItem)
			}),
			fir.OnEvent("delete-todo", func(ctx fir.RouteContext) error {
				req := new(deleteParams)
				if err := ctx.Bind(req); err != nil {
					return err
				}
				if err := db.Delete(req.TodoID[0], &TodoItem{}); err != nil {
					fmt.Println(err)
					return err
				}
				return nil
			}),
			fir.OnEvent("mark-complete", func(ctx fir.RouteContext) error {
				req := new(deleteParams)
				if err := ctx.Bind(req); err != nil {
					return err
				}
				var todoItem TodoItem
				if err := db.Get(req.TodoID[0], &todoItem); err != nil {
					return err
				}
				todoItem.Status = "completed"
				if err := db.Update(req.TodoID[0], &todoItem); err != nil {
					return err
				}
				return ctx.Data(todoItem)
			}),
			fir.OnLoad(func(ctx fir.RouteContext) error {
				var todos []TodoItem
				if err := db.Find(&todos, &bolthold.Query{}); err != nil {
					return err
				}
				return ctx.Data(map[string]any{"todos": todos})
			}),
		}
	}
}

func main() {
	db, err := bolthold.Open("todos1.db", 0666, nil)
	if err != nil {
		panic(err)
	}
	controller := fir.NewController("fir_app", fir.DevelopmentMode(true))
	http.Handle("/counter", controller.RouteFunc(index))
	http.Handle("/", controller.RouteFunc(todo(db)))
	http.ListenAndServe(":9867", nil)
}
