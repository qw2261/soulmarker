package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = defaultDB
	}

	store := NewStore(dbPath)
	log.Printf("📦 数据库已初始化: %s", dbPath)

	h := NewHandler(store)
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/events", h.CreateEvent)
	mux.HandleFunc("GET /api/events", h.ListEvents)
	mux.HandleFunc("GET /api/events/{id}", h.GetEvent)
	mux.HandleFunc("PUT /api/events/{id}", h.UpdateEvent)
	mux.HandleFunc("DELETE /api/events/{id}", h.DeleteEvent)
	mux.HandleFunc("POST /api/events/{id}/register", h.Register)
	mux.HandleFunc("GET /api/events/{id}/registrations", h.ListRegistrations)

	addr := ":" + port
	log.Printf("亦闻 event-go 服务启动，监听端口 %s", port)
	log.Printf("API 文档：")
	log.Printf("  POST   /api/events                      创建活动")
	log.Printf("  GET    /api/events                      活动列表")
	log.Printf("  GET    /api/events/{id}                 活动详情")
	log.Printf("  PUT    /api/events/{id}                 编辑活动")
	log.Printf("  DELETE /api/events/{id}                 删除活动")
	log.Printf("  POST   /api/events/{id}/register        报名活动")
	log.Printf("  GET    /api/events/{id}/registrations   报名列表")

	if err := http.ListenAndServe(addr, cors(mux)); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
