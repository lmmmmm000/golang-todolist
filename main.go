package main

import (
	"context"
	"encoding/json"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/thedevsaddam/renderer"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
)

var rnd *renderer.Render
var db *mgo.Database

const (
	hostName       string = "localhost:27017"
	dbName         string = "demo_todo"
	collectionName string = "todo"
	port           string = ":8080"
)

	// todoModel是一个MongoDB文档的映射，包含了一个待办事项的所有属性
type (
	todoModel struct {
		ID        bson.ObjectId `bson:"_id,omitempty"`
		Title     string        `bson:"title"`
		Completed bool          `bson:"completed"`
		CreatedAt time.Time     `bson:"createdAt"`
	}
	// todo是一个用于返回给客户端的待办事项的简化结构体
	todo struct {
		ID        string    `json:"id"`
		Title     string    `json:"title"`
		Completed bool `json:"completed"`
		CreatedAt time.Time `json:"created_at"`
	}
)

//使用了 mgo 库来连接 MongoDB 数据库，
//并设置了 Monotonic 模式，以确保读取操作总是从主节点进行
//Monotonic 模式是 mgo 库中的一种读取模式，它确保读取操作总是从主节点进行。
//在 MongoDB 中，主节点是负责处理所有写入操作的节点。
//当我们使用 Monotonic 模式时，mgo 库会尝试从主节点读取数据，但如果主节点不可用，则会从从节点读取数据。
//这种模式可以提高读取操作的可用性和性能，因为它可以避免从从节点读取过期的数据。
func init() {
	rnd = renderer.New()
	sess, err := mgo.Dial(hostName)
	checkErr(err)
	sess.SetMode(mgo.Monotonic, true)
	db = sess.DB(dbName)
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	// 创建一个停止信号通道
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)
	// 创建新路由
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", homeHandler)
	r.Mount("/todo", todoHandlers())

	// 创建一个HTTP服务器
	srv := &http.Server{
		Addr:         port,
		Handler:      r,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	// 启动HTTP服务器
	go func() {
		log.Println("Listening on port", port)
		if err := srv.ListenAndServe();err != nil{
			log.Printf("listen:%s\n", err)
		}
	}()
	// 等待停止信号
	<- stopChan
	log.Printf("shutting down server..." )
	ctx, cancel:= context.WithTimeout(context.Background(), 5*time.Second)
	srv.Shutdown(ctx)
	defer cancel()
	log.Println("server gracefully stopped")
}

func todoHandlers() http.Handler {
	rg := chi.NewRouter()
	rg.Group(func(r chi.Router) {
		r.Get("/", fetchTodos)
		r.Post("/", createTodo)
		r.Put("/{id}", updateTodo)
		r.Delete("/{id}", deleteTodo)
	})
	return rg
}

func homeHandler(w http.ResponseWriter, r *http.Request){
	// 渲染主页模板
	err := rnd.Template(w, http.StatusOK, []string{"static/home.tpl"}, nil)
	checkErr(err)

}

func fetchTodos(w http.ResponseWriter, r *http.Request){
	// todos，用于存储从数据库中检索到的所有待办事项
	todos := []todoModel{}
	// 从数据库中获取所有待办事项，bson.M{}是一个空的MongoDB查询，它返回所有文档。
	if err := db.C(collectionName).Find(bson.M{}).All(&todos);err != nil{
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message": "Failed to fetch todo",
			"error":err,
		})
		return
	}
	// todoList，用于存储将要返回给客户端的待办事项
	todoList := []todo{}

	for _, t := range todos{
		todoList = append(todoList, todo{
			ID: t.ID.Hex(),
			Title:  t.Title,
			Completed: t.Completed,
			CreatedAt:  t.CreatedAt,
		})
	}
	rnd.JSON(w, http.StatusOK, renderer.M{
		"data": todoList,
	})
}

func createTodo(w http.ResponseWriter, r *http.Request)  {
	var t todo
	// 首先使用 json.NewDecoder() 函数从请求体中解码 JSON 数据，并将其转换为todo结构体
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil{
		rnd.JSON(w, http.StatusProcessing, err)
		return
	}
	// 检查todo 结构体中的 Title 字段是否为空。
	//如果为空，则返回一个 HTTP 状态码 400 和一个包含错误消息的 JSON 响应。
	if t.Title == ""{
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "the title is required",
		})
		return
	}
	// 创建一个新的 todoModel 结构体，
	//将todo 结构体中的字段复制到其中，并为 ID 字段生成一个新的 bson.ObjectId
	tm := todoModel{
		ID: bson.NewObjectId(),
		Title: t.Title,
		Completed: false,
		CreatedAt:  time.Now(),
	}
	// 将 todoModel 结构体插入到 MongoDB 数据库中
	if err := db.C(collectionName).Insert(&tm); err != nil{
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message": "Failed to save todo",
			"error": err,
		})
		return
	}
	// 返回一个包含成功消息和新创建的待办事项 ID 的 JSON 响应
	rnd.JSON(w, http.StatusCreated, renderer.M{
		"message": "todo create successfully",
		"todo_id": tm.ID.Hex(),
	})
}

func deleteTodo(w http.ResponseWriter, r *http.Request){
	// 先从 URL 参数中获取待删除的待办事项的 ID
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	// 检查该 ID 是否为有效的 bson.ObjectId，
	//如果 ID 无效，则返回一个 HTTP 状态码 400 和一个包含错误消息的 JSON 响应
	if !bson.IsObjectIdHex(id){
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "the id is invalid",

		})
	return
	}
	//  ID 有效，则使用 db.C(collectionName).RemoveId() 函数从数据库中删除该待办事项
	// 如果删除操作失败，则返回一个 HTTP 状态码 500 和一个包含错误消息的 JSON 响应
	if err := db.C(collectionName).RemoveId(bson.ObjectIdHex(id)); err != nil{
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message": "Failed to delete todo",
			"error": err,
		})
		return
	}
	// 如果删除成功，则返回一个 HTTP 状态码 200 和一个包含成功消息的 JSON 响应
	rnd.JSON(w, http.StatusOK, renderer.M{
		"message":"todo deleted successfully",
	})
	return
}

func updateTodo(w http.ResponseWriter, r *http.Request){
	id := strings.TrimSpace(chi.URLParam(r,"id"))
	// 检查待办事项的 ID 是否为有效的 bson.ObjectId，
	//如果 ID 无效，则返回一个 HTTP 状态码 400 和一个包含错误消息的 JSON 响应
	if !bson.IsObjectIdHex(id){
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "The id is invalid",

		})
		return
	}
	var t todo
	// 从请求体中解码 JSON 数据，并将其转换为todo 结构体
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil{
		rnd.JSON(w, http.StatusProcessing, err)
		return
	}
	// 检查待办事项的标题是否为空，
	//如果为空，则返回一个 HTTP 状态码 400 和一个包含错误消息的 JSON 响应
	if t.Title==""{
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "the title field id required",
		})
		return
	}
	// 使用 db.C(collectionName).Update() 函数更新数据库中的待办事项
	if err := db.C(collectionName).
		Update(bson.M{"_id": bson.ObjectIdHex(id)},
			bson.M{"title": t.Title, "completed": t.Completed});
	err != nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message": "failed to update todo",
			"error": err,
		})
		return
	}
}