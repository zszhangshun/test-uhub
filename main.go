package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	api "test/handler"
	"test/pkg/config"
	"test/pkg/uniqinfo"

	"test/pkg/server"
	"test/pkg/store"
)

var ConfigPath *string

func init() {
	ConfigPath = flag.String("c", "config/config.json", "config file path")
	flag.Parse()
}
func main() {
	cfg := config.NewConfig()
	cfgPath := *ConfigPath
	err := cfg.Parse(cfgPath)
	if err != nil {
		log.Fatal("parse config file failed:", err)
	}
	db, err := store.New(cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.Database)
	if err != nil {
		log.Fatal("init db file failed:", err)
	}

	UhubUniqInfo := uniqinfo.NewUniqInfos()
	handle := &api.Handle{
		Config:       cfg,
		Store:        store.NewClient(db, cfg.DB.Table),
		UhubUniqInfo: UhubUniqInfo,
	}
	go handle.Store.Flush(handle.UhubUniqInfo)
	s := server.NewServer(handle)
	ser := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.ServerPort),
		Handler: s.Engine,
	}
	serverErr := make(chan error, 1)
	go func() {
		if err := ser.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
		close(serverErr)
	}()

	// 主程序等待服务器错误或关闭信号
	select {
	case err := <-serverErr:
		if err != nil {
			fmt.Printf("Server encountered an error: %v\n", err)
		}
	}

}
