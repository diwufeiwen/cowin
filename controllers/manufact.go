package controllers

import (
	"github.com/astaxie/beego/orm"
	"strconv"
)

type ManufactController struct {
	OnlineController
}

// sid,pt_id,account,name,principal,phone,address,pwd
func (this *ManufactController) ManufactAdd() {
	// 身份判断
	if this.User.Flag != 1 {
		this.Rec = &Recv{5, ps("仅平台管理员可添加厂商"), nil}
		return
	}

	pt_id, _ := this.GetInt32("pt_id")
	account := this.GetString("account")
	name := this.GetString("name")
	principal := this.GetString("principal")
	phone := this.GetString("phone")
	address := this.GetString("address")
	pwd := this.GetString("pwd")

	//检查参数
	if !CheckArg(pt_id, account, name, principal, phone, address, pwd) {
		this.Rec = &Recv{5, "参数存在空值", nil}
		return
	}

	var sql string = ps("insert into `user` (pt_id,account,nick,realname,`phone`,address,pwd,flag,unix) values ('%d','%s','%s','%s','%s','%s','%s','5','%d');",
		pt_id, account, name, principal, phone, address, StrToMD5(ps("%s_Cowin_%s", account, pwd)), TimeNow)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("添加厂商失败:[%v]", err)
		this.Rec = &Recv{5, "添加厂商失败", nil}
		return
	}
	this.Rec = &Recv{3, "添加厂商成功", nil}
}

// sid
func (this *ManufactController) ManufactQuery() {
	sql := "select * from `user` where flag=5;"
	var result []orm.Params
	db := orm.NewOrm()
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询厂商失败:err[%v]", err)
		this.Rec = &Recv{5, ps("查询厂商失败"), nil}
		return
	}

	this.Rec = &Recv{3, ps("查询厂商成功!"), result}
}

// sid,id,name,principal,phone,address,pwd
func (this *ManufactController) ManufactModify() {
	id, _ := this.GetInt("id")
	name := this.GetString("name")
	principal := this.GetString("principal")
	phone := this.GetString("phone")
	address := this.GetString("address")
	pwd := this.GetString("pwd")

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
		log("修改厂商信息失败:[%v]", err)
		this.Rec = &Recv{5, "修改厂商信息失败", nil}
		return
	}
	this.Rec = &Recv{3, "修改厂商信息成功", nil}
}

// sid,id
func (this *ManufactController) ManufactDel() {
	id, _ := this.GetInt64("id")

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	// 业务逻辑
	var sql = ps("delete from `user` where id='%d' and flag=5;", id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("删除厂商失败:[%v]", err)
		this.Rec = &Recv{5, "删除厂商失败", nil}
		return
	}
	this.Rec = &Recv{3, "删除厂商成功", nil}
	return
}

// sid
func (this *ManufactController) ManufactApplyQuery() {
	sql := "select m.*,u.account,u.pt_id from manufacturer as m,user as u where m.offical=0 and m.uid=u.id;"
	var result []orm.Params
	db := orm.NewOrm()
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询厂商申请失败:err[%v]", err)
		this.Rec = &Recv{5, ps("查询厂商申请失败"), nil}
		return
	}

	this.Rec = &Recv{3, ps("查询厂商申请成功!"), result}
	return
}

// sid,id,agree(1-不通过,2-通过)
func (this *ManufactController) ManufactApplyDeal() {
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
		sql = ps("select * from manufacturer where id=%d and offical=0;", id)
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
				sql += ps("unix='%d' where id='%s';", TimeNow, result[0]["uid"].(string))

				_, err = db.Raw(sql).Exec()
				//log("%s", sql)
				if err != nil {
					log("更新申请厂商信息出错:[%v]", err)
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
	sql = ps("update manufacturer set offical=%d where id=%d;", agree, id)
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("删除申请信息出错:[%v]", err)
		this.Rec = &Recv{5, "申请处理成功,删除申请信息出错", nil}
		return
	}

	this.Rec = &Recv{3, "申请处理成功", nil}
	return
}

