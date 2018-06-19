package controllers

import (
	"github.com/astaxie/beego/orm"
	"strconv"
)

type AssetsController struct {
	OnlineController
}

// sid,pt_id(产品类型),keyword(手机号或编号),bdt,edt,begidx,counts
func (this *AssetsController) AssetsProducting() {
	// 身份判断
	if this.User.Flag != 8 && this.User.Flag != 1 {
		this.Rec = &Recv{5, ps("无此访问权限"), nil}
		return
	}

	// 查询已经付款的订单,生产中指的是未完成的订单
	pt_id, _ := this.GetInt32("pt_id")
	keyword := this.GetString("keyword")
	bdt, _ := this.GetInt64("bdt")
	edt, _ := this.GetInt64("edt")
	begidx, _ := this.GetInt64("begidx")
	counts, _ := this.GetInt32("counts")

	if !CheckArg(counts) {
		this.Rec = &Recv{5, "总数不能为空", nil}
		return
	}

	// 业务逻辑
	var sql, sqlc string
	sql = "SELECT ep.*,u.account,u.realname,tc.code from `enjoy_product` as ep,`user` as u,transport_company as tc where tc.id=ep.tpc_id and ep.user_id=u.id and ep.status<4 and ep.pay_status=1"
	sqlc = "SELECT count(ep.id) as num from `enjoy_product` as ep,`user` as u where ep.user_id=u.id and ep.status<4 and ep.pay_status=1"

	if pt_id > 0 {
		sql += ps(" and ep.pt_id=%d", pt_id)
		sqlc += ps(" and ep.pt_id=%d", pt_id)
	}

	if bdt > 0 && edt > 0 {
		sql += ps(" and ep.unix>=%d and ep.unix<%d", bdt, edt)
		sqlc += ps(" and ep.unix>=%d and ep.unix<%d", bdt, edt)
	}

	if keyword != "" {
		sql += ps(" and (ep.order_no like '%%%s%%' or u.account like '%%%s%%')", keyword, keyword)
		sqlc += ps(" and (ep.order_no like '%%%s%%' or u.account like '%%%s%%')", keyword, keyword)
	}

	sql += ps(" order by ep.unix limit %d,%d;", begidx, counts)
	sqlc += ";"

	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sqlc).Values(&result)
	if err != nil {
		log("查询订单总数出错:[%v]", err)
		this.Rec = &Recv{5, "查询订单总数失败", nil}
		return
	}

	type RecvEx struct {
		Total  int
		Detail interface{}
	}
	var data RecvEx
	if result[0]["num"] != nil {
		data.Total, _ = strconv.Atoi(result[0]["num"].(string))
	} else {
		data.Total = 0
	}

	if data.Total > 0 {
		_, err = db.Raw(sql).Values(&result)
		if err != nil {
			log("查询订单出错:[%v]", err)
			this.Rec = &Recv{5, "查询订单失败", nil}
			return
		}
		data.Detail = result
	}

	this.Rec = &Recv{3, "查询订单成功", &data}
	return
}

