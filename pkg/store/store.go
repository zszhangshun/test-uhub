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

func (c *Client) Updates(update map[string]interface{}, id int) (err error) {
	// 开启事务
	tx := c.dbClient.Begin()
	if err := tx.Error; err != nil {
		return fmt.Errorf("begin transaction failed: %w", err)
	}
	// 捕获 panic并回滚事务
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			msg := fmt.Sprintf("Update panic recovered: %s", r)
			glog.Error(msg)
		}
	}()

	// 执行数据库操作
	if err = tx.Table(c.tableName).Where("uniq_cloud_channel_id = ?", id).
		Updates(update).Error; err != nil {
		tx.Rollback()
		msg := fmt.Errorf("update channel failed, channelId [%d],due to %w", id, err)
		glog.Error(msg)
		return msg
	}
	// 提交事务
	if commitErr := tx.Commit().Error; commitErr != nil {
		msg := fmt.Errorf("commit channel transaction failed channelId [%d],due to  %w", id, commitErr)
		glog.Error(msg)
		return msg
	}

	return nil
}

func (c *Client) Delete(id int) error {
	tx := c.dbClient.Begin()
	if err := tx.Error; err != nil {
		return fmt.Errorf("begin transaction failed: %w", err)
	}
	// 捕获 panic并回滚事务
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			msg := fmt.Sprintf("Update panic recovered: %s", r)
			glog.Error(msg)
		}
	}()
	// 执行数据库操作
	err := tx.Table(c.tableName).Where("uniq_cloud_channel_id = ?", id).
		Update("status", 0).Error
	if err != nil {
		tx.Rollback()
		msg := fmt.Errorf("delete channel failed, channelId [%d],due to %w", id, err)
		glog.Error(msg)
		return msg
	}
	// 提交事务
	if commitErr := tx.Commit().Error; commitErr != nil {
		tx.Rollback()
		msg := fmt.Errorf("commit channel transaction failed channelId [%d],due to  %w", id, commitErr)
		glog.Error(msg)
		return msg
	}

	return nil

}

func (c *Client) Create(value interface{}) error {
	tx := c.dbClient.Begin()
	if err := tx.Error; err != nil {
		return fmt.Errorf("begin transaction failed: %w", err)
	}
	// 捕获 panic并回滚事务
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			msg := fmt.Sprintf("Update panic recovered: %s", r)
			glog.Error(msg)
		}
	}()
	// 执行数据库操作
	err := tx.Table(c.tableName).Create(value).Error
	if err != nil {
		tx.Rollback()
		msg := fmt.Errorf("create channel failed, due to %w", err)
		glog.Error(msg)
		return msg
	}
	// 提交事务
	commitErr := tx.Commit().Error
	if commitErr != nil {
		tx.Rollback()
		msg := fmt.Errorf("commit channel transaction failed, due to  %w", commitErr)
		glog.Error(msg)
		return msg
	}
	return nil
}
