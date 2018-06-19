package controllers

import (
	"github.com/astaxie/beego/orm"
	"strconv"
)

type SvrpvdController struct {
	OnlineController
}

// sid,pt_id,account,name,principal,phone,address,pwd,city
func (this *SvrpvdController) SvrpvdAdd() {
	// 身份判断
	if this.User.Flag != 1 {
		this.Rec = &Recv{5, ps("仅平台管理员可添加服务商"), nil}
		return
	}

	pt_id, _ := this.GetInt32("pt_id")
	account := this.GetString("account")
	name := this.GetString("name")
	principal := this.GetString("principal")
	phone := this.GetString("phone")
	address := this.GetString("address")
	pwd := this.GetString("pwd")
	city := this.GetString("city")

	//检查参数
	if !CheckArg(pt_id, account, name, principal, phone, address, pwd, city) {
		this.Rec = &Recv{5, "参数存在空值", nil}
		return
	}

	var sql string = ps("insert into `user` (pt_id,account,nick,realname,`phone`,address,pwd,flag,city,unix) values ('%d','%s','%s','%s','%s','%s','%s','6','%s','%d');",
		pt_id, account, name, principal, phone, address, StrToMD5(ps("%s_Cowin_%s", account, pwd)), city, TimeNow)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("添加服务商失败:[%v]", err)
		this.Rec = &Recv{5, "添加服务商失败", nil}
		return
	}
	this.Rec = &Recv{3, "添加服务商成功", nil}
}

// sid
func (this *SvrpvdController) SvrpvdQuery() {
	sql := "select * from `user` where flag=6;"
	var result []orm.Params
	db := orm.NewOrm()
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询服务商失败:err[%v]", err)
		this.Rec = &Recv{5, ps("查询服务商失败"), nil}
		return
	}

	this.Rec = &Recv{3, ps("查询服务商成功!"), result}
}

// sid,id,name,principal,phone,address,pwd,city
func (this *SvrpvdController) SvrpvdModify() {
	id, _ := this.GetInt("id")
	name := this.GetString("name")
	principal := this.GetString("principal")
	phone := this.GetString("phone")
	address := this.GetString("address")
	pwd := this.GetString("pwd")
	city := this.GetString("city")

	//检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	//业务逻辑
	db := orm.NewOrm()
	var sql = "update `user` set "
	if name != "" {
		sql += ps("nick='%s',", name)
	}
	if principal != "" {
		sql += ps("realname='%s',", principal)
	}
	if phone != "" {
		sql += ps("phone='%s',", phone)
	}
	if address != "" {
		sql += ps("address='%s',", address)
	}
	if city != "" {
		sql += ps("city='%s',", city)
	}
	if pwd != "" {
		var result []orm.Params
		nums, err := db.Raw("select account from user where id=?", id).Values(&result)
		if err == nil && nums > 0 {
			pwd = StrToMD5(ps("%s_Cowin_%s", result[0]["account"].(string), pwd))
		}
		sql += ps("pwd='%s',", pwd)
	}
	sql += ps("unix='%d' where id=%d", TimeNow, id)
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("修改服务商信息失败:[%v]", err)
		this.Rec = &Recv{5, "修改服务商信息失败", nil}
		return
	}
	this.Rec = &Recv{3, "修改服务商信息成功", nil}
}

// sid,id
func (this *SvrpvdController) SvrpvdDel() {
	id, _ := this.GetInt64("id")

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	// 业务逻辑
	var sql = ps("delete from `user` where id='%d' and flag=6;", id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("删除服务商失败:[%v]", err)
		this.Rec = &Recv{5, "删除服务商失败", nil}
		return
	}
	this.Rec = &Recv{3, "删除服务商成功", nil}
	return
}

// sid
func (this *SvrpvdController) SvrpvdApplyQuery() {
	sql := "select m.*,u.account,u.pt_id from `service_provider` as m,user as u where m.offical=0 and m.uid=u.id;"
	var result []orm.Params
	db := orm.NewOrm()
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询服务商申请失败:err[%v]", err)
		this.Rec = &Recv{5, ps("查询服务商申请失败"), nil}
		return
	}

	this.Rec = &Recv{3, ps("查询服务商申请成功!"), result}
	return
}

