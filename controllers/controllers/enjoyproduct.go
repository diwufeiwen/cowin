package controllers

import (
	"github.com/astaxie/beego/orm"
	"strconv"
	"strings"
	"time"
)

type EnjoyproductController struct {
	OnlineController
}

type EnjoyBaseController struct {
	BaseController
}

// sid
func (this *EnjoyBaseController) HostMethodQuery() {
	sql := "SELECT * from `host_method`;"

	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询订购方式出错:[%v]", err)
		this.Rec = &Recv{5, "查询订购方式失败", nil}
		return
	}

	this.Rec = &Recv{3, "查询订购方式成功", result}
	return
}

// sid,pid,pt_id,order_quantity,hosted_city(托管城市),style,hosted_mid(1-托管,2-自提),recver,address(收货地址),phone,invoice_type(发票类型:0-个人,1-公司),invoice_head(发票抬头),taxNum(税号)
func (this *EnjoyproductController) ProductBuy() {
	pid, _ := this.GetInt32("pid")
	pt_id, _ := this.GetInt32("pt_id")
	order_quantity, _ := this.GetInt("order_quantity")
	hosted_city := this.GetString("hosted_city")
	style := this.GetString("style")
	hosted_mid, _ := this.GetInt("hosted_mid")
	recver := this.GetString("recver")
	address := this.GetString("address")
	phone := this.GetString("phone")
	invoice_type, _ := this.GetInt32("invoice_type")
	invoice_head := this.GetString("invoice_head")
	taxNum := this.GetString("taxNum")

	if !CheckArg(pid, pt_id) {
		this.Rec = &Recv{5, "产品id和类型id不能为空", nil}
		return
	}

	if !CheckArg(order_quantity, hosted_mid) {
		this.Rec = &Recv{5, "请上传购买数量和购买方式", nil}
		return
	}

	if hosted_mid == 2 && address == "" {
		this.Rec = &Recv{5, "请上传收货地址", nil}
		return
	}

	if hosted_mid == 1 && hosted_city == "" {
		this.Rec = &Recv{5, "请上传托管城市", nil}
		return
	}

	// 托管状态下查看是否托管已达上限
	var sql string = ""
	var result []orm.Params
	db := orm.NewOrm()
	if hosted_mid == 1 {
		sql := ps("select sum(order_quantity) as quantity from `enjoy_product` where pid=%d and `hosted_mid`=1 and `hosted_city` like '%%%s%%';", pid, hosted_city)
		_, err := db.Raw(sql).Values(&result)
		if err != nil {
			log("查询托管总数失败:[%v]", err)
			this.Rec = &Recv{5, "查询托管总数失败", nil}
			return
		} else {
			totals := 0
			if result[0]["quantity"] != nil {
				totals, _ = strconv.Atoi(result[0]["quantity"].(string)) //已投放数量
			}
			sql = ps("select num from `product_city` where pid='%d' and city like '%%%s%%';", pid, hosted_city)
			_, err := db.Raw(sql).Values(&result)
			if err == nil {
				presale, _ := strconv.Atoi(result[0]["num"].(string))
				if totals+order_quantity > presale {
					this.Rec = &Recv{5, "该城市投放数量将超上限,无法购买,请减少数量或变更城市.", ps("当前数量:%d", totals)}
					return
				}
			} else {
				log("查询产品预托管总数失败:[%v]", err)
				this.Rec = &Recv{5, "查询产品预托管总数失败", nil}
				return
			}
		}
	}

	// 如果是余额支付,检测余额是否足够
	order_no := ps("%s_%s_%s", this.User.DealerAcc, time.Now().Format("20060102150405"), GetRandomString(3))
	sql = ps("insert into `enjoy_product` (pt_id,pid,user_id,order_no,order_quantity,`style`,hosted_mid,recver,address,phone,hosted_city,invoice_type,invoice_head,taxNum,pay_deadline,unix) values ('%d','%d','%d','%s','%d','%s','%d','%s','%s','%s','%s','%d','%s','%s','%d','%d');",
		pt_id, pid, this.User.UserId, order_no, order_quantity, style, hosted_mid, recver, address, phone, hosted_city, invoice_type, invoice_head, taxNum, TimeNow+30*60, TimeNow)
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("订购失败:err[%v]", err)
		this.Rec = &Recv{5, ps("[%s]订购失败", this.User.Account), nil}
		return
	}

	// 查询订单信息
	sql = ps("select * from `enjoy_product` where order_no='%s';", order_no)
	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询最新订单失败:%s", err.Error())
		this.Rec = &Recv{5, "查询最新订单失败", nil}
		return
	}

	this.Rec = &Recv{3, "订购成功,请前去支付!", result}
	return
}

