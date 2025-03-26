package store

// import (
// 	"fmt"
// 	"test/pkg/config"
// 	"test/pkg/uniqinfo"
// 	"testing"
// )
// func TestStoreGet(t *testing.T) {
//     // 使用相对路径或环境变量来提高移植性
//     cfg := config.NewConfig()
// 	err := cfg.Parse("/Users/user/test/test-xx/config/config.json")
//     if err != nil {
//         t.Fatalf("parse config file failed: %v", err)
//     }

//     db, err := New(cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.Database)
//     if err != nil {
//         t.Fatalf("init db failed: %v", err)
//     }
//     client := NewClient(db, cfg.DB.Table)
//     // 定义一个具体的结构体来接收数据
//     var dst uniqinfo.UhubUniqChannelInfo
//     err = client.Get("*",&dst)
//     if err != nil {
//         t.Errorf("Get() error = %v", err)
//         return
//     }

//     t.Logf("Get() result = %+v", dst)

//     // 进行一些基本的验证
//     if dst.UniqCloudChannelID!="" {
//        fmt.Println(dst)
//     }
// }