// sid,id,agree(1-不通过,2-通过)
func (this *SvrpvdController) SvrpvdApplyDeal() {
	id, _ := this.GetInt64("id")
	agree, _ := this.GetInt32("agree")

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	// 业务逻辑
	var sql string
	db := orm.NewOrm()
	if 2 == agree {
		var result []orm.Params
		sql = ps("select * from `service_provider` where id=%d and offical=0;", id)
		cnts, err := db.Raw(sql).Values(&result)
		if err != nil {
			log("查询申请信息出错:[%v]", err)
			this.Rec = &Recv{5, "申请处理失败", nil}
			return
		} else {
			if cnts > 0 {
				name := result[0]["name"].(string)
				principal := result[0]["principal"].(string)
				phone := result[0]["phone"].(string)
				address := result[0]["address"].(string)
				city := result[0]["city"].(string)

				sql = ps("update `user` set ")
				if name != "" {
					sql += ps("nick='%s',", name)
				}
				if principal != "" {
					sql += ps("realname='%s',", principal)
				}
				if phone != "" {
					sql += ps("phone='%s',", phone)
				}
				if address != "" {
					sql += ps("address='%s',", address)
				}
				if city != "" {
					sql += ps("city='%s',", city)
				}
				sql += ps("unix='%d' where id='%s';", TimeNow, result[0]["uid"].(string))

				_, err = db.Raw(sql).Exec()
				if err != nil {
					log("更新申请服务商信息出错:[%v]", err)
					this.Rec = &Recv{5, "申请处理失败", nil}
					return
				}
			} else {
				this.Rec = &Recv{5, "待处理申请不存在", nil}
				return
			}
		}
	}

	// 删除申请
	sql = ps("update `service_provider` set offical=%d where id=%d;", agree, id)
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("删除申请信息出错:[%v]", err)
		this.Rec = &Recv{5, "申请处理成功,删除申请信息出错", nil}
		return
	}

	this.Rec = &Recv{3, "申请处理成功", nil}
	return
}

// sid,name,principal,phone,address,city
func (this *SvrpvdController) SvrpvdChangeApply() {
	// 身份判断
	if this.User.Flag != 6 {
		this.Rec = &Recv{5, ps("此接口只有运营商可以访问"), nil}
		return
	}

	name := this.GetString("name")
	principal := this.GetString("principal")
	phone := this.GetString("phone")
	address := this.GetString("address")
	city := this.GetString("city")

	// 业务逻辑
	sql := ps("select id from `service_provider` where offical=0 and uid=%d;", this.User.UserId)
	db := orm.NewOrm()
	var result []orm.Params
	cnts, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询变更申请失败:[%v]", err)
		this.Rec = &Recv{5, "查询变更申请失败", nil}
		return
	} else {
		if cnts > 0 {
			id, _ := strconv.Atoi(result[0]["id"].(string))
			sql = ps("update `service_provider` set ")
			if name != "" {
				sql += ps("name='%s'", name)
			}
			if principal != "" {
				sql += ps(",principal='%s'", principal)
			}
			if phone != "" {
				sql += ps(",phone='%s'", phone)
			}
			if address != "" {
				sql += ps(",address='%s'", address)
			}
			if city != "" {
				sql += ps(",city='%s'", city)
			}
			sql += ps(",unix='%d' where id='%d';", TimeNow, id)
		} else {
			sql = ps("insert into `service_provider` (uid,name,principal,phone,address,unix) values('%d','%s','%s','%s','%s','%d');", this.User.UserId, name, principal, phone, address, TimeNow)
		}
	}

	//log("%s", sql)
	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("提交变更信息出错:[%v]", err)
		this.Rec = &Recv{5, "提交信息变更申失败", nil}
		return
	}

	this.Rec = &Recv{3, "提交信息变更申请成功", nil}
	return
}

// sid
func (this *SvrpvdController) SvrpvdChangeQuery() {
	// 身份判断
	if this.User.Flag != 6 {
		this.Rec = &Recv{5, ps("此接口只有运营商可以访问"), nil}
		return
	}

	sql := "select * from `service_provider` where offical=0 order by unix desc;"
	var result []orm.Params
	db := orm.NewOrm()
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询申请失败:err[%v]", err)
		this.Rec = &Recv{5, ps("查询已提交变更申请失败"), nil}
		return
	}

	this.Rec = &Recv{3, ps("查询已提交变更申请成功!"), result}
	return
}

