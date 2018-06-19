package controllers

import (
	"github.com/astaxie/beego/orm"
	"path/filepath"
	"strconv"
)

type ProductBaseController struct {
	BaseController
}

type ProductController struct {
	OnlineController
}

func (this *ProductBaseController) ProductTypeQuery() {
	sql := "SELECT * from `product_type`;"

	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询出错:[%v]", err)
		this.Rec = &Recv{5, "查询失败", nil}
		return
	}

	this.Rec = &Recv{3, "查询成功", result}
	return
}

// sid,name
func (this *ProductController) ProductTypeAdd() {
	name := this.GetString("name")

	sql := ps("insert `product_type` (name,unix) values('%s','%d');", name, TimeNow)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("添加产品类型出错:[%v]", err)
		this.Rec = &Recv{5, "添加产品类型失败", nil}
		return
	}

	this.Rec = &Recv{3, "添加产品类型成功", nil}
	return
}

// sid,id,name
func (this *ProductController) ProductTypeModify() {
	id, _ := this.GetInt32("id")
	name := this.GetString("name")

	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	sql := ps("UPDATE `product_type` set name='%s',unix='%d' where id=%d;", name, TimeNow, id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("修改产品类型出错:[%v]", err)
		this.Rec = &Recv{5, "修改产品类型失败", nil}
		return
	}

	this.Rec = &Recv{3, "修改产品类型成功", nil}
	return
}

// sid,id
func (this *ProductController) ProductTypeDel() {
	id, _ := this.GetInt32("id")
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	sql := ps("delete from `product_type` where id=%d;", id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("删除产品类型出错:[%v]", err)
		this.Rec = &Recv{5, "删除产品类型失败", nil}
		return
	}

	this.Rec = &Recv{3, "删除产品类型成功", nil}
	return
}

// sid,pt_id(产品类型id),product_name,original_price,discount_price,style,start_date,end_date,review_auth,imgs(总数),img1...imgn(图片参数),img(封面图),web_intro,app_intro
func (this *ProductController) ProductAdd() {
	pt_id, _ := this.GetInt32("pt_id")
	product_name := this.GetString("product_name")
	original_price, _ := this.GetFloat("original_price")
	discount_price, _ := this.GetFloat("discount_price")
	style := this.GetString("style")
	start_date, _ := this.GetInt64("start_date")
	end_date, _ := this.GetInt64("end_date")
	review_auth, _ := this.GetInt32("review_auth")
	web_intro := this.GetString("web_intro")
	app_intro := this.GetString("app_intro")

	//检查参数
	if !CheckArg(pt_id) {
		this.Rec = &Recv{5, "产品大类不存在", nil}
		return
	}

	if !CheckArg(original_price, discount_price) {
		this.Rec = &Recv{5, "产品价格不能为空", nil}
		return
	}

	if !CheckArg(product_name) {
		this.Rec = &Recv{5, "产品名称不能为空", nil}
		return
	}

	if !CheckArg(start_date, end_date) {
		this.Rec = &Recv{5, "产品预售起止时间不能为空", nil}
		return
	}

	// 存储图像
	f, h, err := this.GetFile("img")
	var coverurl string = ""
	if f == nil { //空文件
		log("图片为空")
	} else {
		defer f.Close()
		if err != nil {
			log("上传传输失败:err[%v]", err)
		} else {
			// 保存位置在 static/talk
			filename := GetSid()
			filename += filepath.Ext(h.Filename)
			err = this.SaveToFile("img", filepath.Join(conf("mallpath"), filename))
			if err != nil {
				log("文件保存失败:err[%v]", err)
			} else {
				coverurl += ps("https://%s/%s", conf("malldown"), filename)
			}
		}
	}

	imgs, _ := this.GetInt32("imgs")
	var imgurl string = ""
	if imgs > 0 {
		for i := 1; i <= int(imgs); i++ {
			strimg := ps("img%d", i)
			f, h, err = this.GetFile(strimg)
			if f == nil { //空文件
				log("产品图[%s]为空", strimg)
			} else {
				defer f.Close()
				if err != nil {
					log("上传%s传输失败:err[%v]", strimg, err)
					continue //遍历下一张图片
				} else {
					// 保存位置在 static/mall
					filename := GetSid()
					filename += filepath.Ext(h.Filename)
					err = this.SaveToFile(strimg, filepath.Join(conf("mallpath"), filename))
					if err != nil {
						log("文件%s保存失败:err[%v]", strimg, err)
						continue
					} else {
						imgurl += ps("https://%s/%s;", conf("malldown"), filename)
					}
				}
			}
		}
	}

	var sql = ps("insert into `product` (pt_id,product_name,original_price,discount_price,style,start_date,end_date,review_auth,imgurl,coverurl,web_intro,app_intro,update_unix,unix) values ('%d','%s','%v','%v','%s','%d','%d','%d','%s','%s','%s','%s','%d','%d');",
		pt_id, product_name, original_price, discount_price, style, start_date, end_date, review_auth, imgurl, coverurl, web_intro, app_intro, TimeNow, TimeNow)

	db := orm.NewOrm()
	_, err = db.Raw(sql).Exec()
	if err != nil {
		_, strerr := ChecSQLerr(err)
		log("添加产品失败:[%v]", err)
		this.Rec = &Recv{5, ps("添加产品失败:[%s]", strerr), nil}
		return
	}
	this.Rec = &Recv{3, "恭喜你,添加产品成功!", nil}
}