// sid,pay_status(0-未支付,1-已支付),status(1-进行中,2-已完成),begidx,counts
func (this *EnjoyproductController) ProductOrderQuery() {
	pay_status, _ := this.GetInt32("pay_status")
	status, _ := this.GetInt32("status")
	begidx, _ := this.GetInt64("begidx")
	counts, _ := this.GetInt32("counts")

	if pay_status < 0 {
		this.Rec = &Recv{5, "支付状态不能为空", nil}
		return
	}

	if !CheckArg(counts) {
		this.Rec = &Recv{5, "总数不能为空", nil}
		return
	}

	// 业务逻辑
	var sql, sqlc string
	switch status {
	case 0:
		sql = ps("SELECT ep.*,p.product_name,p.discount_price,p.imgurl,p.web_intro,p.app_intro,tc.code,tc.name from `enjoy_product` as ep,`product` as p,`transport_company` as tc where p.id=ep.pid and ep.tpc_id=tc.id and ep.user_id='%d' and ep.pay_status=%d order by ep.unix desc limit %d,%d;",
			this.User.UserId, pay_status, begidx, counts)
		sqlc = ps("SELECT count(id) as total from `enjoy_product` where user_id='%d' and pay_status=%d;", this.User.UserId, pay_status)
	case 1:
		sql = ps("SELECT ep.*,p.product_name,p.discount_price,p.imgurl,p.web_intro,p.app_intro,tc.code,tc.name from `enjoy_product` as ep,`product` as p,`transport_company` as tc where p.id=ep.pid and ep.tpc_id=tc.id and ep.user_id='%d' and ep.pay_status=%d and ep.status<4 order by ep.unix desc limit %d,%d;",
			this.User.UserId, pay_status, begidx, counts)
		sqlc = ps("SELECT count(id) as total from `enjoy_product` where user_id='%d' and pay_status=%d and status<4;", this.User.UserId, pay_status)
	case 2:
		sql = ps("SELECT ep.*,p.product_name,p.discount_price,p.imgurl,p.web_intro,p.app_intro,tc.code,tc.name from `enjoy_product` as ep,`product` as p,`transport_company` as tc where p.id=ep.pid and ep.tpc_id=tc.id and ep.user_id='%d' and ep.pay_status=%d and ep.status=4 order by ep.unix desc limit %d,%d;",
			this.User.UserId, pay_status, begidx, counts)
		sqlc = ps("SELECT count(id) as total from `enjoy_product` where user_id='%d' and pay_status=%d and status=4;", this.User.UserId, pay_status)
	}
	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sqlc).Values(&result)
	if err != nil {
		log("查询订单总数出错:[%v]", err)
		this.Rec = &Recv{5, "查询订单总数失败", nil}
		return
	}
	total, _ := strconv.Atoi(result[0]["total"].(string))

	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询订单出错:[%v]", err)
		this.Rec = &Recv{5, "查询订单失败", nil}
		return
	}

	type RecvEx struct {
		Total  int
		Detail interface{}
	}
	this.Rec = &Recv{3, "查询订单成功", &RecvEx{total, result}}
	return
}

