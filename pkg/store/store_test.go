package store

import (
	"fmt"
	"test/pkg/config"
	"testing"
)

func TestStore(t *testing.T) {
	// 使用相对路径或环境变量来提高移植性
	cfg := config.NewConfig()
	err := cfg.Parse("/Users/user/test/test-xx/config/config.json")
	if err != nil {
		t.Fatalf("parse config file failed: %v", err)
	}

	db, err := New(cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.Database)
	if err != nil {
		t.Fatalf("init db failed: %v", err)
	}
	client := NewClient(db, cfg.DB.Table)
	update := map[string]interface{}{
		"uniq_cloud_domain":     "iop",
		"uniq_cloud_domain_crt": "iop",
	}
	id := 22
	err = client.Updates(update, id)
	if err != nil {
		fmt.Println("更新失败:", err)
	} else {
		fmt.Println("更新成功")
	}
}
