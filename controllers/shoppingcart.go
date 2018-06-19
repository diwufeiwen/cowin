package controllers

import (
	"github.com/astaxie/beego/orm"
	"strconv"
	"strings"
	"time"
)

type ShopcartController struct {
	OnlineController
}

// sid,pid,pt_id,order_quantity,style(样式:多个以分号分隔),hosted_mid(托管方式id),hosted_city(托管城市)
func (this *ShopcartController) ShopcartAdd() {
	pid, _ := this.GetInt32("pid")
	pt_id, _ := this.GetInt32("pt_id")
	order_quantity, _ := this.GetInt("order_quantity")
	style := this.GetString("style")
	hosted_mid, _ := this.GetInt32("hosted_mid")
	hosted_city := this.GetString("hosted_city")

	if !CheckArg(pid, pt_id) {
		this.Rec = &Recv{5, "产品id和类型id不能为空", nil}
		return
	}

	if !CheckArg(order_quantity, hosted_mid) {
		this.Rec = &Recv{5, "请上传购买数量和购买方式", nil}
		return
	}

	if hosted_mid == 1 && hosted_city == "" {
		this.Rec = &Recv{5, "请上传托管城市", nil}
		return
	}

	// 判断每个城市的购买数量
	var sql string = ""
	db := orm.NewOrm()
	var result []orm.Params
	sql = ps("select id,`style` from `shopping_cart` where pid=%d and user_id=%d and hosted_mid=%d and hosted_city like '%%%s%%';", pid, this.User.UserId, hosted_mid, hosted_city)
	nums, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询购物车失败:[%v]", err)
		this.Rec = &Recv{5, "查询购物车失败", nil}
		return
	}

	if nums > 0 {
		var style1 string
		if result[0]["style"] != nil {
			style1 = result[0]["style"].(string)
		}

		if !strings.Contains(style1, style) {
			style1 += style
		}
		id, _ := strconv.Atoi(result[0]["id"].(string))
		sql = ps("update `shopping_cart` set order_quantity=order_quantity+%d,`style`='%s' where id=%d;", order_quantity, style1, id)
	} else {
		sql = ps("insert into `shopping_cart` (pt_id,pid,user_id,order_quantity,`style`,hosted_mid,hosted_city,unix) values ('%d','%d','%d','%d','%s','%d','%s','%d');",
			pt_id, pid, this.User.UserId, order_quantity, style, hosted_mid, hosted_city, TimeNow)
	}
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("添加购物车失败:err[%v]", err)
		this.Rec = &Recv{5, "添加购物车失败", nil}
		return
	}
	this.Rec = &Recv{3, "添加购物车成功", nil}
	return
}

// sid
func (this *ShopcartController) ShopcartQuery() {
	sql := ps("SELECT sc.*,p.product_name,p.original_price,p.discount_price,p.imgurl,p.coverurl,p.web_intro,p.app_intro,p.update_unix,p.unix,p.start_date<%d as presale from `shopping_cart` as sc,`product` as p where p.id=sc.pid and p.status=1 and p.end_date>'%d' and sc.user_id='%d';", TimeNow,TimeNow, this.User.UserId)

	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询购物车出错:[%v]", err)
		this.Rec = &Recv{5, "查询购物车失败", nil}
		return
	}

	this.Rec = &Recv{3, "查询购物车成功", result}
	return
}

// sid,id(购物车订单id)
func (this *ShopcartController) ShopcartDel() {
	id, _ := this.GetInt64("id")

	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	sql := ps("delete from `shopping_cart` where id=%d;", id)
	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("删除购物车出错:[%v]", err)
		this.Rec = &Recv{5, "删除购物车失败", nil}
		return
	}

	this.Rec = &Recv{3, "删除购物车成功", result}
	return
}