// sid,name,principal,phone,address
func (this *ManufactController) ManufactChangeApply() {
	name := this.GetString("name")
	principal := this.GetString("principal")
	phone := this.GetString("phone")
	address := this.GetString("address")

	// 业务逻辑
	sql := ps("select id from `manufacturer` where offical=0 and uid=%d;", this.User.UserId)
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
			sql = ps("update `manufacturer` set ")
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
			sql += ps(",unix='%d' where id='%d';", TimeNow, id)
		} else {
			sql = ps("insert into `manufacturer` (uid,name,principal,phone,address,unix) values('%d','%s','%s','%s','%s','%d');", this.User.UserId, name, principal, phone, address, TimeNow)
		}
	}

	// log("%s", sql)
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
func (this *ManufactController) ManufactChangeQuery() {
	sql := "select * from `manufacturer` where offical=0 order by unix desc;"
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

// sid,begidx,counts
func (this *ManufactController) ManufactOrderUndone() {
	// 身份判断
	if this.User.Flag != 5 {
		this.Rec = &Recv{5, ps("此接口只有厂商可以访问"), nil}
		return
	}

	begidx, _ := this.GetInt64("begidx")
	counts, _ := this.GetInt64("counts")

	// 检查参数
	if !CheckArg(counts) {
		this.Rec = &Recv{5, "产品id和请求总数不能为空", nil}
		return
	}

	// 定义数据结构
	type TagProduct struct {
		Product interface{}
		Order   interface{}
	}
	type RecvEx struct {
		Total  int64
		Detail []*TagProduct
	}

	// 查询所有商品名称信息
	sql := ps("select id from `product` where pt_id=%d;", this.User.Ptid)
	var result []orm.Params
	db := orm.NewOrm()
	nums, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询产品总数失败:[%v]", err)
		this.Rec = &Recv{5, ps("查询产品总数失败"), nil}
		return
	} else if nums <= 0 {
		this.Rec = &Recv{5, ps("该厂商旗下无产品"), nil}
		return
	}

	var data RecvEx
	data.Total = nums

	sql = ps("select id,product_name,start_date,end_date,coverurl from `product` where pt_id=%d limit %d,%d;", this.User.Ptid, begidx, counts)
	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询产品失败:[%v]", err)
		this.Rec = &Recv{5, ps("查询产品失败"), nil}
		return
	}

	data.Detail = make([]*TagProduct, len(result))
	// 循环查询该商品下所有订单和预售,已售信息
	for idx := range result {
		item := result[idx]
		// 查询预售信息
		sql = ps("select city,num from `product_city` where pid='%s';", item["id"].(string))
		var res []orm.Params
		_, err = db.Raw(sql).Values(&res)
		if err != nil {
			log("查询产品预售信息失败:[%v]", err)
			this.Rec = &Recv{5, ps("查询产品预售信息失败"), nil}
			return
		} else {
			str := ""
			for _, itcity := range res {
				str += itcity["city"].(string) + ":" + itcity["num"].(string) + ";"
			}
			item["presale"] = str
		}

		// 查询已售信息
		sql = ps("select sum(order_quantity) as quantity from `enjoy_product` where pid='%s';", item["id"].(string))
		_, err = db.Raw(sql).Values(&res)
		if err != nil {
			log("查询产品已售总数失败:[%v]", err)
			this.Rec = &Recv{5, ps("查询产品已售总数失败"), nil}
			return
		} else {
			if res[0]["quantity"] != nil {
				item["quantity"] = res[0]["quantity"]
			} else {
				item["quantity"] = "0"
			}
		}

		// 查询自提总数
		sql = ps("select  + sum(order_quantity) as ztquantity from `enjoy_product` where pid='%s' and hosted_mid=2;", item["id"].(string))
		_, err = db.Raw(sql).Values(&res)
		if err != nil {
			log("查询产品自提总数失败:[%v]", err)
			this.Rec = &Recv{5, ps("查询产品自提总数失败"), nil}
			return
		} else {
			if res[0]["ztquantity"] != nil {
				item["ztquantity"] = res[0]["ztquantity"]
			} else {
				item["ztquantity"] = "0"
			}
		}

		// 查询托管信息
		sql = ps("select sum(order_quantity) as num,hosted_city as tgquantity from `enjoy_product` where pid='%s' and hosted_mid=1 group by hosted_city;", item["id"].(string))
		_, err = db.Raw(sql).Values(&res)
		if err != nil {
			log("查询产品托管信息失败:[%v]", err)
			this.Rec = &Recv{5, ps("查询产品托管信息失败"), nil}
			return
		} else {
			str := ""
			for _, itcity := range res {
				str += itcity["tgquantity"].(string) + ":" + itcity["num"].(string) + ";"
			}
			item["tginfo"] = str
		}

		// 查询产品下所有订单
		sql = ps("select ep.id,ep.pt_id,ep.order_no,u.realname,u.account,ep.hosted_mid,ep.hosted_city,ep.order_quantity,ep.recver,ep.phone,ep.address,ep.shipment_num,ep.status,tc.code,tc.name from `enjoy_product` as ep,`user` as u,`transport_company` as tc where ep.tpc_id=tc.id and ep.user_id=u.id and ep.pid='%s' and ep.status<4;", item["id"].(string))
		_, err = db.Raw(sql).Values(&res)
		if err != nil {
			log("查询订单下产品失败:[%v]", err)
			this.Rec = &Recv{5, ps("查询订单下产品失败"), nil}
			return
		}

		// 托管订单查询厂商信息
		for i := range res {
			item := res[i]
			hosted_mid, _ := strconv.Atoi(item["hosted_mid"].(string))
			if hosted_mid == 1 {
				var restmp []orm.Params
				pt_id, _ := strconv.Atoi(item["pt_id"].(string))
				cnts, err := db.Raw("select realname,address,phone from `user` where flag=6 and pt_id=?;", pt_id).Values(&restmp)
				if err == nil && cnts > 0 {
					item["y_realname"] = restmp[0]["realname"]
					item["y_address"] = restmp[0]["address"]
					item["y_phone"] = restmp[0]["phone"]
				}
			}
		}
		data.Detail[idx] = &TagProduct{item, res}
	}

	this.Rec = &Recv{3, ps("查询未完成订单成功!"), &data}
	return
}