// sid,status(1-未完成,2-已完成),begidx,counts
func (this *SvrpvdController) SvrpvdPrdttsh() {
	// 身份判断
	if this.User.Flag != 6 {
		this.Rec = &Recv{5, ps("此接口只有运营商可以访问"), nil}
		return
	}

	status, _ := this.GetInt32("status")
	begidx, _ := this.GetInt64("begidx")
	counts, _ := this.GetInt32("counts")

	if !CheckArg(status) {
		this.Rec = &Recv{5, ps("查询状态不能为空"), nil}
		return
	}

	var sql, sqlc string
	switch status {
	case 1:
		sql = ps("select up.id,ep.order_no,u.account,u.nick,ep.hosted_city,up.product_no,up.friendpdt_no,up.status,up.unix from `user_product` as up,`enjoy_product` as ep,`user` as u where up.ep_id=ep.id and up.user_id=u.id and up.`hosted_mid`=1 and up.`status`<4 limit %d,%d;", begidx, counts)
		sqlc = "select id from `user_product` where `hosted_mid`=1 and `status`<4;"
	case 2:
		sql = ps("select up.id,ep.order_no,u.account,u.nick,ep.hosted_city,up.product_no,up.friendpdt_no,up.status,up.unix from `user_product` as up,`enjoy_product` as ep,`user` as u where up.ep_id=ep.id and up.user_id=u.id and up.`hosted_mid`=1 and up.`status`=4 limit %d,%d;", begidx, counts)
		sqlc = "select id from `user_product` where `hosted_mid`=1 and `status`=4;"
	}

	db := orm.NewOrm()
	var result []orm.Params
	nums, err := db.Raw(sqlc).Values(&result)
	if err != nil {
		log("查询总数失败:[%v]", err)
		this.Rec = &Recv{5, "查询失败", nil}
		return
	}

	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询失败:[%v]", err)
		this.Rec = &Recv{5, "查询失败", nil}
		return
	}

	type RecvEx struct {
		Total  int64
		Detail interface{}
	}
	this.Rec = &Recv{3, ps("查询成功!"), &RecvEx{nums, result}}
	return
}

// sid,status(1-未完成,2-已完成),begidx,counts
func (this *SvrpvdController) SvrpvdPrdpick() {
	// 身份判断
	if this.User.Flag != 6 {
		this.Rec = &Recv{5, ps("此接口只有运营商可以访问"), nil}
		return
	}

	status, _ := this.GetInt32("status")
	begidx, _ := this.GetInt64("begidx")
	counts, _ := this.GetInt32("counts")

	if !CheckArg(status) {
		this.Rec = &Recv{5, ps("查询状态不能为空"), nil}
		return
	}

	var sql, sqlc string
	switch status {
	case 1:
		sql = ps("select upu.*,u.account,u.nick,tc.name from `userpdt_pickup` as upu,`user` as u,transport_company as tc where tc.id=upu.tpc_id and upu.uid=u.id and upu.`status`<2 limit %d,%d;", begidx, counts)
		sqlc = "select id from `userpdt_pickup` where `status`<2;"
	case 2:
		sql = ps("select upu.*,u.account,u.nick,tc.name  from `userpdt_pickup` as upu,`user` as u,transport_company as tc where tc.id=upu.tpc_id and upu.uid=u.id and upu.`status`=2 limit %d,%d;", begidx, counts)
		sqlc = "select id from `userpdt_pickup` where `status`=2;"
	}

	db := orm.NewOrm()
	var result []orm.Params
	nums, err := db.Raw(sqlc).Values(&result)
	if err != nil {
		log("查询总数失败:[%v]", err)
		this.Rec = &Recv{5, "查询失败", nil}
		return
	}

	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询失败:[%v]", err)
		this.Rec = &Recv{5, "查询失败", nil}
		return
	}

	type RecvEx struct {
		Total  int64
		Detail interface{}
	}
	this.Rec = &Recv{3, ps("查询成功!"), &RecvEx{nums, result}}
	return
}

// sid,id
func (this *SvrpvdController) SvrpvdPickpdtno() {
	id, _ := this.GetInt64("id")

	if !CheckArg(id) {
		this.Rec = &Recv{5, ps("id不能为空"), nil}
		return
	}

	sql := ps("select product_no,friendpdt_no from `user_product` where id in (select up_id from `userpdt_pickup` where id=%d);", id)
	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询自提订单下产品编号失败:[%v]", err)
		this.Rec = &Recv{5, "查询自提订单下产品编号失败", nil}
		return
	}

	this.Rec = &Recv{3, ps("查询自提订单下产品编号成功"), result}
	return
}