// sid,id,product_name(产品名称),original_price,discount_price,style,start_date,end_date,review_auth(0-不可评论,1-可以评论),imgs(总数),img1...imgn(图片参数),img(封面图),web_intro,app_intro,recommend(0-一般,1-推荐)
func (this *ProductController) ProductEdit() {
	id, _ := this.GetInt64("id")
	product_name := this.GetString("product_name")
	original_price, _ := this.GetFloat("original_price")
	discount_price, _ := this.GetFloat("discount_price")
	style := this.GetString("style")
	start_date, _ := this.GetInt64("start_date")
	end_date, _ := this.GetInt64("end_date")
	review_auth, _ := this.GetInt32("review_auth")
	imgs, _ := this.GetInt32("imgs")
	web_intro := this.GetString("web_intro")
	app_intro := this.GetString("app_intro")
	recommend, _ := this.GetInt32("recommend")

	//检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	// 存储图像
	f, h, err := this.GetFile("img")
	var coverurl string = ""
	if f == nil { //空文件
		log("图片为空")
	} else {
		defer f.Close()
		if err != nil {
			log("上传传输失败:err[%v]", err)
		} else {
			// 保存位置在 static/talk
			filename := GetSid()
			filename += filepath.Ext(h.Filename)
			err = this.SaveToFile("img", filepath.Join(conf("mallpath"), filename))
			if err != nil {
				log("文件保存失败:err[%v]", err)
			} else {
				coverurl += ps("https://%s/%s", conf("malldown"), filename)
			}
		}
	}

	var imgurl string = ""
	if imgs > 0 {
		for i := 1; i <= int(imgs); i++ {
			strimg := ps("img%d", i)
			f, h, err = this.GetFile(strimg)
			if f == nil { //空文件
				log("%s图片为空", strimg)
			} else {
				defer f.Close()
				if err != nil {
					log("上传%s传输失败:err[%v]", strimg, err)
					continue //遍历下一张图片
				} else {
					// 保存位置在 static/talk
					filename := GetSid()
					filename += filepath.Ext(h.Filename)
					err = this.SaveToFile(strimg, filepath.Join(conf("mallpath"), filename))
					if err != nil {
						log("文件%s保存失败:err[%v]", strimg, err)
						continue
					} else {
						imgurl += ps("https://%s/%s;", conf("malldown"), filename)
					}
				}
			}
		}
	}

	var sql = "update product set "
	if product_name != "" {
		sql += ps("product_name='%s',", product_name)
	}
	if original_price > 0.0 {
		sql += ps("original_price='%v',", original_price)
	}
	if discount_price > 0.0 {
		sql += ps("discount_price='%v',", discount_price)
	}
	if style != "" {
		sql += ps("style='%s',", style)
	}
	if start_date > 0 {
		sql += ps("start_date='%d',", start_date)
	}
	if end_date > 0 {
		sql += ps("end_date='%d',", end_date)
	}
	if review_auth > 0 {
		sql += ps("review_auth='%d',", review_auth)
	}
	if imgurl != "" {
		sql += ps("imgurl='%s',", imgurl)
	}
	if coverurl != "" {
		sql += ps("coverurl='%s',", coverurl)
	}
	if web_intro != "" {
		sql += ps("web_intro='%s',", web_intro)
	}
	if app_intro != "" {
		sql += ps("app_intro='%s',", app_intro)
	}
	if recommend >= 0 {
		sql += ps("recommend='%d',", recommend)
	}

	sql += ps("update_unix='%d' where id=%d", TimeNow, id)
	db := orm.NewOrm()
	_, err = db.Raw(sql).Exec()
	if err != nil {
		_, strerr := ChecSQLerr(err)
		log("编辑产品失败:[%v]", err)
		this.Rec = &Recv{5, ps("编辑产品失败:[%s]", strerr), nil}
		return
	}
	this.Rec = &Recv{3, "编辑产品成功", nil}
}