// sid,begidx,counts
func (this *ManufactController) ManufactOrderHistory() {
	// 身份判断
	if this.User.Flag != 5 {
		this.Rec = &Recv{5, ps("此接口只有厂商可以访问"), nil}
		return
	}

	begidx, _ := this.GetInt64("begidx")
	counts, _ := this.GetInt64("counts")

	// 检查参数
	if !CheckArg(counts) {
		this.Rec = &Recv{5, "产品id和请求总数不能为空", nil}
		return
	}

	// 定义数据结构
	type TagProduct struct {
		Product interface{}
		Order   interface{}
	}
	type RecvEx struct {
		Total  int64
		Detail []*TagProduct
	}

	// 查询所有商品名称信息
	sql := ps("select id from `product` where pt_id=%d;", this.User.Ptid)
	var result []orm.Params
	db := orm.NewOrm()
	nums, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询产品总数失败:[%v]", err)
		this.Rec = &Recv{5, ps("查询产品总数失败"), nil}
		return
	} else if nums <= 0 {
		this.Rec = &Recv{5, ps("该厂商旗下无产品"), nil}
		return
	}

	var data RecvEx
	data.Total = nums

	sql = ps("select id,product_name,start_date,end_date from `product` where pt_id=%d limit %d,%d;", this.User.Ptid, begidx, counts)
	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询产品失败:[%v]", err)
		this.Rec = &Recv{5, ps("查询产品失败"), nil}
		return
	}

	data.Detail = make([]*TagProduct, len(result))
	// 循环查询该商品下所有订单和预售,已售信息
	for idx := range result {
		item := result[idx]
		// 查询预售信息
		sql = ps("select city,num from `product_city` where pid='%s';", item["id"].(string))
		var res []orm.Params
		_, err = db.Raw(sql).Values(&res)
		if err != nil {
			log("查询产品预售信息失败:[%v]", err)
			this.Rec = &Recv{5, ps("查询产品预售信息失败"), nil}
			return
		} else {
			str := ""
			for _, itcity := range res {
				str += itcity["city"].(string) + ":" + itcity["num"].(string) + ";"
			}
			item["presale"] = str
		}

		// 查询已售信息
		sql = ps("select sum(order_quantity) as quantity from `enjoy_product` where pid='%s';", item["id"].(string))
		_, err = db.Raw(sql).Values(&res)
		if err != nil {
			log("查询产品已售总数失败:[%v]", err)
			this.Rec = &Recv{5, ps("查询产品已售总数失败"), nil}
			return
		} else {
			if res[0]["quantity"] != nil {
				item["quantity"] = res[0]["quantity"]
			} else {
				item["quantity"] = "0"
			}
		}

		// 查询自提总数
		sql = ps("select sum(order_quantity) as ztquantity from `enjoy_product` where pid='%s' and hosted_mid=2;", item["id"].(string))
		_, err = db.Raw(sql).Values(&res)
		if err != nil {
			log("查询产品自提总数失败:[%v]", err)
			this.Rec = &Recv{5, ps("查询产品自提总数失败"), nil}
			return
		} else {
			if res[0]["ztquantity"] != nil {
				item["ztquantity"] = res[0]["ztquantity"]
			} else {
				item["ztquantity"] = "0"
			}
		}

		// 查询托管信息
		sql = ps("select sum(order_quantity) as num,hosted_city as tgquantity from `enjoy_product` where pid='%s' and hosted_mid=1 group by hosted_city;", item["id"].(string))
		_, err = db.Raw(sql).Values(&res)
		if err != nil {
			log("查询产品托管信息失败:[%v]", err)
			this.Rec = &Recv{5, ps("查询产品托管信息失败"), nil}
			return
		} else {
			str := ""
			for _, itcity := range res {
				str += itcity["tgquantity"].(string) + ":" + itcity["num"].(string) + ";"
			}
			item["tginfo"] = str
		}

		// 查询产品下所有订单
		sql = ps("select ep.id,ep.order_no,u.realname,u.account,ep.hosted_mid,ep.hosted_city,ep.order_quantity,ep.recver,ep.phone,ep.address,ep.shipment_num,ep.status,tc.code,tc.name from `enjoy_product` as ep,`user` as u,`transport_company` as tc where ep.tpc_id=tc.id and ep.user_id=u.id and ep.pid='%s' and ep.status=4;", item["id"].(string))
		_, err = db.Raw(sql).Values(&res)
		if err != nil {
			log("查询订单下产品失败:[%v]", err)
			this.Rec = &Recv{5, ps("查询订单下产品失败"), nil}
			return
		}
		data.Detail[idx] = &TagProduct{item, res}
	}

	this.Rec = &Recv{3, ps("查询历史订单成功"), &data}
	return
}