// sid,id(自提查询返回的订单id),shipment_num,tpc_id(物流公司id)
func (this *SvrpvdController) SvrpvdPrdship() {
	// 身份判断
	if this.User.Flag != 6 {
		this.Rec = &Recv{5, ps("此接口只有运营商可以访问"), nil}
		return
	}

	id, _ := this.GetInt64("id")
	shipment_num := this.GetString("shipment_num")
	tpc_id, _ := this.GetInt64("tpc_id")

	if !CheckArg(id, shipment_num, tpc_id) {
		this.Rec = &Recv{5, ps("id和和物流信息不能为空"), nil}
		return
	}

	sql := ps("update `userpdt_pickup` set `status`=1,shipment_num='%s',tpc_id='%d' where id=%d;", shipment_num, tpc_id, id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("自提添加物流信息失败:[%v]", err)
		this.Rec = &Recv{5, "添加物流单号失败", nil}
		return
	}

	// 添加通知
	var result []orm.Params
	cnts, err := db.Raw("select uid from `userpdt_pickup` where id=?;", id).Values(&result)
	if err != nil {
		log("查询自提单用户id失败:[%v]", err)
	} else {
		if cnts > 0 {
			str := ps("你的编号为[%d]的自提产品已发货,你可以随时查看物流动态.", id)
			sql = ps("insert into `letter` (title,content,send_uid,recv_uid,unix) values ('%s','%s','0','%v','%d')",
				"通知", str, result[0]["uid"].(string), TimeNow)
			_, err = db.Raw(sql).Exec()
			if err != nil {
				log("添加通知失败:[%v]", err)
			}
		}
	}

	this.Rec = &Recv{3, ps("添加物流单号成功"), nil}
	return
}

// sid,idstr(订单id,多个之间逗号分隔,最后一个不后不加)
func (this *SvrpvdController) SvrpvdPrdReceiprt() {
	// 身份判断
	if this.User.Flag != 6 {
		this.Rec = &Recv{5, ps("此接口只有运营商可以访问"), nil}
		return
	}

	idstr := this.GetString("idstr")

	if !CheckArg(idstr) {
		this.Rec = &Recv{5, ps("id不能为空"), nil}
		return
	}

	sql := ps("update `user_product` set `status`=2 where id in (%s) and `status`=1 and `hosted_mid`=1;", idstr)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("更新产品状态失败:[%v]", err)
		this.Rec = &Recv{5, "确认收到厂商发货失败", nil}
		return
	}

	// 处理订单
	sql = "SELECT sum(up.`status`<4 AND up.`status`>-1) as cnts,ep.id,ep.user_id FROM user_product as up,enjoy_product as ep where up.ep_id=ep.id GROUP BY ep.id;"
	var result []orm.Params
	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询托管产品列表失败:[%v]", err)
	} else {
		for i := range result {
			cnts, _ := strconv.Atoi(result[i]["cnts"].(string))
			if cnts == 0 {
				ep_id, _ := strconv.Atoi(result[i]["id"].(string))
				sql = ps("UPDATE `enjoy_product` set pay_status='1',status=4 where `id`='%d';", ep_id)

				// 添加物流信息
				AddLogisticsInfo(0, int64(ep_id), "运营公司已收货...")
				AddLogisticsInfo(0, int64(ep_id), "等待投入运营...")

				_, err = db.Raw(sql).Exec()
				if err != nil {
					log("确认订单已收货失败:[%v]", err)
				} else {
					// 添加通知消息
					user_id, _ := strconv.Atoi(result[i]["user_id"].(string))
					str := ps("你的编号为[%d]的托管订单已完成,请前去查看", ep_id)
					sql = ps("insert into `letter` (title,content,send_uid,recv_uid,unix) values ('%s','%s','0','%d','%d')",
						"通知", str, user_id, TimeNow)
					_, err = db.Raw(sql).Exec()
					if err != nil {
						log("添加通知失败:[%v]", err)
					}
				}
			}
		}
	}

	this.Rec = &Recv{3, ps("确认收到厂商发货成功"), nil}
	return
}
