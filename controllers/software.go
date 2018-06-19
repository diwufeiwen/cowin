package controllers

import "github.com/astaxie/beego/orm"

type SoftwareController struct {
	OnlineController
}

type SoftwareBaseController struct {
	BaseController
}

func (this *SoftwareBaseController)  QuerySoftwareInfo(){
	client , _ := this.GetInt32("client")


	sql := ps("select*from software_update where client=%d " , client)
	var result []orm.Params
	db := orm.NewOrm()

	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询软件更新信息失败:%s", err.Error())
		this.Rec = &Recv{5, "查询软件更新信息失败", nil}
		return
	}

	this.Rec = &Recv{3, ps("查询更新信息成功!"), result}
}
