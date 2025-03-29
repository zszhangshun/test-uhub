package uniqinfo

import (
	"fmt"
	"reflect"

	"github.com/golang/glog"
)

type UhubUniqChannelInfo struct {
	UniqCloudChannelID int    `json:"uniqCloudChannelID" db:"uniq_cloud_channel_id" gorm:"column:uniq_cloud_channel_id"`
	UniqCloudDomain    string `json:"uniqCloudDomain" db:"uniq_cloud_domain" gorm:"column:uniq_cloud_domain"`
	UniqCloudDomainCrt string `json:"uniqCloudDomainCrt" db:"uniq_cloud_domain_crt" gorm:"column:uniq_cloud_domain_crt"`
	UniqCloudDomainKey string `json:"uniqCloudDomainKey" db:"uniq_cloud_domain_key" gorm:"column:uniq_cloud_domain_key"`
	UniqType           string `json:"uniqType" db:"uniq_type" gorm:"column:uniq_type"`
	AllRegion          string `json:"allRegion" db:"all_region" gorm:"column:all_region"`          // 如果需要布尔值，可以在解码后转换
	DeployRegion       string `json:"deployRegion" db:"deploy_region" gorm:"column:deploy_region"` // 前端传递的可能是字符串，这里先作为字符串接收
	ChannelStatus      string `json:"channelStatus" db:"status" gorm:"column:status;default:1"`    // 渠道状态，0：停用，1：启用
}
type UniqInfos struct {
	Info []*UhubUniqChannelInfo
}

func NewUniqInfos() *UniqInfos {
	return &UniqInfos{}
}

func GetUniqInfoByOneID(uis *UniqInfos, id int) (data UniqInfos, notFound []int, err error) {
	ids := []int{}
	ids = append(ids, id)
	data, notFound, err = GetUniqInfoByID(uis, ids)
	if err != nil {
		glog.Error("get channel %s,info faild: %s", id, err.Error())
		return UniqInfos{}, nil, err
	}
	return data, notFound, nil
}

// 获取多个 uniqinfo
func GetUniqInfoByID(uis *UniqInfos, ids []int) (data UniqInfos, notFound []int, err error) {

	// 初始化 notFound 列表，包含所有的 ids
	idSet := make(map[int]struct{})

	for _, id := range ids {
		idSet[id] = struct{}{}
	}
	// fmt.Println("len(ids):", len(ids))
	if len(ids) < 1 {

		return *uis, nil, nil
	}
	for _, info := range uis.Info {
		for id, _ := range idSet {

			// 输入的id和数据库中的ChannelID进行对比
			if id == info.UniqCloudChannelID {
				data.Info = append(data.Info, info)
				delete(idSet, id)
				fmt.Println("id:", id)

			}
		}

	}
	for id, _ := range idSet {
		notFound = append(notFound, id)
	}
	return data, notFound, nil
}

func GetFieldTag(st interface{}, fieldName, tagKey string) (fieldame string, tagValue string) {
	t := reflect.TypeOf(st)
	field, found := t.FieldByName(fieldName)
	if !found {
		return "", ""
	}
	tagValue = field.Tag.Get(tagKey)
	fieldame = field.Type.Name()
	return fieldame, tagValue
}