// sid,status(1-生产中(0,1),2-提货中(2,3)),begidx,counts,city(托管城市),hosted_mid,pt_id
func (this *EnjoyproductController) UserProductOrderQuery() {
	status, _ := this.GetInt32("status")
	begidx, _ := this.GetInt64("begidx")
	counts, _ := this.GetInt32("counts")
	city := this.GetString("city")
	hosted_mid, _ := this.GetInt32("hosted_mid")
	pt_id, _ := this.GetInt32("pt_id")

	if !CheckArg(counts) {
		this.Rec = &Recv{5, "总数不能为空", nil}
		return
	}

	// 业务逻辑
	var sql, sqlc string
	switch status {
	case 1:
		sql = ps("SELECT ep.*,p.product_name,p.discount_price,p.imgurl,p.web_intro,p.app_intro,tc.code,tc.name from `enjoy_product` as ep,`product` as p,`transport_company` as tc where ep.tpc_id=tc.id and p.id=ep.pid and ep.user_id='%d' and ep.pt_id=%d and ep.status<2",
			this.User.UserId, pt_id)
		sqlc = ps("SELECT id from `enjoy_product` where user_id='%d' and pt_id=%d and status<2", this.User.UserId, pt_id)
	case 2:
		sql = ps("SELECT ep.*,p.product_name,p.discount_price,p.imgurl,p.web_intro,p.app_intro,tc.code,tc.name from `enjoy_product` as ep,`product` as p,`transport_company` as tc where ep.tpc_id=tc.id and p.id=ep.pid and ep.user_id='%d' and ep.pt_id=%d and ep.status>1 and ep.status<4",
			this.User.UserId, pt_id)
		sqlc = ps("SELECT id from `enjoy_product` where user_id='%d' and pt_id=%d and status>1 and status<4", this.User.UserId, pt_id)
	}

	if city != "" {
		sql += ps(" and ep.hosted_city like '%%%s%%'", city)
	}

	if hosted_mid > 0 {
		sql += ps(" and ep.hosted_mid=%d", hosted_mid)
	}
	sql += ps(" limit %d,%d;", begidx, counts)
	log("%s", sql)
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
		this.Rec = &Recv{3, "查询订单总数成功", &RecvEx{nums, nil}}
		return
	}

	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询订单出错:[%v]", err)
		this.Rec = &Recv{5, "查询订单失败", nil}
		return
	}

	this.Rec = &Recv{3, "查询订单成功", &RecvEx{nums, result}}
	return
}

// sid,id(订单id)
func (this *EnjoyproductController) ProductOrderReceipt() {
	id, _ := this.GetInt32("id")

	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	sql := ps("UPDATE `enjoy_product` set status=4 where id=%d and user_id=%d;", id, this.User.UserId)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("更新订单状态失败:[%v]", err)
		this.Rec = &Recv{5, "订单确认收货失败", nil}
		return
	}

	this.Rec = &Recv{3, "订单确认收货成功", nil}
}

