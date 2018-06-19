package controllers

import (
	"github.com/astaxie/beego/orm"
	"path/filepath"
	"strconv"
)

type MallController struct {
	OnlineController
}

type MallBaseController struct {
	BaseController
}

// recommend(0-一般,1-推荐),status(1-上架,2下架),begidx,counts
func (this *MallBaseController) ProductQuery() {
	status, _ := this.GetInt32("status")
	begidx, _ := this.GetInt64("begidx")
	counts, _ := this.GetInt32("counts")
	recommend, _ := this.GetInt64("recommend")

	// 参数检测
	if !CheckArg(counts) {
		this.Rec = &Recv{5, "请求总数不能为空", nil}
		return
	}

	var sql, sqlc string = "", ""
	if recommend >= 0 {
		sql = ps("select p.*,p.start_date<%d as presale,pt.name from product as p,product_type as pt where p.pt_id=pt.id and p.`status`=%d and p.end_date>=%d and p.recommend=%d order by p.unix desc limit %d,%d;", TimeNow, status, TimeNow, recommend, begidx, counts)
		sqlc = ps("select count(id) as num from `product` where `status`=%d and end_date>=%d and recommend=%d;", status, TimeNow, recommend)
	} else {
		sql = ps("select p.*,p.start_date<%d as presale,pt.name from product as p,product_type as pt where p.pt_id=pt.id and p.`status`=%d and p.end_date>=%d order by p.unix desc limit %d,%d;", TimeNow, status, TimeNow, begidx, counts)
		sqlc = ps("select count(id) as num from `product` where `status`=%d and end_date>=%d;", status, TimeNow)
	}

	db := orm.NewOrm()
	var total int = 0
	var result, res []orm.Params
	_, err := db.Raw(sqlc).Values(&result)
	if err != nil {
		log("查询失败:err[%v]", err)
		this.Rec = &Recv{5, "查询总数失败", nil}
	} else {
		total, _ = strconv.Atoi(result[0]["num"].(string))
	}

	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询失败:err[%v]", err)
		this.Rec = &Recv{5, "查询失败", nil}
		return
	}

	// 查询预售总数
	for idx := range result {
		item := result[idx]
		sql = ps("select sum(num) as num from `product_city` where pid=%s;", item["id"].(string))
		_, err = db.Raw(sql).Values(&res)
		if err == nil {
			if res[0]["num"] != nil {
				item["presale_total"], _ = strconv.Atoi(res[0]["num"].(string))
			} else {
				item["presale_total"] = 0
			}
		}

		sql = ps("select sum(order_quantity) as quantity from `enjoy_product` where `pid`=%s;", item["id"].(string))
		_, err = db.Raw(sql).Values(&res)
		if err == nil {
			if res[0]["quantity"] != nil {
				item["sale_total"], _ = strconv.Atoi(res[0]["quantity"].(string))
			} else {
				item["sale_total"] = 0
			}
		}
	}

	type RecvEx struct {
		Total  int
		Detail interface{}
	}
	this.Rec = &Recv{3, "查询成功", &RecvEx{total, result}}
	return
}