// sid,id(不传表示全部开始生产)
func (this *ManufactController) ManufactOrderProducting() {
	// 身份判断
	if this.User.Flag != 5 {
		this.Rec = &Recv{5, ps("此接口只有厂商可以访问"), nil}
		return
	}

	id, _ := this.GetInt64("id")
	pid, _ := this.GetInt64("pid")

	if !CheckArg(pid) {
		this.Rec = &Recv{5, ps("产品编号不能为空"), nil}
		return
	}

	var sql string
	if id > 0 {
		sql = ps("update `enjoy_product` set status=1 where id=%d and status=0 and pid=%d", id, pid)
	} else {
		sql = ps("update `enjoy_product` set status=1 where status=0 and pid=%d", pid)
	}

	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("更新状态失败:[%v]", err)
		this.Rec = &Recv{5, ps("设置订单生产中状态失败"), nil}
		return
	}

	// 添加物流信息
	AddLogisticsInfo(0, id, "厂商已接单,生产中...")
	this.Rec = &Recv{3, ps("设置订单生产中成功"), nil}
	return
}

// sid,id(不传表示全部生产完成)
func (this *ManufactController) ManufactOrderProducted() {
	// 身份判断
	if this.User.Flag != 5 {
		this.Rec = &Recv{5, ps("此接口只有厂商可以访问"), nil}
		return
	}

	id, _ := this.GetInt64("id")
	pid, _ := this.GetInt64("pid")

	if !CheckArg(pid) {
		this.Rec = &Recv{5, ps("产品编号不能为空"), nil}
		return
	}

	var sql string
	if id > 0 {
		sql = ps("update `enjoy_product` set status=2 where id=%d and status=1 and pid=%d", id, pid)
	} else {
		sql = ps("update `enjoy_product` set status=2 where status=1 and pid=%d", pid)
	}

	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("更新状态失败:[%v]", err)
		this.Rec = &Recv{5, ps("设置订单生产完成状态失败"), nil}
		return
	}
	// 添加物流信息
	AddLogisticsInfo(0, id, "生产已完成,打包发货中...")
	this.Rec = &Recv{3, ps("设置订单生产完成状态成功"), nil}
	return
}

