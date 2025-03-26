package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"test/pkg/config"
	"test/pkg/store"
	"test/pkg/uniqinfo"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
)

const (
	DESCRIBEUNIQINFO = "DescribeUniqInfo"
	DEPLOYUNIQ       = "DeployUniq"
	UPDATEINFO       = "UpdateInfo"
)

type DeleteChannelResponse struct {
	ChannelID    string `json:"channel_id"`
	DeleteStatus string `json:"delete_status"`
}
type Handle struct {
	Config       *config.Config
	Store        *store.Client
	UhubUniqInfo *uniqinfo.UniqInfos
}

//检查变化

func (h *Handle) checkChanges(leastInfoinfo *uniqinfo.UhubUniqChannelInfo) (changes map[string]string, err error) {
	// 实现变化检查逻辑
	// 返回变化字段列表和可能的错误
	if leastInfoinfo.UniqCloudChannelID == 0 {
		err = errors.New("uniq_cloud_channel_id 不能为空")
		return nil, err
	}
	older, notFound, err := uniqinfo.GetUniqInfoByOneID(h.UhubUniqInfo, leastInfoinfo.UniqCloudChannelID)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	if len(notFound) > 0 {
		err = fmt.Errorf("channel id %d not found", leastInfoinfo.UniqCloudChannelID)
		glog.Error(err)
		return nil, err
	}
	v1 := reflect.ValueOf(leastInfoinfo).Elem() // 获取指针指向的值
	v2 := reflect.ValueOf(older.Info[0]).Elem()
	//保存渠道id
	// changes = map[string]string{
	// 	"UniqCloudChannelID": leastInfoinfo.UniqCloudChannelID,
	// }
	for i := 0; i < v1.NumField(); i++ {
		field := v1.Type().Field(i)
		value1 := v1.Field(i).String()
		value2 := v2.Field(i).String()

		if !reflect.DeepEqual(value1, value2) {
			// changes = append(changes, map[string]string{
			// 	field.Name: change,
			// })
			fmt.Println("field.Name:", field.Name)
			changes = map[string]string{

				field.Name: value1,
			}
		}
	}
	return changes, nil
}

// 应用变化
func (h *Handle) applyChanges(changes map[string]string, id string) (err error) {
	// 开启事务
	tx := h.Store.DBClient().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// 构建更新map
	st := uniqinfo.UhubUniqChannelInfo{}
	updates := make(map[string]interface{})
	for field, value := range changes {
		// 使用GORM的默认列名转换规则（蛇形命名）
		_, tag := uniqinfo.GetFieldTag(st, field, "db")

		updates[tag] = value
	}

	// 批量更新
	if err := tx.Table(h.Config.DB.Table).
		Where("uniq_cloud_channel_id = ?", id).
		Updates(updates).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("事务提交失败: %w", err)
	}

	return nil

}

// 跟新渠道信息
func (h *Handle) UpdateChannelinfo() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		isConfirm := ctx.GetHeader("X-Confirm") == "true"
		id := ctx.Param("id")
		// 1. 解析请求
		var updateInfo *uniqinfo.UhubUniqChannelInfo
		if err := ctx.ShouldBindJSON(&updateInfo); err != nil {
			glog.Errorf("请求解析失败: %v", err)
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "无效的请求格式",
				"changes": nil,
			})
			return
		}
		fmt.Println("updateInfo:", updateInfo)
		// 2. 检查变化
		changes, err := h.checkChanges(updateInfo)
		fmt.Println("changes:", changes)
		if !isConfirm {
			if err != nil {
				glog.Errorf("变化检查失败: %v", err)
				ctx.JSON(http.StatusInternalServerError, gin.H{
					"Code":    500,
					"Message": "无法检查数据变化",
					"Changes": nil,
				})
				return
			}
		}
		fmt.Println("len:", len(changes))
		if len(changes) < 1 {
			ctx.JSON(http.StatusOK, gin.H{
				"code":    204,
				"message": "数据无变化",
				"changes": map[string]string{},
			})
			return
		}
		// 第二阶段：执行更新
		if err := h.applyChanges(changes, id); err != nil {
			glog.Errorf("更新失败: %v", err)
			message := fmt.Sprintf("数据库更新失败: %s", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": message,
				"changes": changes,
			})
			return
		}
		// 4. 刷新缓存
		h.flushVaule()
		ctx.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "更新成功",
			"changes": changes,
		})
	}
}