// id(产品id)
func (this *MallBaseController) ProductSalesQuery() {
	id, _ := this.GetInt32("id")

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "产品id不能为空", nil}
		return
	}

	// 定义数据结构
	type TagProduct struct {
		Product interface{}
		Order   interface{}
	}

	// 查询所有商品名称信息
	sql := ps("select id,product_name,start_date,end_date,coverurl,imgurl from `product` where id=%d;", id)
	var result []orm.Params
	db := orm.NewOrm()
	nums, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询产品失败:[%v]", err)
		this.Rec = &Recv{5, ps("查询产品失败"), nil}
		return
	} else if nums <= 0 {
		this.Rec = &Recv{5, ps("无此产品"), nil}
		return
	}

	var data TagProduct
	// 查询预售信息
	sql = ps("select city,num from `product_city` where pid='%d';", id)
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
		result[0]["presale"] = str
	}

	// 查询已售信息
	sql = ps("select sum(order_quantity) as order_quantity,hosted_city from `enjoy_product` where pid='%d' group by hosted_city;", id)
	_, err = db.Raw(sql).Values(&res)
	if err != nil {
		log("查询产品已售失败:[%v]", err)
		this.Rec = &Recv{5, ps("查询产品已售失败"), nil}
		return
	} else {
		str := ""
		saletotal := 0
		for _, itcity := range res {
			str += itcity["hosted_city"].(string) + ":" + itcity["order_quantity"].(string) + ";"
			tmpsale, _ := strconv.Atoi(itcity["order_quantity"].(string))
			saletotal += tmpsale
		}
		result[0]["sale"] = str
		result[0]["sale_total"] = saletotal
	}

	// 查询产品下所有订单
	sql = ps("select ep.order_no,u.realname,u.account,ep.hosted_mid,ep.hosted_city,ep.order_quantity,ep.pay_status,p.discount_price from `enjoy_product` as ep,`product` as p,`user` as u where p.id=ep.pid and ep.user_id=u.id and ep.pid='%d';", id)
	_, err = db.Raw(sql).Values(&res)
	if err != nil {
		log("查询订单下产品失败:[%v]", err)
		this.Rec = &Recv{5, ps("查询订单下产品失败"), nil}
		return
	}
	data.Product = result[0]
	data.Order = res

	this.Rec = &Recv{3, ps("查询未完成订单成功!"), &data}
	return
}

// sid,id
func (this *MallBaseController) ProductSoldQuery() {
	id, _ := this.GetInt64("id")

	// 参数检测
	if !CheckArg(id) {
		this.Rec = &Recv{5, "产品id不能为空", nil}
		return
	}

	// 逻辑
	db := orm.NewOrm()
	var result, res []orm.Params
	sql := ps("select city,num from `product_city` where pid=%d;", id)
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询城市预售数量出错:[%v]", err)
		this.Rec = &Recv{5, "产品预售数量查询失败", nil}
		return
	}

	sql = ps("select sum(order_quantity) as quantity,hosted_city from `enjoy_product` where `pid`=%d;", id)
	_, err = db.Raw(sql).Values(&res)
	if err != nil {
		log("查询城市托管数量出错:[%v]", err)
		this.Rec = &Recv{5, "查询城市托管数量失败", nil}
		return
	}

	type RecvEx struct {
		PreSale interface{}
		Sale    interface{}
	}
	this.Rec = &Recv{3, "查询成功", &RecvEx{result, res}}
	return
}

// sid,id,begidx,counts
func (this *MallController) ProductBuyrdQuery() {
	id, _ := this.GetInt64("id")
	begidx, _ := this.GetInt32("begidx")
	counts, _ := this.GetInt32("counts")

	// 参数检测
	if !CheckArg(counts) {
		this.Rec = &Recv{5, "请求总数不能为空", nil}
		return
	}

	var sql, sqlc string = "", ""
	sql = ps("select * from `enjoy_product` where pid=%d order by unix desc limit %d,%d;", id, begidx, counts)
	sqlc = ps("select count(id) as num from `enjoy_product` where pid=%d;", id)

	db := orm.NewOrm()
	var total int = 0
	var result []orm.Params
	_, err := db.Raw(sqlc).Values(&result)
	if err != nil {
		log("查询失败:err[%v]", err)
		this.Rec = &Recv{5, "查询总数失败", nil}
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
	this.Rec = &Recv{3, "查询成功", &RecvEx{total, result}}
	return
}

// pt_id,product_price,file(产品介绍文件),company,phone,src
func (this *MallBaseController) WillProductAdd() {
	pt_id, _ := this.GetInt32("pt_id")
	product_price, _ := this.GetFloat("product_price")
	company := this.GetString("company")
	phone := this.GetString("phone")
	src := this.GetString("src")

	// 检查参数
	if !CheckArg(pt_id, product_price, company, phone, src) {
		this.Rec = &Recv{5, "此接口各参数都不能为空", nil}
		return
	}

	var product_intro string = ""
	f, h, err := this.GetFile("file")
	if f == nil {
		product_intro = ""
	} else {
		defer f.Close()
		if err != nil {
			log("上传文件失败:err[%v]", err)
			this.Rec = &Recv{5, "文件上传失败,请检查", nil}
			return
		} else {
			// 保存位置在 /static/cowin/mall
			filename := GetSid()
			filename += filepath.Ext(h.Filename)
			err = this.SaveToFile("file", filepath.Join(conf("mallpath"), filename))
			if err != nil {
				log("文件保存失败:err[%v]", err)
				this.Rec = &Recv{5, "上传文件失败", nil}
				return
			} else {
				product_intro = ps("https://%s/%s", conf("malldown"), filename)
			}
		}
	}

	// 业务逻辑
	var sql string = ps("insert into `will_product` (pt_id,product_price,product_intro,company,phone,src,submit_unix,unix) values ('%d','%v','%s','%s','%s','%s','%d','%d');",
		pt_id, product_price, product_intro, company, phone, src, TimeNow, TimeNow)
	db := orm.NewOrm()
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("添加待审核产品失败:err[%v]", err)
		this.Rec = &Recv{5, "添加待审核产品失败", nil}
		return
	}
	this.Rec = &Recv{3, "添加待审核产品成功", nil}
}