// sid,id,shipment_num(物流单号),tpc_id(物流公司id)
func (this *ManufactController) ManufactShipnum() {
	// 身份判断
	if this.User.Flag != 5 {
		this.Rec = &Recv{5, ps("此接口只有厂商可以访问"), nil}
		return
	}

	id, _ := this.GetInt64("id")
	shipment_num := this.GetString("shipment_num")
	tpc_id, _ := this.GetInt32("tpc_id")

	if !CheckArg(id, shipment_num) {
		this.Rec = &Recv{5, ps("订单编号和物流信息不能为空"), nil}
		return
	}

	var sql string = ps("update `enjoy_product` set status=3,shipment_num='%s',tpc_id='%d' where id=%d", shipment_num, tpc_id, id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("添加物流单号失败:[%v]", err)
		this.Rec = &Recv{5, ps("添加物流单号失败"), nil}
		return
	}

	// 添加通知消息
	sql = ps("SELECT user_id from `enjoy_product` where id=%d;", id)
	var result []orm.Params
	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询产品用户id失败:[%v]", err)
	} else {
		user_id, _ := strconv.Atoi(result[0]["user_id"].(string))
		str := ps("你的编号为[%d]的订单已发货,你可以随时关注物流状态", id)
		sql = ps("insert into `letter` (title,content,send_uid,recv_uid,unix) values ('%s','%s','0','%d','%d')",
			"通知", str, user_id, TimeNow)
		_, err = db.Raw(sql).Exec()
		if err != nil {
			log("添加通知失败:[%v]", err)
		}
	}

	sql = ps("update `user_product` set `status`=1 where ep_id=%d", id)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("更新订单下产品已发货状态失败:[%v]", err)
		this.Rec = &Recv{5, ps("更新订单下产品已发货状态失败"), nil}
		return
	}

	this.Rec = &Recv{3, ps("添加物流单号成功"), nil}
	return
}

// sid,id
func (this *ManufactController) ManufactProductno() {
	// 身份判断
	// if this.User.Flag != 5 {
	// 	this.Rec = &Recv{5, ps("此接口只有厂商可以访问"), nil}
	// 	return
	// }

	id, _ := this.GetInt64("id")

	if !CheckArg(id) {
		this.Rec = &Recv{5, ps("订单编号不能为空"), nil}
		return
	}

	var sql string = ps("select product_no from  `user_product` where ep_id=%d", id)

	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询失败:[%v]", err)
		this.Rec = &Recv{5, ps("查询订单下产品编号失败"), nil}
		return
	}

	this.Rec = &Recv{3, ps("查询订单下产品编号成功"), result}
	return
}