// sid,id(产品id),city(城市所在字符窜),num(投放数量)
func (this *ProductController) ProductCity() {
	id, _ := this.GetInt64("id")
	city := this.GetString("city")
	num, _ := this.GetInt64("num")

	//检查参数
	if !CheckArg(id, city, num) {
		this.Rec = &Recv{5, "产品id,城市,投放数量都不能为空,请检查!", nil}
		return
	}

	// 存储图像
	sql := ps("select id from `product_city` where pid=%d and city like '%%%s%%';", id, city)
	var result []orm.Params
	db := orm.NewOrm()
	nums, err := db.Raw(sql).Values(&result)
	if err == nil {
		if nums > 0 { //更新
			tid, _ := strconv.Atoi(result[0]["id"].(string))
			sql = ps("update `product_city` set num='%d' where id=%d;", tid)
		} else { //添加
			sql = ps("insert into `product_city` (pid,city,num) values('%d','%s','%d');", id, city, num)
		}

		_, err := db.Raw(sql).Exec()
		if err != nil {
			_, strerr := ChecSQLerr(err)
			log("编辑产品投放城市失败:[%v]", err)
			this.Rec = &Recv{5, ps("编辑产品投放城市失败:[%s]", strerr), nil}
			return
		}
	} else {
		_, strerr := ChecSQLerr(err)
		log("读取产品投放信息失败:[%v]", err)
		this.Rec = &Recv{5, ps("读取产品投放信息失败:[%s]", strerr), nil}
		return
	}

	this.Rec = &Recv{3, "编辑产品投放信息成功", nil}
}

// sid,id,status(1-上架,2下架),reason
func (this *ProductController) ProductCheck() {
	id, _ := this.GetInt64("id")
	status, _ := this.GetInt32("status")
	reason := this.GetString("reason")

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	if status == 2 && !CheckArg(reason) {
		this.Rec = &Recv{5, "下架原因不能为空", nil}
		return
	}

	// 业务逻辑
	var sql string = ps("update `product` set status='%d',reason='%s',update_unix='%d' where id='%d';", status, reason, TimeNow, id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("审核产品[%d]失败:err[%v]", id, err)
		this.Rec = &Recv{5, "审核失败", nil}
		return
	}

	this.Rec = &Recv{3, "审核成功", nil}
	return
}

// sid,id,auth(0-不可,1-可以)
func (this *ProductController) ProductReviewAuth() {
	id, _ := this.GetInt64("id")
	auth, _ := this.GetInt32("auth")

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "产品id不能为空", nil}
		return
	}

	// 业务逻辑
	var sql string = ps("update `product` set review_auth='%d',update_unix='%d' where id='%d';", auth, TimeNow, id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("修改产品评论权限[%d]失败:err[%v]", id, err)
		this.Rec = &Recv{5, "修改产品评论权限失败", nil}
		return
	}
	this.Rec = &Recv{3, "修改产品评论权限成功", nil}
	return
}