// sid,status(0-未审核;1-已通过;2-作废),begidx,counts
func (this *MallBaseController) WillProductQuery() {
	status, _ := this.GetInt32("status")
	begidx, _ := this.GetInt32("begidx")
	counts, _ := this.GetInt32("counts")

	// 参数检测
	if !CheckArg(counts) {
		this.Rec = &Recv{5, "请求总数不能为空", nil}
		return
	}

	var sql, sqlc string = "", ""
	sql = ps("select wp.*,pt.name from will_product as wp,product_type as pt where wp.pt_id=pt.id and wp.`status`=%d limit %d,%d;", status, begidx, counts)
	sqlc = ps("select count(id) as num from will_product where `status`=%d;", status)

	db := orm.NewOrm()
	var total int = 0
	var result []orm.Params
	_, err := db.Raw(sqlc).Values(&result)
	if err != nil {
		log("查询失败:err[%v]", err)
		this.Rec = &Recv{5, "查询总数失败", nil}
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
	this.Rec = &Recv{3, "查询成功", &RecvEx{total, result}}
	return
}

// sid,id,status(1-通过,2-作废),reason
func (this *MallController) WillProductCheck() {
	id, _ := this.GetInt64("id")
	status, _ := this.GetInt32("status")
	reason := this.GetString("reason")

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	if status == 2 && !CheckArg(reason) {
		this.Rec = &Recv{5, "作废时原因不能为空", nil}
		return
	}

	// 业务逻辑
	db := orm.NewOrm()
	var sql string = ""
	if status == 2 {
		sql = ps("update `will_product` set status='%d',reason='%s' where id='%d';", status, reason, id)
	} else {
		var result []orm.Params
		num, err := db.Raw("select wp.*,pt.name from `will_product` as wp,`product_type` as pt where wp.id=? and wp.pt_id=pt.id;", id).Values(&result)
		if err != nil {
			this.Rec = &Recv{5, "查询审核产品信息时出错", nil}
			return
		}
		if num > 0 {
			sql = ps("update `will_product` set status='1' where id='%d';", id)
			_, err := db.Raw(sql).Exec()
			if err != nil {
				log("更新产品状态失败:err[%v]", id, err)
				this.Rec = &Recv{5, "审核失败", nil}
				return
			}

			sql = ps("insert into `product` (will_pid,pt_id,product_name,original_price,status,unix) values ('%d','%d','%s','%s','%d','%d');",
				id, result[0]["pt_id"].(string), result[0]["name"].(string), result[0]["product_price"].(string), status, TimeNow)
		} else {
			this.Rec = &Recv{5, "待审核产品不存在", nil}
			return
		}
	}

	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("审核产品[%d]失败:err[%v]", id, err)
		this.Rec = &Recv{5, "审核失败", nil}
		return
	}
	this.Rec = &Recv{3, "审核成功", nil}
}