// 刷新缓存接口
func (h *Handle) FlushVaule() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		h.flushVaule()

	}

}

// 刷新缓存
func (h *Handle) flushVaule() {
	h.Store.FlushVaule()
}

// 首页
func (h *Handle) IndexHtml(ctx *gin.Context) {

	var total int64
	_, err := h.Store.DBClient().DB()
	if err != nil {
		ctx.HTML(200, "404.tmpl", gin.H{
			"error": err.Error(),
		})
		return
	}

	err = h.Store.DBClient().Table(h.Config.DB.Table).Count(&total).Error
	if err != nil {
		ctx.HTML(200, "404.tmpl", gin.H{
			"error": err.Error(),
		})
		return
	}
	ctx.HTML(200, "index.tmpl", gin.H{
		"uhub_uniq_total": total,
		"uniq_list":       h.UhubUniqInfo.Info,
	})
}

// 渠道总览
func (h *Handle) ChannelTotal(ctx *gin.Context) {
	h.flushVaule()
	area := "10"
	var currentPage int
	if ctx.Query("page") != "" {
		currentPage, _ = strconv.Atoi(ctx.Query("page"))
	}
	if currentPage == 0 {
		currentPage = 1
	}
	ctx.HTML(200, "channel.tmpl", gin.H{
		"area":               area,
		"uhub_uniq_total":    len(h.UhubUniqInfo.Info),
		"uniq_channel_infos": h.UhubUniqInfo.Info,
		"uniq_list":          h.UhubUniqInfo.Info,
	})
}

// 创建新渠道
func (h *Handle) CreateNewChannel(ctx *gin.Context) {
	// 开启事务
	tx := h.Store.DBClient().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	var newChannel uniqinfo.UhubUniqChannelInfo
	if err := ctx.ShouldBindJSON(&newChannel); err != nil {
		msg := fmt.Sprintf("create new channel params feild due to err:%s", err.Error())
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": msg,
		})
		log.Println("Error decoding JSON:", err)
		return
	}
	// 新增渠道状态设置为1
	if newChannel.ChannelStatus == "" {
		newChannel.ChannelStatus = "1"
	}
	err := h.Store.DBClient().Table(h.Config.DB.Table).Create(newChannel).Error
	if err != nil {
		msg := fmt.Sprintf("create new channel failed due to err:%s", err.Error())
		ctx.JSON(404, gin.H{
			"error": msg,
		})
		log.Println(":", err)
		return

	}
	if err := tx.Commit().Error; err != nil {
		msg := fmt.Sprintf("create new channel failed due to err:%s", err.Error())
		ctx.JSON(404, gin.H{
			"error": msg,
		})
		log.Println(":", err)
		return
	}
	// 返回成功的响应
	response := gin.H{
		"message": "Channel added successfully",
	}
	ctx.JSON(http.StatusOK, response)
}

// 删除渠道
func (h *Handle) DeleteChannel(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "id is required",
		})
		return
	}
	var req *DeleteChannelResponse
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error()})
		return
	}
	if req.DeleteStatus != "true" {
		log.Printf("Begion delete channel  %s", req.ChannelID)
		err := h.Store.DBClient().Table(h.Config.DB.Table).Where("uniq_cloud_channel_id = ?", id).Update("status", 0).Error
		if err != nil {
			msg := fmt.Sprintf("delete channel failed due to err:%s", err.Error())
			ctx.JSON(404, gin.H{
				"code":  404,
				"error": msg,
			})
			log.Printf("Delete channel  %s failed,err:%s", req.ChannelID, err)
			return
		}
		log.Printf("Delete channel  %s suceessful", req.ChannelID)
	}

	// 返回成功的响应
	response := gin.H{
		"message": "Channel deleted successfully",
		"code":    200,
	}
	h.FlushVaule()
	ctx.JSON(http.StatusOK, response)
}

func (h *Handle) CheckUniqInfoParam(info *uniqinfo.UhubUniqChannelInfo) error {

	return nil
}