// sid,order_no(订单号),phone(购买人手机号),status(1-未完成,2-已完成),begidx,counts
func (this *EnjoyproductController) ProductOrderSearch() {
	status, _ := this.GetInt("status")
	order_no := this.GetString("order_no")
	phone := this.GetString("phone")
	begidx, _ := this.GetInt64("begidx")
	counts, _ := this.GetInt32("counts")

	if !CheckArg(counts) {
		this.Rec = &Recv{5, "总数不能为空", nil}
		return
	}

	var sql, sqlc string
	if CheckArg(phone) {
		if CheckArg(order_no) { //订单号不为空
			switch status {
			case 0:
				sql = ps("SELECT ep.*,p.product_name,p.discount_price,p.start_date,p.end_date,p.imgurl from `enjoy_product` as ep,`product` as p,`user` as u where p.id=ep.pid and ep.user_id=u.id and u.account='%s' and ep.order_no like '%%%s%%' limit %d,%d;",
					phone, order_no, begidx, counts)
				sqlc = ps("SELECT ep.id from `enjoy_product` as ep,`user` as u where ep.order_no like '%%%s%%' and ep.user_id=u.id and u.account='%s';", order_no, phone)
			case 1:
				sql = ps("SELECT ep.*,p.product_name,p.discount_price,p.start_date,p.end_date,p.imgurl from `enjoy_product` as ep,`product` as p,`user` as u where p.id=ep.pid and ep.user_id=u.id and u.account='%s' and ep.order_no like '%%%s%%' and ep.status<4 limit %d,%d;",
					phone, order_no, begidx, counts)
				sqlc = ps("SELECT ep.id from `enjoy_product` as ep,`user` as u where ep.order_no like '%%%s%%' and ep.status<4 and ep.user_id=u.id and u.account='%s';", order_no, phone)
			case 2:
				sql = ps("SELECT ep.*,p.product_name,p.discount_price,p.start_date,p.end_date,p.imgurl from `enjoy_product` as ep,`product` as p,`user` as u where p.id=ep.pid and ep.user_id=u.id and u.account='%s' and ep.order_no like '%%%s%%' and ep.status=4 limit %d,%d;",
					phone, order_no, begidx, counts)
				sqlc = ps("SELECT ep.id from `enjoy_product` as ep,`user` as u where ep.order_no like '%%%s%%' and ep.status=4 and ep.user_id=u.id and u.account='%s';", order_no, phone)
			}
		} else {
			switch status {
			case 0:
				sql = ps("SELECT ep.*,p.product_name,p.discount_price,p.start_date,p.end_date,p.imgurl from `enjoy_product` as ep,`product` as p,`user` as u where p.id=ep.pid and ep.user_id=u.id and u.account='%s' limit %d,%d;", phone, begidx, counts)
				sqlc = ps("SELECT ep.id from `enjoy_product` as ep,`user` as u where ep.user_id=u.id and u.account='%s';", phone)
			case 1:
				sql = ps("SELECT ep.*,p.product_name,p.discount_price,p.start_date,p.end_date,p.imgurl from `enjoy_product` as ep,`product` as p,`user` as u where p.id=ep.pid and ep.user_id=u.id and u.account='%s' and ep.status<4 limit %d,%d;", phone, begidx, counts)
				sqlc = ps("SELECT ep.id from `enjoy_product` as ep,`user` as u where ep.user_id=u.id and u.account='%s' and ep.status<4;", phone)
			case 2:
				sql = ps("SELECT ep.*,p.product_name,p.discount_price,p.start_date,p.end_date,p.imgurl from `enjoy_product` as ep,`product` as p,`user` as u where p.id=ep.pid and ep.user_id=u.id and u.account='%s' and ep.status=4 limit %d,%d;", phone, begidx, counts)
				sqlc = ps("SELECT ep.id from `enjoy_product` as ep,`user` as u where ep.user_id=u.id and u.account='%s' and ep.status=4;", phone)
			}
		}
	} else {
		if CheckArg(order_no) { //订单号不为空
			switch status {
			case 0:
				sql = ps("SELECT ep.*,p.product_name,p.discount_price,p.start_date,p.end_date,p.imgurl from `enjoy_product` as ep,`product` as p where p.id=ep.pid and ep.order_no like '%%%s%%' limit %d,%d;",
					order_no, begidx, counts)
				sqlc = ps("SELECT id from `enjoy_product` where order_no like '%%%s%%';", order_no)
			case 1:
				sql = ps("SELECT ep.*,p.product_name,p.discount_price,p.start_date,p.end_date,p.imgurl from `enjoy_product` as ep,`product` as p where p.id=ep.pid and ep.order_no like '%%%s%%' and ep.status<4 limit %d,%d;",
					order_no, begidx, counts)
				sqlc = ps("SELECT id from `enjoy_product` where order_no like '%%%s%%' and status<4;", order_no)
			case 2:
				sql = ps("SELECT ep.*,p.product_name,p.discount_price,p.start_date,p.end_date,p.imgurl from `enjoy_product` as ep,`product` as p where p.id=ep.pid and ep.order_no like '%%%s%%' and ep.status=4 limit %d,%d;",
					order_no, begidx, counts)
				sqlc = ps("SELECT id from `enjoy_product` where order_no like '%%%s%%' and status=4;", order_no)
			}
		} else {
			switch status {
			case 0:
				sql = ps("SELECT ep.*,p.product_name,p.discount_price,p.start_date,p.end_date,p.imgurl from `enjoy_product` as ep,`product` as p where p.id=ep.pid limit %d,%d;", begidx, counts)
				sqlc = "SELECT id from `enjoy_product`;"
			case 1:
				sql = ps("SELECT ep.*,p.product_name,p.discount_price,p.start_date,p.end_date,p.imgurl from `enjoy_product` as ep,`product` as p where p.id=ep.pid and ep.status<4 limit %d,%d;", begidx, counts)
				sqlc = "SELECT id from `enjoy_product` where status<4;"
			case 2:
				sql = ps("SELECT ep.*,p.product_name,p.discount_price,p.start_date,p.end_date,p.imgurl from `enjoy_product` as ep,`product` as p where p.id=ep.pid and ep.status=4 limit %d,%d;", begidx, counts)
				sqlc = "SELECT id from `enjoy_product` where status=4;"
			}
		}
	}

	db := orm.NewOrm()
	var result []orm.Params
	cnts, err := db.Raw(sqlc).Values(&result)
	if err != nil {
		log("搜索订单数出错:[%v]", err)
		this.Rec = &Recv{5, "搜索订单失败", nil}
		return
	}

	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("搜索订单出错:[%v]", err)
		this.Rec = &Recv{5, "搜索订单失败", nil}
		return
	}

	type RecvEx struct {
		Total  int64
		Detail interface{}
	}
	this.Rec = &Recv{3, "搜索订单成功", &RecvEx{cnts, result}}
	return
}

