package store

import (
	"fmt"
	"test/pkg/uniqinfo"
	"time"

	"github.com/golang/glog"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	duration = time.NewTicker(2 * time.Minute) // 定义ticker但不立即发送数据到通道
	flush    = make(chan bool, 1)
)

func init() {
	flush <- true
}

type Client struct {
	dbClient  *gorm.DB
	tableName string
}

func NewClient(db *gorm.DB, tableName string) *Client {
	return &Client{
		dbClient:  db,
		tableName: tableName,
	}
}

func (c *Client) DBClient() *gorm.DB {
	if c == nil {
		return nil
	}
	return c.dbClient
}

// 初始化数据库
func New(user, password, host, port, database string) (*gorm.DB, error) {
	dsn := user + ":" + password + "@tcp(" + host + ":" + port + ")/" + database + "?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		return nil, fmt.Errorf("gorm open failed: %w", err)
	}

	// sqlDB, err := db.DB()
	// if err != nil {
	// 	return nil, fmt.Errorf("get underlying sql.DB failed: %w", err)
	// }

	// // 配置连接池
	// sqlDB.SetMaxOpenConns(c.DB.MaxOpenConns)
	// sqlDB.SetMaxIdleConns(c.DB.MaxIdleConns)
	// sqlDB.SetConnMaxLifetime(time.Duration(c.DB.ConnMaxLifetime) * time.Minute)
	return db, nil
}

// 获取所有的uniqinfo
func (c *Client) GetAllUniqInfo(allInfo *uniqinfo.UniqInfos) error {
	// 执行查询操作
	if err := c.dbClient.Table("uhub_uniq_info").Where("status != 0").Find(&allInfo.Info).Error; err != nil {
		// 记录错误日志并返回错误
		glog.Errorf("get uniq info failed: %v", err)
		return fmt.Errorf("get uniq info failed: %w", err)
	}

	// 检查是否有数据返回
	if len(allInfo.Info) < 1 {
		// 数据为空时记录警告日志并返回特定错误
		glog.Warningf("no uniq info found")
		return fmt.Errorf("no uniq info found")
	}

	return nil
}

// 定时刷新或者手动刷新
func (c *Client) Flush(allInfo *uniqinfo.UniqInfos) error {
	if c == nil {
		return fmt.Errorf("UniqInfos instance is nil")
	}
	if allInfo == nil {
		return fmt.Errorf("UniqInfos instance is nil")
	}

	for {
		select {
		case <-duration.C:
			fmt.Println("定时刷新")
			if err := c.GetAllUniqInfo(allInfo); err != nil {
				glog.Errorf("定时刷新失败: %v", err)
				// 根据业务逻辑决定是否继续运行或返回错误
				continue // 或者 break/return 根据实际情况调整
			}
			fmt.Println("定时刷新成功")

		case <-flush:
			fmt.Println("手动刷新")
			if err := c.GetAllUniqInfo(allInfo); err != nil {
				glog.Errorf("手动刷新失败: %v", err)
				// 同样根据业务逻辑决定后续操作
				continue // 或者 break/return 根据实际情况调整
			}
			fmt.Println("手动刷新成功")
		}
	}
}

func (c *Client) FlushVaule() {
	flush <- true
}