// sid,ep_id(已购订单id),content,imgs(总数),img1...imgn(图片参数)
func (this *ProductController) ProductReview() {
	content := this.GetString("content")
	imgs, _ := this.GetInt32("imgs")
	ep_id, _ := this.GetInt64("ep_id")

	//检查参数
	if !CheckArg(ep_id) {
		this.Rec = &Recv{5, "待评论产品不存在", nil}
		return
	}

	if !CheckArg(content) {
		this.Rec = &Recv{5, "请输入评论内容", nil}
		return
	}

	// 存储图像
	var imgurl string = ""
	if imgs > 0 {
		for i := 1; i <= int(imgs); i++ {
			strimg := ps("img%d", i)
			f, h, err := this.GetFile(strimg)
			if f == nil { //空文件
				log("%s图片为空", strimg)
			} else {
				defer f.Close()
				if err != nil {
					log("上传%s传输失败:err[%v]", strimg, err)
					continue //遍历下一张图片
				} else {
					// 保存位置在 static/talk
					filename := ps("%d_%d_%d_%s", this.User.UserId, ep_id, TimeNow, strimg)
					filename += filepath.Ext(h.Filename)
					err = this.SaveToFile(strimg, filepath.Join(conf("mallpath"), filename))
					if err != nil {
						log("文件%s保存失败:err[%v]", strimg, err)
						continue
					} else {
						imgurl += ps("https://%s/%s;", conf("malldown"), filename)
					}
				}
			}
		}
	}

	var sql string = ps("insert into product_review (ep_id,uid,headurl,content,imgurl,unix) values ('%d','%d','%s','%s','%s','%d');",
		ep_id, this.User.UserId, ps("https://api.yddtv.cn:10032/cowin/head/head%d", this.User.UserId), content, imgurl, TimeNow)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("评论产品失败:err[%v]", err)
		this.Rec = &Recv{5, "评论产品失败", nil}
		return
	}
	this.Rec = &Recv{3, "评论产品成功", nil}
}

// sid,id
func (this *ProductController) ProductReviewFans() {
	id, _ := this.GetInt64("id")

	//检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	//开始业务逻辑
	db := orm.NewOrm()
	var sql string = ps("update product_review set fans=fans+1 where id=%d", id)

	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("点赞产品评论[%d]失败:err[%v]", id, err)
		this.Rec = &Recv{5, "点赞失败", nil}
	}

	this.Rec = &Recv{3, ps("[%s]点赞成功", this.User.Account), nil}
	return
}

// pid,begidx,counts
func (this *ProductBaseController) ProductReviewQuery() {
	pid, _ := this.GetInt64("pid")
	begidx, _ := this.GetInt64("begidx")
	counts, _ := this.GetInt64("counts")

	// 检查参数
	if !CheckArg(pid, counts) {
		this.Rec = &Recv{5, "产品id和请求总数不能为空", nil}
		return
	}

	// 开始业务逻辑
	var totals int
	var results, res []orm.Params
	db := orm.NewOrm()
	// 查询一级评论总数
	_, err := db.Raw("SELECT count(pr.id) as num FROM `product_review` as pr,`enjoy_product` as ep WHERE pr.ep_id=ep.id and ep.pid=? and pr.status=1 and pr.prid=0;", pid).Values(&results)
	if err != nil {
		log("查询评论总数失败:[%v]", err)
		this.Rec = &Recv{5, "查询评论总数失败", nil}
		return
	} else {
		num, _ := strconv.Atoi(results[0]["num"].(string))
		if num <= 0 {
			this.Rec = &Recv{3, "无评论", nil}
			return
		}
		totals = num
	}

	var sql string = ps("SELECT pr.id,pr.ep_id,pr.prid,pr.uid,pr.headurl,pr.content,pr.imgurl,pr.fans,pr.unix,u.nick,ep.style,ep.hosted_mid,ep.hosted_city from `product_review` as pr,`enjoy_product` as ep,`user` as u WHERE pr.ep_id=ep.id and pr.uid=u.id and pr.status=1 AND ep.pid='%d' and pr.prid=0 ORDER BY pr.unix desc limit %d,%d;", pid, begidx, counts)
	// log("%s", sql)
	_, err = db.Raw(sql).Values(&results)
	if err != nil {
		log("查询评论失败:[%v]", err)
		this.Rec = &Recv{5, "查询评论失败", nil}
	} else {
		type TagReview struct {
			Review interface{}
			Reply  interface{}
		}
		type RecvEx struct {
			Total  int
			Detail []*TagReview
		}
		data := make([]*TagReview, len(results)) // 分配内存
		for idx := range results {
			item := results[idx]

			// 查询回复
			sql := ps("SELECT pr.id,pr.ep_id,pr.prid,pr.uid,pr.headurl,pr.content,pr.imgurl,pr.unix,u.nick from `product_review` as pr,`user` as u WHERE pr.uid=u.id and pr.prid='%s' ORDER BY pr.unix desc;", item["id"].(string))
			//log("%s", sql)
			_, err = db.Raw(sql).Values(&res)
			if err != nil {
				log("查询回复失败:[%v]", err)
				continue
			} else {
				if len(res) > 0 {
					data[idx] = &TagReview{item, res[0]}
				} else {
					data[idx] = &TagReview{item, res}
				}

			}
		}
		this.Rec = &Recv{3, "查询评论成功", &RecvEx{totals, data}}
	}
}