// sid,id(购物车订单id),invoice_type(0-不要发票,1-个人,2-公司),invoice_head(发票抬头),recver,address(收货地址),phone,order_quantity,taxNum(税号)
func (this *ShopcartController) ShopcartSettle() {
	id, _ := this.GetInt64("id")
	recver := this.GetString("recver")
	address := this.GetString("address")
	phone := this.GetString("phone")
	invoice_type, _ := this.GetInt32("invoice_type")
	invoice_head := this.GetString("invoice_head")
	order_quantity, _ := this.GetInt("order_quantity")
	taxNum := this.GetString("taxNum")

	if !CheckArg(order_quantity, id) {
		this.Rec = &Recv{5, "请上传购买数量和购物单id", nil}
		return
	}

	// 查询购物单中产品信息
	var sql string = ps("SELECT p.`status` FROM shopping_cart as spc,product as p WHERE spc.pid=p.id and spc.id=%d;", id)
	var result []orm.Params
	db := orm.NewOrm()
	nums, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询购物单失败:[%v]", err)
		this.Rec = &Recv{5, "查询购物单失败", nil}
		return
	}
	if nums > 0 {
		status, _ := strconv.Atoi(result[0]["status"].(string))
		if status == 2 {
			this.Rec = &Recv{5, "此产品已下架,无法结算.", nil}
			return
		}
	} else {
		this.Rec = &Recv{5, "购物车无此订单", nil}
		return
	}

	sql = ps("select * from `shopping_cart` where id=%d;", id)
	nums, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询购物单失败:[%v]", err)
		this.Rec = &Recv{5, "查询购物单失败", nil}
		return
	}
	if nums <= 0 {
		this.Rec = &Recv{5, "购物单中无此订单", nil}
		return
	}
	pid, _ := strconv.Atoi(result[0]["pid"].(string))
	pt_id, _ := strconv.Atoi(result[0]["pt_id"].(string))
	hosted_city := result[0]["hosted_city"].(string)
	style := result[0]["style"].(string)
	hosted_mid, _ := strconv.Atoi(result[0]["hosted_mid"].(string))
	if hosted_mid == 2 && address == "" {
		this.Rec = &Recv{5, "请填写收货地址", nil}
		return
	}

	// 托管状态下查看是否托管已达上限
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
				totals, _ = strconv.Atoi(result[0]["quantity"].(string)) // 已投放数量
			}

			sql = ps("select num from `product_city` where pid='%d' and city like '%%%s%%';", pid, hosted_city)
			_, err := db.Raw(sql).Values(&result)
			if err == nil {
				presale, _ := strconv.Atoi(result[0]["num"].(string))
				if totals+order_quantity > presale {
					this.Rec = &Recv{5, "该城市投放数量将超上限,请减少数量或变更城市.", ps("当前已投放数量:%d", totals)}
					return
				}
			} else {
				log("查询产品预售总数失败:[%v]", err)
				this.Rec = &Recv{5, "查询产品预售总数失败", nil}
				return
			}
		}
	}

	order_no := ps("%s_%s_%s", this.User.DealerAcc, time.Now().Format("20060102150405"), GetRandomString(3))
	sql = ps("insert into `enjoy_product` (pt_id,pid,user_id,order_no,order_quantity,`style`,hosted_mid,recver,address,phone,hosted_city,invoice_type,invoice_head,taxNum,pay_deadline,unix) values ('%d','%d','%d','%s','%d','%s','%d','%s','%s','%s','%s','%d','%s','%s','%d','%d');",
		pt_id, pid, this.User.UserId, order_no, order_quantity, style, hosted_mid, recver, address, phone, hosted_city, invoice_type, invoice_head, taxNum, TimeNow+30*60, TimeNow)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("存入订单失败:[%v]", err)
		this.Rec = &Recv{5, "存入订单失败", nil}
		return
	}

	sql = ps("delete from `shopping_cart` where id=%d;", id)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("购物单删除失败:err[%v]", err)
		this.Rec = &Recv{5, "购物单删除失败", nil}
		return
	}

	sql = ps("select * from `enjoy_product` where order_no='%s';", order_no)
	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询生成订单失败:%s", err.Error())
		this.Rec = &Recv{5, "查询生成订单失败", nil}
		return
	}

	this.Rec = &Recv{3, "订单已生成,请去支付", result}
	return
}
