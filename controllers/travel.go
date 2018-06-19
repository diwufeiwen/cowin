package controllers

import (
	"github.com/astaxie/beego/orm"
	"strconv"
)

type TravelController struct {
	OnlineController
}

// sid,begidx,counts
func (this *ExpendController) TravelQuery() {
	begidx, _ := this.GetInt64("begidx")
	counts, _ := this.GetInt64("counts")

	sql := ps("select * from `travel` where uid='%d' order by unix desc limit %d,%d;", this.User.UserId, begidx, counts)
	sqlc := ps("select count(id) as num from `travel` where uid='%d';", this.User.UserId)

	var total int = 0
	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sqlc).Values(&result)
	if err != nil {
		log("查询失败:err[%v]", err)
		this.Rec = &Recv{5, "查询总数失败", nil}
		return
	} else {
		total, _ = strconv.Atoi(result[0]["num"].(string))
	}

	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询失败:err[%v]", err)
		this.Rec = &Recv{5, "查询失败", nil}
		return
	}

	type RecvEx struct {
		Total  int
		Detail interface{}
	}
	this.Rec = &Recv{3, "查询成功!", &RecvEx{total, result}}

	return
}
