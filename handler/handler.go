package api

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"test/pkg/config"
	"test/pkg/store"
	"test/pkg/uniqinfo"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"gorm.io/gorm"
)

var (
	ErrRequestBodyEmpty   = errors.New("请求体为空")
	ErrInvalidContentType = errors.New("无效的 Content-Type")
)

type DeleteChannelResponse struct {
	ChannelID    int    `json:"channel_id"`
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
func (h *Handle) applyChanges(changes map[string]string, id int) (err error) {
	// 构建更新map
	st := uniqinfo.UhubUniqChannelInfo{}
	updates := make(map[string]interface{})
	for field, value := range changes {
		// 使用GORM的默认列名转换规则（蛇形命名）
		_, tag := uniqinfo.GetFieldTag(st, field, "db")

		updates[tag] = value
	}

	// 批量更新
	err = h.Store.Updates(updates, id)
	if err != nil {
		msg := fmt.Errorf("更新渠道信息失败:%s", err.Error())
		glog.Error(msg)
		return msg
	}
	return nil

}

// 身份认证
func (h *Handle) Authentication(ctx *gin.Context) {
	// token := ctx.GetHeader("Authorization")
	// if token == "" {
	// 	message := "Token 不能为空"
	// 	glog.Error(message)
	// 	ctx.JSON(http.StatusUnauthorized, gin.H{
	// 		"code":    401,
	// 		"message": message,
	// 		"changes": nil,
	// 	})
	// 	return
	// }
	// // 验证Token
	// 假设 token 验证通过
	ctx.Next()

}

// 参数验证:
func (h *Handle) ValidateParamsCheck(ctx *gin.Context) {
	// 首先读取整个请求体
	bodyBytes, err := io.ReadAll(ctx.Request.Body)
	// 将读取的数据重新赋值给 ctx.Request.Body
	if err != nil {
		msg := fmt.Errorf("读取请求体失败: %v", err)
		glog.Error(msg)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": msg,
			"changes": nil,
		})
		return
		return
	}

	// 将读取的数据重新赋值给 ctx.Request.Body，以便后续处理
	ctx.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	var channelInfo uniqinfo.UhubUniqChannelInfo
	if err := ctx.ShouldBindJSON(&channelInfo); err != nil {
		message := fmt.Sprintf("请求解析失败: %v", err)
		glog.Errorf(message)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": message,
		})
		return
	}

	ctx.Set("channelInfoSave", channelInfo)
	ctx.Next()
}

// 跟新渠道信息
func (h *Handle) UpdateChannelinfo() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		isConfirm := ctx.GetHeader("X-Confirm") == "true"
		queryId := ctx.Param("id")
		id, err := strconv.Atoi(queryId)
		if err != nil {
			message := fmt.Sprintf("id 转换失败: %v", err)
			glog.Error(message)
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": message,
				"changes": nil,
			})
			return
		}
		// 1. 获取更新信息
		channelInfo, err := h.GetRequestBody(ctx)
		if err != nil {
			msg := fmt.Sprintf("获取新渠道信息失败：%s", err)
			glog.Error(msg)
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": msg,
				"changes": nil,
			})

			return
		}

		// 2. 检查变化
		changes, err := h.checkChanges(channelInfo)
		fmt.Println("changes:", changes)
		if !isConfirm {
			if err != nil {
				message := fmt.Sprintf("变化检查失败: %v", err)
				glog.Errorf(message)
				ctx.JSON(http.StatusInternalServerError, gin.H{
					"code":    500,
					"message": message,
					"changes": nil,
				})
				return
			}
		}
		if len(changes) < 1 {
			glog.Info("数据无变化")
			ctx.JSON(http.StatusOK, gin.H{
				"code":    204,
				"message": "数据无变化",
				"changes": map[string]string{},
			})
			return
		}
		// 检查证书有效性
		err = h.checkCertAndKey(ctx)
		if err != nil {
			msg := fmt.Sprintf("检查证书有效性失败，原因:%s", err)
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    404,
				"message": msg,
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
	var currentPage int
	if ctx.Query("page") != "" {
		currentPage, _ = strconv.Atoi(ctx.Query("page"))
	}
	if currentPage == 0 {
		currentPage = 1
	}
	ctx.HTML(200, "channel.tmpl", gin.H{
		"base_prefix":        "/uhub/v1",
		"uhub_uniq_total":    len(h.UhubUniqInfo.Info),
		"uniq_channel_infos": h.UhubUniqInfo.Info,
		"uniq_list":          h.UhubUniqInfo.Info,
	})
}

func (h *Handle) checkUniqExists(id int) (status bool, err error) {
	var result uniqinfo.UhubUniqChannelInfo
	err = h.Store.DBClient().Table(h.Config.DB.Table).
		Where("uniq_cloud_channel_id = ?", id).
		//过滤status为0的数据，0为删除状态
		Where("status != 0").
		First(&result).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		message := fmt.Sprintf("check channel exists failed due to err:%s", err.Error())
		return false, errors.New(message)
	}
	if result.UniqCloudChannelID != 0 {
		message := fmt.Sprintf("check channel exists %d", id)
		return false, errors.New(message)

	}
	return true, nil
}

