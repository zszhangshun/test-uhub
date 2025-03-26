package uniqinfo

// import (
// 	"fmt"
// 	"log"
// 	"test/pkg/config"
// 	"testing"
// 	"time"
// )

// func TestGetkUniqInfo(t *testing.T) {
// 	cfg := config.NewConfig()
// 	err := cfg.Parse("/Users/user/test/test-xx/config/config.json")
// 	if err != nil {
// 		log.Fatal("parse config file failed:", err)
// 	}
// 	cfg.InitDB()
// 	uns := NewUniqInfos()
// 	uns.Db = config.GetDB()
// 	flash := make(chan bool, 1)
// 	flash <- true
// 	duration := time.NewTicker(2 * time.Minute)
// 	uns.FlashUhubUniqInfo(flash, duration)
// 	for _, v := range uns.Info {
// 		fmt.Println("xxx:", v.UniqCloudChannelID)
// 	}

// }