// sid,pid(产品id),id(评论id),content,imgs(总数),img1...imgn(图片参数)
func (this *ProductController) ProductReviewReply() {
	content := this.GetString("content")
	imgs, _ := this.GetInt32("imgs")
	id, _ := this.GetInt64("id")
	pid, _ := this.GetInt64("pid")

	//检查参数
	if !CheckArg(id, pid) {
		this.Rec = &Recv{5, "评论id和产品id不能为空", nil}
		return
	}

	if !CheckArg(content) {
		this.Rec = &Recv{5, "请输入评论内容", nil}
		return
	}

	// 存储图像
	var imgurl string = ""
	if imgs > 0 {
		for i := 1; i <= int(imgs); i++ {
			strimg := ps("img%d", i)
			f, h, err := this.GetFile(strimg)
			if f == nil { //空文件
				log("%s图片为空", strimg)
			} else {
				defer f.Close()
				if err != nil {
					log("上传%s传输失败:err[%v]", strimg, err)
					continue //遍历下一张图片
				} else {
					// 保存位置在 static/talk
					filename := ps("%d_%d_%d_%d_%s", this.User.UserId, pid, id, TimeNow, strimg)
					filename += filepath.Ext(h.Filename)
					err = this.SaveToFile(strimg, filepath.Join(conf("mallpath"), filename))
					if err != nil {
						log("文件%s保存失败:err[%v]", strimg, err)
						continue
					} else {
						imgurl += ps("https://%s/%s;", conf("malldown"), filename)
					}
				}
			}
		}
	}

	var sql string = ps("insert into product_review (pid,prid,uid,headurl,content,imgurl,unix) values ('%d','%d','%d','%s','%s','%s','%d');",
		pid, id, this.User.UserId, ps("https://api.yddtv.cn:10032/cowin/head/head%d", this.User.UserId), content, imgurl, TimeNow)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("回复产品评论失败:err[%v]", err)
		this.Rec = &Recv{5, "回复产品评论失败", nil}
		return
	}
	this.Rec = &Recv{3, "回复产品评论成功", nil}
}

// sid,id,status,reason(原因)
func (this *ProductController) ProductReviewCheck() {
	id, _ := this.GetInt64("id")
	status, _ := this.GetInt32("status")
	reason := this.GetString("reason")

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "评论id不能为空", nil}
		return
	}

	// 业务逻辑
	var sql string = ps("update `product_review` set status='%d',reason='%s',auditor_acc='%s' where id='%d';", status, reason, this.User.Account, id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("审核产品评论[%d]失败:err[%v]", id, err)
		this.Rec = &Recv{5, "审核产品评论失败", nil}
		return
	}
	this.Rec = &Recv{3, "审核产品评论成功", nil}
	return
}

// sid,key
func (this *ProductController) ProductReviewSearch() {
	key := this.GetString("key")

	//检查参数
	if !CheckArg(key) {
		this.Rec = &Recv{5, "搜索关键字不能为空", nil}
		return
	}

	//业务逻辑
	var sql string = ps("select * from `product_review` where `content` like '%%%s%%';", key)
	db := orm.NewOrm()
	var result []orm.Params
	nums, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("搜索产品评论失败:err[%v]", err)
		this.Rec = &Recv{5, ps("搜索产品评论失败"), nil}
		return
	}

	type RecvEx struct {
		Totals int64
		Detail interface{}
	}

	this.Rec = &Recv{3, ps("搜索产品评论成功"), &RecvEx{nums, result}}
}

// sid,id
func (this *ProductController) ProductReviewDel() {
	id, _ := this.GetInt64("id")

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "评论id不能为空", nil}
		return
	}

	// 业务逻辑
	var sql = ps("delete from `product_review` where id='%d';", id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("删除产品评论[%d]失败:err[%v]", id, err)
		this.Rec = &Recv{5, "删除产品评论失败", nil}
		return
	}
	this.Rec = &Recv{3, "删除产品评论失败", nil}
	return
}

func (this *ProductBaseController) ProductCityQuery() {

	haveby := this.GetString("haveby")

	//检查参数
	sql := "SELECT * from `product_city`"

	if !CheckArg(haveby) {
		sql += ";"
	} else {
		sql += " group by city;"
	}

	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询产品城市出错:[%v]", err)
		this.Rec = &Recv{5, "查询失败", nil}
		return
	}

	this.Rec = &Recv{3, "查询成功", result}
	return
}