// 创建新渠道
func (h *Handle) CreateNewChannel(ctx *gin.Context) {
	success := true
	//updateInfoInterface, exists := ctx.Get("updateInfo")
	newChannel, err := h.GetRequestBody(ctx)
	if err != nil {
		msg := fmt.Sprintf("获取新渠道信息失败：%s", err)
		glog.Error(msg)
		success = false
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": msg,
			"changes": nil,
		})

		return
	}
	//newChannel, ok := updateInfoInterface.(uniqinfo.UhubUniqChannelInfo)

	// 新增渠道状态设置为1
	if newChannel.ChannelStatus == "" {
		newChannel.ChannelStatus = "1"
	}

	ok, err := h.checkUniqExists(newChannel.UniqCloudChannelID)
	if err != nil {
		success = false
		message := fmt.Sprintf("无法创建渠道，原因:%s", err.Error())
		ctx.JSON(http.StatusConflict, gin.H{
			"code":    http.StatusConflict,
			"message": message,
		})

		return
	}
	if !ok {
		success = false
		msg := fmt.Sprintf("检查渠道是否存在失败 id :%d", newChannel.UniqCloudChannelID)
		glog.Error(msg)
		ctx.JSON(http.StatusConflict, gin.H{
			"code":    http.StatusConflict,
			"message": msg,
		})

		return
	}
	//检查证书有效性
	err = h.checkCertAndKey(ctx)
	if err != nil {
		success = false
		msg := fmt.Sprintf("检查证书有效性失败，原因:%s", err)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    404,
			"message": msg,
		})
		return
	}
	//执行数据库操作
	err = h.Store.Create(&newChannel)
	if err != nil {
		success = false
		msg := fmt.Sprintf("create channel failed with error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": msg,
		})
		return
	}
	if success {
		ctx.JSON(http.StatusCreated, gin.H{
			"code":    201,
			"message": "创建渠道成功",
		})
	}
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
		msg := fmt.Sprintf("json  Mashal failed,err: %s", err)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": msg,
		})
		return
	}
	if req.DeleteStatus != "true" {
		glog.Info("Begion delete channel  %s", req.ChannelID)
		err := h.Store.Delete(req.ChannelID)
		if err != nil {
			msg := fmt.Sprintf("Delete channel  %d failed,err: %s", req.ChannelID, err)
			glog.Error(msg)
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": msg,
			})
			return
		}

	}
	msg := fmt.Sprintf("Delete channel  %d suceessful", req.ChannelID)
	glog.Info(msg)
	// 返回成功的响应
	h.FlushVaule()
	ctx.JSON(http.StatusOK, gin.H{
		"message": msg,
		"code":    200,
	})
}

// 检查cert和key文件的有效性
func (h *Handle) checkCertAndKey(ctx *gin.Context) error {
	channelInfo, err := h.GetRequestBody(ctx)
	if err != nil {
		message := fmt.Errorf("验证证书时，获取渠道信息失败: %v", err)
		log.Println(message)
		return message
	}

	cert := channelInfo.UniqCloudDomainCrt
	key := channelInfo.UniqCloudDomainKey
	if cert == "" || key == "" {
		message := fmt.Errorf("渠道信息中未找到证书或私钥")
		log.Println(message)
		return message
	}

	// 由于cert和key是字符串形式的证书和密钥，我们需要先将它们写入临时文件或者使用字节流处理
	// 这里假设cert和key是以PEM格式的字符串存储
	certBytes := []byte(cert)
	keyBytes := []byte(key)

	// 使用临时文件或直接从内存中加载证书和私钥
	certificate, err := tls.X509KeyPair(certBytes, keyBytes)
	if err != nil {
		msg := fmt.Errorf("无法解析证书或私钥: %v", err)
		glog.Error(msg)
		return msg
	}

	// 解析证书以获取详细信息
	block, _ := pem.Decode(certificate.Certificate[0])
	if block == nil {
		msg := fmt.Errorf("无法解码证书数据")
		glog.Error(msg)
		return msg
	}
	x509Cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		msg := fmt.Errorf("解析证书失败: %v", err)
		glog.Error(msg)
		return msg

	}

	// 检查证书有效期
	now := time.Now()
	if now.Before(x509Cert.NotBefore) {
		msg := fmt.Errorf("证书未生效，有效期从: %v 开始", x509Cert.NotBefore)
		glog.Error(msg)
		return msg

	}
	if now.After(x509Cert.NotAfter) {
		msg := fmt.Errorf("证书已过期，有效期至: %v", x509Cert.NotAfter)
		glog.Error(msg)
		return msg
	}
	fmt.Printf("证书有效，有效期: %v 至 %v\n", x509Cert.NotBefore, x509Cert.NotAfter)

	// 打印确认消息
	msg := fmt.Sprintf("%d 证书和私钥匹配", channelInfo.UniqCloudChannelID)
	glog.Info(msg)

	return nil
}

// GetRequestBody 从请求体中读取原始字节数据
func (h *Handle) GetRequestBody(ctx *gin.Context) (channelInfo *uniqinfo.UhubUniqChannelInfo, err error) {
	// 1. 读取请求体
	body, err := io.ReadAll(ctx.Request.Body)
	ctx.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	if err != nil {
		// 返回自定义错误响应
		var errMsg string
		switch {
		case errors.Is(err, ErrRequestBodyEmpty):
			errMsg = "请求体为空，请提供有效的渠道信息"
		// case errors.Is(err, ErrInvalidContentType):
		// 	errMsg = "无效的 Content-Type，期望 application/json"
		default:
			errMsg = "无法解析请求体，请检查请求格式"
		}
		return nil, errors.New(errMsg)
	}
	// 2. 检查请求体是否为空
	if len(body) == 0 {
		msg := "请求体为空"
		return nil, errors.New(msg)
	}

	// 3. 可选：验证 Content-Type（例如 JSON 格式）
	// if ctx.Request.Header.Get("Content-Type") != "application/json" {
	//     return nil, fmt.Errorf("无效的 Content-Type，期望 application/json")
	// }
	err = json.Unmarshal(body, &channelInfo)
	if err != nil {
		//message := fmt.Sprintf("类型断言失败: %v", updateInfoInterface)
		msg := fmt.Errorf("请求体无法转换： %s", err)
		glog.Error(msg)

		return nil, msg
	}
	return channelInfo, nil
}