// sid,pt_id(产品类型),friend_status(0-全部,1-正常使用,2-非正常),begidx,counts
func (this *AssetsController) AssetsPutin() {
	// 身份判断
	if this.User.Flag != 8 && this.User.Flag != 1 {
		this.Rec = &Recv{5, ps("无此访问权限"), nil}
		return
	}

	pt_id, _ := this.GetInt32("pt_id")
	friend_status, _ := this.GetInt32("friend_status")
	begidx, _ := this.GetInt64("begidx")
	counts, _ := this.GetInt32("counts")

	if !CheckArg(counts) {
		this.Rec = &Recv{5, "总数不能为空", nil}
		return
	}

	// 业务逻辑
	var sql, sqlc string
	sql = "SELECT up.*,u.account,u.realname from `user_product` as up,`enjoy_product` as ep,`user` as u where up.user_id=u.id and up.ep_id=ep.id and up.product_no!=''"
	sqlc = "SELECT count(up.id) as num from `user_product` as up,`enjoy_product` as ep,`user` as u where up.user_id=u.id and up.ep_id=ep.id and up.product_no!=''"

	if pt_id > 0 {
		sql += ps(" and ep.pt_id=%d", pt_id)
		sqlc += ps(" and ep.pt_id=%d", pt_id)
	}

	if friend_status > 0 {
		switch friend_status {
		case 1:
			sql += " and up.friend_status=1"
			sqlc += " and up.friend_status=1"
		case 2:
			sql += " and up.friend_status!=1"
			sqlc += " and up.friend_status!=1"
		}

	}
	sql += ps(" and up.status=4 order by up.unix limit %d,%d;", begidx, counts)
	sqlc += ";"

	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sqlc).Values(&result)
	if err != nil {
		log("查询产品总数出错:[%v]", err)
		this.Rec = &Recv{5, "查询产品总数失败", nil}
		return
	}

	type RecvEx struct {
		Total  int
		Detail interface{}
	}
	var data RecvEx
	if result[0]["num"] != nil {
		data.Total, _ = strconv.Atoi(result[0]["num"].(string))
	} else {
		data.Total = 0
	}

	if data.Total > 0 {
		_, err = db.Raw(sql).Values(&result)
		if err != nil {
			log("查询产品出错:[%v]", err)
			this.Rec = &Recv{5, "查询产品失败", nil}
			return
		}
		data.Detail = result

		// 涨粉数,使用次数
		for idx := range result {
			item := result[idx]

			sql = ps("select sum(growth_fans_num) as growth_fans_num,sum(use_num) as use_num from `product_use` where up_id='%s';", item["id"].(string))
			var res []orm.Params
			_, err = db.Raw(sql).Values(&res)
			if err != nil {
				log("查询产品使用信息出错:[%v]", err)
			} else {
				item["growth_fans_num"] = res[0]["growth_fans_num"]
				item["use_num"] = res[0]["use_num"]
			}
		}
	}

	this.Rec = &Recv{3, "查询产品成功", &data}
	return
}

// sid,pt_id,status(1-未处理,2-已核实),begidx,counts
func (this *AssetsController) AssetsOrderPickingup() {
	// 身份判断
	if this.User.Flag != 8 && this.User.Flag != 1 {
		this.Rec = &Recv{5, ps("无此访问权限"), nil}
		return
	}
	pt_id, _ := this.GetInt32("pt_id")
	status, _ := this.GetInt32("status")
	begidx, _ := this.GetInt64("begidx")
	counts, _ := this.GetInt32("counts")

	if !CheckArg(counts, status) {
		this.Rec = &Recv{5, "总数和status不能为空", nil}
		return
	}

	// 业务逻辑
	var sql, sqlc string
	switch status {
	case 1:
		sql = "SELECT ep.*,u.account,u.realname,tc.code,tc.name from `enjoy_product` as ep,`user` as u,`transport_company` as tc where ep.tpc_id=tc.id and ep.user_id=u.id and ep.status<4 and ep.pay_status=1 and ep.hosted_mid=2"
		sqlc = "SELECT count(ep.id) as num from `enjoy_product` as ep,`user` as u where ep.user_id=u.id and ep.status<4 and ep.pay_status=1 and ep.hosted_mid=2"
	case 2:
		sql = "SELECT ep.*,u.account,u.realname,tc.code,tc.name from `enjoy_product` as ep,`user` as u,`transport_company` as tc where ep.tpc_id=tc.id and ep.user_id=u.id and ep.status=4 and ep.pay_status=1 and ep.hosted_mid=2"
		sqlc = "SELECT count(ep.id) as num from `enjoy_product` as ep,`user` as u where ep.user_id=u.id and ep.status<4 and ep.pay_status=1 and ep.hosted_mid=2"
	}

	if pt_id > 0 {
		sql += ps(" and ep.pt_id=%d", pt_id)
		sqlc += ps(" and ep.pt_id=%d", pt_id)
	}

	sql += ps(" order by ep.unix limit %d,%d;", begidx, counts)
	sqlc += ";"

	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sqlc).Values(&result)
	if err != nil {
		log("查询订单总数出错:[%v]", err)
		this.Rec = &Recv{5, "查询订单总数失败", nil}
		return
	}

	type RecvEx struct {
		Total  int
		Detail interface{}
	}
	var data RecvEx
	if result[0]["num"] != nil {
		data.Total, _ = strconv.Atoi(result[0]["num"].(string))
	} else {
		data.Total = 0
	}

	if data.Total > 0 {
		_, err = db.Raw(sql).Values(&result)
		if err != nil {
			log("查询订单出错:[%v]", err)
			this.Rec = &Recv{5, "查询订单失败", nil}
			return
		}
		data.Detail = result
	}

	this.Rec = &Recv{3, "查询订单成功", &data}
	return
}