// sid,id(订单id)
func (this *EnjoyproductController) ProductOrderCancel() {
	id, _ := this.GetInt64("id")

	if !CheckArg("id") {
		this.Rec = &Recv{5, "订单id不能为空", nil}
		return
	}

	sql := ps("delete from `enjoy_product` where id=%d and user_id=%d and pay_status=0;", id, this.User.UserId)

	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("取消订单出错:[%v]", err)
		this.Rec = &Recv{5, "取消订单失败", nil}
		return
	}

	this.Rec = &Recv{3, "取消订单成功", result}
	return
}

// sid,idstr(多个id以分号分隔,最后一个后不加分号),pay_method(0-余额支付,1-其他方式支付)
func (this *EnjoyproductController) ProductOrderPay() {
	idstr := this.GetString("idstr")
	pay_method, _ := this.GetInt("pay_method")

	//log("%s", idstr)
	if !CheckArg(idstr) {
		this.Rec = &Recv{5, "订单id不能为空", nil}
		return
	}

	var sql string = ""
	db := orm.NewOrm()
	var result []orm.Params

	idarr := strings.Split(idstr, ";")
	//log("%v", idarr)
	for _, item := range idarr {
		id, _ := strconv.Atoi(item)

		if pay_method == 0 {
			sql = ps("select p.discount_price as price,ep.order_quantity from `product` as p,`enjoy_product` as ep where p.id=ep.pid and ep.id=%d;", id)
			//log("%s", sql)
			nums, err := db.Raw(sql).Values(&result)
			if err != nil {
				log("查询订单[%d]价格失败:[%v]", err)
				this.Rec = &Recv{5, ps("查询订单[%d]价格失败", id), nil}
				return
			} else {
				if nums > 0 {
					order_quantity, _ := strconv.Atoi(result[0]["order_quantity"].(string))
					price, _ := strconv.ParseFloat(result[0]["price"].(string), 64) //已投放数量
					if (price * float64(order_quantity)) > this.User.Wallet {
						this.Rec = &Recv{5, "账户余额不足,请充值.", nil}
						return
					} else {
						sql = ps("update `user` set wallet='%v' where id=%d;", this.User.Wallet-(price*float64(order_quantity)), this.User.UserId)
						_, err = db.Raw(sql).Values(&result)
						if err != nil {
							log("更新用户资金失败:[%v]", err)
							this.Rec = &Recv{5, "更新用户资金失败", nil}
							return
						}
						this.User.Wallet -= price * float64(order_quantity)
					}
				} else {
					log("订单[%d]下的产品不存在", id)
					continue
				}
			}
		}

		sql = ps("SELECT order_quantity,hosted_mid,user_id from `enjoy_product` where id='%d';", id)
		nums, err := db.Raw(sql).Values(&result)
		if err != nil {
			log("查询订单[%d]信息失败:[%v]", err)
			this.Rec = &Recv{5, ps("查询订单[%d]信息失败", id), nil}
			return
		}

		if nums <= 0 {
			this.Rec = &Recv{5, ps("订单[%d]不存在", id), nil}
			return
		}

		order_quantity, _ := strconv.Atoi(result[0]["order_quantity"].(string))
		hosted_mid, _ := strconv.Atoi(result[0]["hosted_mid"].(string))
		user_id, _ := strconv.Atoi(result[0]["user_id"].(string))
		code, strerr := GenerateUserProduct(int64(id), order_quantity, hosted_mid, int64(user_id))
		if code == 5 {
			this.Rec = &Recv{5, strerr, nil}
			return
		}

		sql = ps("update `enjoy_product` set pay_status=1 where id='%d';", id)
		_, err = db.Raw(sql).Exec()
		if err != nil {
			log("更新订单[%d]支付状态出错:[%v]", id, err)
			this.Rec = &Recv{5, ps("更新订单[%d]支付状态失败", id), nil}
			return
		}

		// 生成合同
		if hosted_mid == 1 {
			sql = ps("insert into `agreement` (ep_id,text,unix) values('%d','%s','%d');", id, "", TimeNow)
			_, err = db.Raw(sql).Exec()
			if err != nil {
				log("生成订单[%s]合同出错:[%v]", id, err)
				this.Rec = &Recv{5, ps("生成订单合同失败", id), nil}
				return
			}
		}
	}

	this.Rec = &Recv{3, "支付处理成功", nil}
}

// sid,idstr(订单id以逗号分隔)
func (this *EnjoyproductController) OrderAgreementSign() {
	idstr := this.GetString("idstr")

	if !CheckArg(idstr) {
		this.Rec = &Recv{5, "订单id不能为空", nil}
		return
	}

	var sqls []string
	db := orm.NewOrm()

	var result []orm.Params
	nums, err := db.Raw("select realname,idnumber,positive_img,negative_img from user where id=? and verify_status=3", this.User.UserId).Values(&result)
	if err != nil {
		log("查询用户认证信息失败:[%v]", err)
		this.Rec = &Recv{5, "查询用户认证信息失败,合同签署失败.", nil}
		return
	}
	verify_status := 3
	if nums <= 0 {
		verify_status = 0
	}

	idarr := strings.Split(idstr, ",")
	for _, item := range idarr {
		id, _ := strconv.Atoi(item)
		if verify_status == 3 {
			sqls = append(sqls, ps("update `agreement` set text='%s',realname='%s',idnumber='%s',positive_img='%s',negative_img='%s',unix='%d',status=2 where ep_id=%d;",
				"", result[0]["realname"].(string), result[0]["idnumber"].(string), result[0]["positive_img"].(string), result[0]["negative_img"].(string), TimeNow, id))
		} else {
			sqls = append(sqls, ps("update `agreement` set text='%s',realname='',idnumber='',positive_img='',negative_img='',unix='%d',status=1 where ep_id=%d;",
				"", TimeNow, id))
		}
	}

	db.Begin() //开启事务
	for _, ele := range sqls {
		_, err = db.Raw(ele).Exec()
		if err != nil {
			log("合同签署失败:%s", err.Error())
			this.Rec = &Recv{5, "合同签署失败.", nil}
			return
		}
	}
	db.Commit() //提交事务

	str := "合同签署成功"
	if verify_status == 0 {
		str += ",实名认证后生效"
	}
	this.Rec = &Recv{3, str, nil}
	return
}

// sid,status(0-未签署,1-已签署),ep_id
func (this *EnjoyproductController) OrderAgreementQuery() {
	status, _ := this.GetInt32("status")
	ep_id, _ := this.GetInt32("ep_id")

	db := orm.NewOrm()

	var result []orm.Params
	sql := ps("select a.* from `agreement` as a,`enjoy_product` as ep where a.ep_id=ep.id and ep.user_id=%d", this.User.UserId)
	if ep_id > 0 {
		sql += ps(" and a.ep_id=%d", ep_id)
	}
	if status >= 0 {
		sql += ps(" and a.status=%d", status)
	}
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询合同失败:[%v]", err)
		this.Rec = &Recv{5, "查询合同失败.", nil}
		return
	}

	this.Rec = &Recv{3, "查询合同成功.", result}
	return
}

// 订单正式转入用户账下产品(充电宝)
func GenerateUserProduct(id int64, order_quantity int, hosted_mid int, user_id int64) (code int64, strerr string) {
	var sqls []string
	strtm := time.Now().Format("20060102150405") //当前时间字符窜
	today := time.Now().Format("2006-01-02")
	today_t, _ := time.ParseInLocation("2006-01-02", today, time.Local)
	dt, _ := time.ParseDuration("24h")
	tomorrow_t := today_t.Add(dt).Unix() //明天
	// 获取当日产品数量
	sql := ps("SELECT count(up.id) as num from `user_product` as up,`enjoy_product` as ep where up.ep_id=ep.id and ep.pt_id=(SELECT pt_id FROM enjoy_product WHERE id=%d) and up.unix>=%d and up.unix<%d;",
		id, today_t.Unix(), tomorrow_t)
	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询当日产品数量出错:[%s]", sql)
		code = 5
		strerr = "查询当日产品数量失败"
		return
	}

	begidx, _ := strconv.Atoi(result[0]["num"].(string))
	for i := 0; i < order_quantity; i++ {
		sqls = append(sqls, ps("insert into `user_product` (ep_id,user_id,product_no,hosted_mid,unix) values ('%d','%d','%s','%d','%d');",
			id, user_id, ps("PB_%s_%06d", strtm, begidx+i+1), hosted_mid, TimeNow))
	}

	db.Begin() //开启事务
	for _, ele := range sqls {
		db.Raw(ele).Exec()
	}
	db.Commit() //提交事务

	code = 3
	return
}