// sid,status(1-未处理,2-已核实),begidx,counts
func (this *AssetsController) AssetsPdtPickingup() {
	// 身份判断
	if this.User.Flag != 8 && this.User.Flag != 1 {
		this.Rec = &Recv{5, ps("无此访问权限"), nil}
		return
	}
	begidx, _ := this.GetInt64("begidx")
	counts, _ := this.GetInt32("counts")
	status, _ := this.GetInt32("status")

	if !CheckArg(counts) {
		this.Rec = &Recv{5, "总数不能为空", nil}
		return
	}

	// 业务逻辑
	var sql, sqlc string
	switch status {
	case 1:
		sql = ps("SELECT u.account,upu.*,tc.code,tc.name from `userpdt_pickup` as upu,`transport_company` as tc,`user` as u where upu.uid=u.id and upu.tpc_id=tc.id and upu.`status`<2 limit %d,%d;", begidx, counts)
		sqlc = "SELECT id from `userpdt_pickup` where `status`<2;"
	case 2:
		sql = ps("SELECT u.account,upu.*,tc.code,tc.name from `userpdt_pickup` as upu,`transport_company` as tc,`user` as u where upu.uid=u.id and upu.tpc_id=tc.id and upu.`status`=2 limit %d,%d;", begidx, counts)
		sqlc = "SELECT id from `userpdt_pickup` where `status`=2;"
	}

	db := orm.NewOrm()
	var result []orm.Params
	nums, err := db.Raw(sqlc).Values(&result)
	if err != nil {
		log("查询订单总数出错:[%v]", err)
		this.Rec = &Recv{5, "查询订单总数失败", nil}
		return
	}

	type RecvEx struct {
		Total  int64
		Detail interface{}
	}

	if nums <= 0 {
		this.Rec = &Recv{3, "查询订单成功", &RecvEx{nums, nil}}
		return
	}

	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询订单出错:[%v]", err)
		this.Rec = &Recv{5, "查询订单失败", nil}
		return
	}

	this.Rec = &Recv{3, "查询订单成功", nil}
	return
}

// sid,id(产品id),status(-1-报废,0-重新投放),reason
func (this *AssetsController) AssetsScrap() {
	// 身份判断
	if this.User.Flag != 8 && this.User.Flag != 1 {
		this.Rec = &Recv{5, ps("无此访问权限"), nil}
		return
	}

	id, _ := this.GetInt32("id")
	status, _ := this.GetInt32("status")
	reason := this.GetString("reason")

	if !CheckArg(id) {
		this.Rec = &Recv{5, "产品id不能为空", nil}
		return
	}

	var sql string
	if status == 0 {
		status = 4
	}
	db := orm.NewOrm()
	sql = ps("update `user_product` set status='%d',reason='%s' where id=%d", status, reason, id)
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("报废或重新投放失败:[%v]", err)
		this.Rec = &Recv{3, "报废或重新投放失败", nil}
		return
	}

	this.Rec = &Recv{3, "报废或重新投放成功", nil}
	return
}
