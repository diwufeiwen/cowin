package controllers

import (
	"encoding/json"
	"github.com/astaxie/beego/orm"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type UserProductController struct {
	OnlineController
}

type UserProductBaseController struct {
	BaseController
}

func (this *UserProductBaseController) ProductSerialQuery() {
	sql := "SELECT up.product_no,hm.text as buyway,up.unix from `user_product` as up,`host_method` as hm where up.`friendpdt_no`='' and up.hosted_mid=hm.id;"

	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询产品编号出错:[%v]", err)
		this.Rec = &Recv{5, "查询失败", nil}
		return
	}

	this.Rec = &Recv{3, "查询成功", result}
	return
}

// product_no(内部编号),ex_no(外部编号)
func (this *UserProductBaseController) ProductSerialMatch() {
	product_no := this.GetString("product_no")
	ex_no := this.GetString("ex_no")

	if !CheckArg(product_no, ex_no) {
		this.Rec = &Recv{5, "编号不能为空", nil}
		return
	}

	sql := ps("SELECT id,ep_id,hosted_mid,user_id from `user_product` where `product_no`='%s';", product_no)
	db := orm.NewOrm()
	var result []orm.Params
	nums, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询产品出错:[%v]", err)
		this.Rec = &Recv{5, "查询产品失败", nil}
		return
	}
	var hosted_mid, id, ep_id, user_id int
	if nums > 0 {
		hosted_mid, _ = strconv.Atoi(result[0]["hosted_mid"].(string))
		id, _ = strconv.Atoi(result[0]["id"].(string))
		ep_id, _ = strconv.Atoi(result[0]["ep_id"].(string))
		user_id, _ = strconv.Atoi(result[0]["user_id"].(string))
	} else {
		this.Rec = &Recv{5, "产品编号不存在", nil}
		return
	}

	// 请求产品信息
	client := &http.Client{}
	strval := ps("alias=%s&i=/v1/charging/all&rf=3&ts=%d&v=1.0.1", ex_no, time.Now().Unix())
	strpara := strval + "0q238ie8347fj3659fh$&HF^IE812*(23z7^&*12ksjSKW0"
	strsign := StrToMD5(strpara)
	strval += ps("&encry=%s", strsign)
	req, err := http.NewRequest("POST", "https://www.wacdd.com/external/v1/charging/all", strings.NewReader(strval))
	if err != nil {
		log("创建HttpRequest失败:%s", err.Error())
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	resp, err := client.Do(req)
	if err != nil {
		log("请求失败:%s", err.Error())
		return
	}
	defer resp.Body.Close()
	type ProductInfo struct {
		Code int `json:"code"`
		Data []struct {
			Alias           string `json:"alias"`
			LastUseTime     string `json:"last_use_time"`
			Status          string `json:"status"`
			Latitude        string `json:"latitude"`
			Longitude       string `json:"longitude"`
			SpecificAddress string `json:"specific_address"`
			ProvinceName    string `json:"province_name"`
			CityName        string `json:"city_name"`
			AreaName        string `json:"area_name"`
			PositionName    string `json:"position_name"`
			StoreName       string `json:"store_name"`
		} `json:"data"`
	}
	var pi ProductInfo
	if resp.StatusCode == 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		//解析JSON数据
		err = json.Unmarshal(body, &pi)
		if err != nil {
			log("解析json数据[%v]失败:[%s]", body, err.Error())
		}
	} else {
		log("http状态错误:%v", resp.StatusCode)
	}
	status := 0
	if hosted_mid == 1 {
		status = 1 //投放中
	} else {
		status = 2 //提货中
	}
	if len(pi.Data) > 0 {
		sql = ps("update `user_product` set last_use_time='%s',friend_status='%s',latitude='%s',longitude='%s',specific_address='%s',province_name='%s',city_name='%s',area_name='%s',position_name='%s',store_name='%s',`friendpdt_no`='%s',status='%d',unix='%d' where `product_no`='%s';",
			pi.Data[0].LastUseTime, pi.Data[0].Status, pi.Data[0].Latitude, pi.Data[0].Longitude, pi.Data[0].SpecificAddress, pi.Data[0].ProvinceName, pi.Data[0].CityName, pi.Data[0].AreaName, pi.Data[0].PositionName, pi.Data[0].StoreName, ex_no, status, TimeNow, product_no)
	} else {
		sql = ps("update `user_product` set `friendpdt_no`='%s',status='%d',unix='%d' where `product_no`='%s';", ex_no, status, TimeNow, product_no)
	}
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("匹配出错:[%v]", err)
		this.Rec = &Recv{5, "匹配失败", nil}
		return
	}

	// 更新订单状态为已完成
	sql = ps("SELECT count(id) as cnts from user_product where ep_id=%d and status=0;", ep_id)
	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询产品信息失败:[%v]", err)
	} else {
		cnts, _ := strconv.Atoi(result[0]["cnts"].(string))
		if cnts == 0 {
			sql = ps("UPDATE `enjoy_product` set pay_status='2' where `id`='%d';", ep_id)
			_, err = db.Raw(sql).Exec()
			if err != nil {
				log("更新订单状态失败:[%v]", err)
			}
		}
	}

	// 添加通知消息
	var str string = ""
	switch status {
	case 1:
		str = ps("你的编号为[%s]的产品已经投放,请前去查看", product_no)
	case 2:
		str = ps("你的编号为[%s]的产品正在快递中,请随时关注状态", product_no)
	}
	sql = ps("insert into `letter` (title,content,send_uid,recv_uid,unix) values ('%s','%s','0','%d','%d')",
		"通知", str, user_id, TimeNow)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("添加通知失败:[%v]", err)
	}

	// 请求产品使用信息
	strval = ps("alias=%s&i=/v1/charging/all&rf=3&ts=%d&v=1.0.1", ex_no, time.Now().Unix())
	strpara = strval + "0q238ie8347fj3659fh$&HF^IE812*(23z7^&*12ksjSKW0"
	strsign = StrToMD5(strpara)
	strval += ps("&encry=%s", strsign)
	req, err = http.NewRequest("POST", "https://www.wacdd.com/external/v1/charging/device_data", strings.NewReader(strval))
	if err != nil {
		log("创建HttpRequest失败:%s", err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	resp, err = client.Do(req)
	if err != nil {
		log("请求失败:%s", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		//解析JSON数据
		type HistoryUseinfo struct {
			Code int `json:"code"`
			Data []struct {
				GrowthFansNum string `json:"growth_fans_num"`
				UseNum        string `json:"use_num"`
				Date          string `json:"date"`
			} `json:"data"`
		}
		var hui HistoryUseinfo
		err = json.Unmarshal(body, &hui)
		if err != nil {
			log("解析json数据失败:[%s]", err.Error())
		} else {
			totals := len(hui.Data)
			var sqls []string
			for i := 0; i < totals; i++ {
				sqls = append(sqls, ps("insert into `product_use` (up_id,growth_fans_num,use_num,`date`,unix) values('%d','%s','%s','%s','%d')",
					id, hui.Data[i].GrowthFansNum, hui.Data[i].UseNum, hui.Data[i].Date, TimeNow))
			}
			db.Begin() //开启事务
			for _, ele := range sqls {
				_, err := db.Raw(ele).Exec()
				if err != nil {
					log("更新产品信息失败:[%v]", err)
				}
			}
			db.Commit() //提交事务
		}
	} else {
		log("http状态错误:%v", resp.StatusCode)
	}

	this.Rec = &Recv{3, "匹配成功", nil}
	return
}

// sid,id,use_time,total_fee,content
func (this *UserProductController) ProductUsePay() {
	id, _ := this.GetInt64("id")
	use_time, _ := this.GetInt64("use_time")
	total_fee, _ := this.GetFloat("total_fee", 64)
	content := this.GetString("content")

	if !CheckArg(id) {
		this.Rec = &Recv{5, "产品id不能为空", nil}
		return
	}

	opera_income := total_fee * 0.45
	platform_income := total_fee * 0.55 * 0.1
	owner_income := total_fee*0.55 - platform_income

	var sql string = ps("insert into `income` (up_id,owner_income,platform_income,opera_income,user_id,use_time,total_fee,content,unix) values ('%d','%v','%v','%v','%d','%d','%v','%s','%d');",
		id, owner_income, platform_income, opera_income, this.User.UserId, use_time, total_fee, content, TimeNow)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("支付处理失败:err[%v]", err)
		this.Rec = &Recv{5, "支付处理失败", nil}
		return
	}

	this.Rec = &Recv{3, "支付处理成功", nil}
}

// sid,id(产品id号)
func (this *UserProductController) RepairDeal() {
	id, _ := this.GetInt64("id")

	if !CheckArg(id) {
		this.Rec = &Recv{5, "产品id不能为空", nil}
		return
	}

	// 更新产品状态
	sql := ps("update `user_product` set repair_status=0,repair_num=0,repair_acc='%s' where id='%d';", this.User.Account, id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("处理报修失败:err[%v]", err)
		this.Rec = &Recv{5, "处理报修失败", nil}
		return
	}

	this.Rec = &Recv{3, "处理报修成功!", nil}
	return
}

// sid,id(产品id),pos(位置),reason(原因),img(拍照)
func (this *UserProductController) ProductRepair() {
	id, _ := this.GetInt64("id")
	pos := this.GetString("pos")
	reason := this.GetString("reason")

	if !CheckArg(id) {
		this.Rec = &Recv{5, "产品id不能为空", nil}
		return
	}

	f, h, err := this.GetFile("img")
	var imgurl string = ""
	if f == nil { //空文件
		imgurl = ""
		log("上传文件为空")
	} else {
		defer f.Close()
		if err != nil {
			log("上传文件传输失败:err[%v]", err)
			this.Rec = &Recv{5, "文件传输失败,请检查", nil}
			return
		} else {
			// 保存位置在 static/upload,没有文件夹要先创建
			filename := GetSid()
			filename += filepath.Ext(h.Filename)
			date := time.Now().Format("20060102")
			dirpath := filepath.Join(conf("tmppath"), date)
			err = os.MkdirAll(dirpath, os.ModePerm)
			if err != nil {
				log("创建文件夹[%s]失败err[%v]", dirpath, err)
			}
			err = this.SaveToFile("img", filepath.Join(dirpath, filename)) //第一个参数是http请求中的参数名
			if err != nil {
				log("上传文件保存失败:err[%v]", err)
				this.Rec = &Recv{5, "上传文件失败", nil}
				return
			} else {
				downdir := filepath.Join(conf("tmpdown"), date)
				imgurl = "https://" + filepath.Join(downdir, filename)
				imgurl = strings.Replace(imgurl, "\\", "/", -1)

			}
		}
	}

	var sql string = ps("insert into `repair` (uid,up_id,pos,reason,imgurl,unix) values ('%d','%d','%s','%s','%s','%d');", this.User.UserId, id, pos, reason, imgurl, TimeNow)
	db := orm.NewOrm()
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("报修失败:err[%v]", err)
		this.Rec = &Recv{5, ps("[%s]报修失败", this.User.Account), nil}
		return
	}

	// 更新产品状态
	sql = ps("update `user_product` set repair_status=1,repair_num=repair_num+1 where id='%d';", id)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("报修失败:err[%v]", err)
		this.Rec = &Recv{5, ps("[%s]报修失败", this.User.Account), nil}
		return
	}

	this.Rec = &Recv{3, ps("[%s]报修成功!", this.User.Account), nil}
}

// sid,id(产品id)
func (this *UserProductController) UserProductUseRecord() {
	id, _ := this.GetInt32("id")

	// 参数检测
	if !CheckArg(id) {
		this.Rec = &Recv{5, "产品id不能为空", nil}
		return
	}

	// 业务逻辑
	sql := ps("SELECT * from `product_use` where up_id=%d;", id)
	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询产品使用信息出错:[%v]", err)
		this.Rec = &Recv{5, "查询使用记录失败", nil}
		return
	}

	this.Rec = &Recv{3, "查询使用记录成功", result}
}

// sid,city(投放城市),pt_id(类型id)
func (this *UserProductController) UserProductGeneral() {
	pt_id, _ := this.GetInt32("pt_id")
	city := this.GetString("city")

	if !CheckArg(pt_id) {
		this.Rec = &Recv{5, "产品类型不能为空", nil}
		return
	}

	// 查询自己在该城市运营产品总数
	sql := ps("select count(up.id) as opts from `user_product` as up,`enjoy_product` as ep where ep.id=up.ep_id and ep.pt_id=%d and ep.hosted_city like '%%%s%%' and up.user_id=%d and up.status=2;", pt_id, city, this.User.UserId)
	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询总数出错:[%v]", err)
		this.Rec = &Recv{5, "查询总数失败", nil}
		return
	}
	var opts, pts int
	var yincome, income float64
	if result[0]["opts"] != nil {
		opts, _ = strconv.Atoi(result[0]["opts"].(string))
	}

	// 查询该城市运营产品总数
	sql = ps("select count(up.id) as pts from `user_product` as up,`enjoy_product` as ep where ep.id=up.ep_id and ep.pt_id=%d and ep.hosted_city like '%%%s%%'and up.status=2;", pt_id, city)
	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询总数出错:[%v]", err)
		this.Rec = &Recv{5, "查询总数失败", nil}
		return
	}
	if result[0]["pts"] != nil {
		pts, _ = strconv.Atoi(result[0]["pts"].(string))
	}

	// 昨日总收益
	today := time.Now().Format("2006-01-02")
	today_t, _ := time.ParseInLocation("2006-01-02", today, time.Local)

	sql = ps("select sum(i.owner_income) as yincome from `user_product` as up,`enjoy_product` as ep,`income` as i where  ep.id=up.ep_id and up.id=i.up_id and up.status=2 and ep.pt_id=%d and ep.hosted_city like '%%%s%%' and i.unix>=%d and i.unix<%d;", pt_id, city, today_t.AddDate(0, 0, -1).Unix(), today_t.Unix())
	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询总数出错:[%v]", err)
		this.Rec = &Recv{5, "查询总数失败", nil}
		return
	}
	if result[0]["yincome"] != nil {
		yincome, _ = strconv.ParseFloat(result[0]["yincome"].(string), 64)
	}

	// 总收益
	sql = ps("select sum(i.owner_income) as income from `user_product` as up,`enjoy_product` as ep,`income` as i where ep.id=up.ep_id and up.id=i.up_id and up.status=2 and ep.pt_id=%d and ep.hosted_city like '%%%s%%';", pt_id, city)
	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询总数出错:[%v]", err)
		this.Rec = &Recv{5, "查询总数失败", nil}
		return
	}
	if result[0]["income"] != nil {
		income, _ = strconv.ParseFloat(result[0]["income"].(string), 64)
	}

	type RecvEx struct {
		Opts    int
		Pts     int
		Yincome float64
		Income  float64
	}

	this.Rec = &Recv{3, "查询成功", &RecvEx{opts, pts, yincome, income}}
	return
}

// sid,city(投放城市),pt_id(类型id),begidx,counts
func (this *UserProductController) UserProductQuery() {
	pt_id, _ := this.GetInt32("pt_id")
	city := this.GetString("city")
	begidx, _ := this.GetInt64("begidx")
	counts, _ := this.GetInt32("counts")

	if !CheckArg(pt_id, counts) {
		this.Rec = &Recv{5, "产品类型不能为空", nil}
		return
	}

	sql := ps("select count(up.id) as totals from `user_product` as up,`enjoy_product` as ep where ep.id=up.ep_id and ep.pt_id=%d and up.user_id=%d and up.status=2", pt_id, this.User.UserId)
	if city != "" {
		sql += ps(" and ep.hosted_city like '%%%s%%';", city)
	}
	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询总数出错:[%v]", err)
		this.Rec = &Recv{5, "查询总数失败", nil}
		return
	}
	totals, _ := strconv.Atoi(result[0]["totals"].(string))

	sql = ps("select up.product_no,up.id,up.ep_id,up.unix,up.latitude,up.longitude,ep.style,ep.hosted_city,ep.pt_id,p.coverurl,p.imgurl,ep.pid from `user_product` as up,`enjoy_product` as ep,`product` as p where ep.id=up.ep_id and ep.pid=p.id and ep.pt_id=%d and ep.hosted_city like '%%%s%%' and up.user_id=%d and up.status=2 limit %d,%d;",
		pt_id, city, this.User.UserId, begidx, counts)
	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询产品出错:[%v]", err)
		this.Rec = &Recv{5, "查询产品出错", nil}
		return
	}

	// 查询使用数和涨分数
	for idx := range result {
		item := result[idx]
		sql = ps("select sum(growth_fans_num) as growth_fans_num,sum(use_num) as use_num from `product_use` where up_id='%s';", item["id"].(string))
		var res []orm.Params
		_, err = db.Raw(sql).Values(&res)
		if err != nil {
			log("查询使用数据失败:[%v]", err)
			this.Rec = &Recv{5, "查询使用数据失败", nil}
			return
		}
		if res[0]["growth_fans_num"] != nil {
			income, _ := strconv.ParseFloat(res[0]["growth_fans_num"].(string), 64)
			item["growth_fans_num"] = income
		} else {
			item["growth_fans_num"] = 0
		}

		if res[0]["use_num"] != nil {
			income, _ := strconv.ParseFloat(res[0]["use_num"].(string), 64)
			item["use_num"] = income
		} else {
			item["use_num"] = 0
		}

		//查询累计收益
		sql = ps("select sum(i.owner_income) as income from `user_product` as up,`income` as i where up.id=i.up_id and up.status=2 and up.id='%s';", item["id"].(string))
		_, err = db.Raw(sql).Values(&res)
		if err != nil {
			log("查询总数出错:[%v]", err)
			this.Rec = &Recv{5, "查询总数失败", nil}
			return
		}
		if res[0]["income"] != nil {
			income, _ := strconv.ParseFloat(res[0]["income"].(string), 64)
			item["income"] = income
		} else {
			item["income"] = 0
		}

	}

	// 查询累计收益
	type RecvEx struct {
		Total  int
		Detail interface{}
	}
	this.Rec = &Recv{3, "查询成功", &RecvEx{totals, result}}
	return
}

// sid,id(产品id号),status(-1-报废,3-已签收),reason
func (this *UserProductController) UserProductModify() {
	id, _ := this.GetInt64("id")
	status, _ := this.GetInt32("status")
	reason := this.GetString("reason")

	if !CheckArg(id) {
		this.Rec = &Recv{5, "产品id不能为空", nil}
		return
	}

	sql := ps("update `user_product` set status='%d',reason='%s' where id=%d;", status, reason, id)

	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("修改出错:[%v]", err)
		this.Rec = &Recv{5, "修改失败", nil}
		return
	}

	this.Rec = &Recv{3, "修改成功", nil}
	return
}

// sid,id,track_num
func (this *UserProductController) UserProductTracknum() {
	id, _ := this.GetInt64("id")
	track_num := this.GetString("track_num")

	if !CheckArg(id, track_num) {
		this.Rec = &Recv{5, "参数不能为空", nil}
		return
	}

	sql := ps("update `user_product` set track_num='%s' where id=%d and hosted_mid=2;", track_num, id)

	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("添加快递号失败:[%v]", err)
		this.Rec = &Recv{5, "添加快递号失败", nil}
		return
	}

	this.Rec = &Recv{3, "添加快递号成功", nil}
	return
}

// latit_min,latit_max,longit_min,longit_max
func (this *UserProductBaseController) ProductBaseinfo() {
	latit_min, _ := this.GetFloat("latit_min", 32)
	latit_max, _ := this.GetFloat("latit_max", 32)
	longit_min, _ := this.GetFloat("longit_min", 32)
	longit_max, _ := this.GetFloat("longit_max", 32)

	if !CheckArg(latit_min, latit_max, longit_min, longit_max) {
		log("该接口参数都不能为空")
		return
	}

	// 逻辑接口
	sql := ps("select * from `user_product` where latitude>=%v and latitude<=%v and longitude>=%v and longitude<=%v", latit_min, latit_max, longit_min, longit_max)
	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询产品出错:[%v]", err)
		this.Rec = &Recv{5, "查询产品失败", nil}
		return
	}

	this.Rec = &Recv{3, "查询产品成功", result}
	return
}

// sid,id(产品id)
func (this *UserProductController) ProductUseinfo() {
	id, _ := this.GetInt64("id")

	if !CheckArg(id) {
		log("id不能为空")
		return
	}

	// 逻辑接口
	sql := ps("select * from `product_use` where up_id=%d order by id desc;", id)
	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询产品使用信息出错:[%v]", err)
		this.Rec = &Recv{5, "查询产品使用信息失败", nil}
		return
	}

	this.Rec = &Recv{3, "查询产品使用信息成功", result}
	return
}

// sid,up_id(多个之间以,分隔),recver,address,phone,quantity(提货总数)
func (this *UserProductController) UserProductPickup() {
	up_id := this.GetString("up_id")
	recver := this.GetString("recver")
	address := this.GetString("address")
	phone := this.GetString("phone")
	quantity, _ := this.GetInt32("quantity")

	if !CheckArg(up_id, recver, address, phone) {
		this.Rec = &Recv{5, "参数存在空值", nil}
		return
	}

	// 检测是否可以提为一单
	sql := ps("select count(DISTINCT up.user_id) as user_num,count(DISTINCT ep.hosted_city) as city_num,count(DISTINCT ep.pt_id) as pt_num from `user_product` as up,`enjoy_product` as ep where up.ep_id=ep.id and up.id in (%s);", up_id)
	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询失败:[%v]", err)
		this.Rec = &Recv{5, "提货失败", nil}
		return
	}

	if result[0]["user_num"] != nil {
		user_num, _ := strconv.Atoi(result[0]["user_num"].(string))
		if user_num > 1 {
			this.Rec = &Recv{5, "不同用户的订单不能同时提", nil}
			return
		}
	}

	if result[0]["city_num"] != nil {
		city_num, _ := strconv.Atoi(result[0]["city_num"].(string))
		if city_num > 1 {
			this.Rec = &Recv{5, "不同城市的订单不能同时提", nil}
			return
		}
	}

	if result[0]["pt_num"] != nil {
		pt_num, _ := strconv.Atoi(result[0]["pt_num"].(string))
		if pt_num > 1 {
			this.Rec = &Recv{5, "不同产品类型的订单不能同时提", nil}
			return
		}
	}

	// 提货订单生成
	order_no := ps("%s_%s_%s", this.User.DealerAcc, time.Now().Format("20060102150405"), GetRandomString(3))
	sql = ps("insert into `userpdt_pickup` (uid,up_id,order_no,recver,address,phone,quantity,unix) values('%d','%s','%s','%s','%s','%s','%d','%d');",
		this.User.UserId, up_id, order_no, recver, address, phone, quantity, TimeNow)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("提货失败:[%v]", err)
		this.Rec = &Recv{5, "提货失败", nil}
		return
	}

	// 修改订单下产品状态
	sql = ps("update `user_product` set `status`=0,`hosted_mid`=2 where id in (%s);", up_id)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("提货失败:[%v]", err)
		this.Rec = &Recv{5, "提货失败", nil}
		return
	}

	this.Rec = &Recv{3, "产品提货成功", nil}
}

// sid,id(订单id)
func (this *UserProductController) UserPdtPickReceipt() {
	id, _ := this.GetInt32("id")

	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	sql := ps("UPDATE `userpdt_pickup` set status=2 where id=%d and uid=%d;", id, this.User.UserId)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("更新订单状态失败:[%v]", err)
		this.Rec = &Recv{5, "订单确认收货失败", nil}
		return
	}

	// 更新订单下所有产品信息
	sql = ps("select up_id from `userpdt_pickup` where id=%d;", id)
	var result []orm.Params
	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询订单下id失败:[%v]", err)
		this.Rec = &Recv{5, "订单确认收货失败", nil}
		return
	}

	sql = ps("update `user_product` set `status`=2 where id in (%s);", result[0]["up_id"].(string))
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("更新订单下产品状态失败:[%v]", err)
		this.Rec = &Recv{5, "订单确认收货失败", nil}
		return
	}

	this.Rec = &Recv{3, "订单确认收货成功", nil}
}

// sid,begidx,counts,status(0-未完成,1-已完成)
func (this *UserProductController) PickupProductOrderQuery() {
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
	case 0:
		sql = ps("SELECT upu.*,tc.code,tc.name from `userpdt_pickup` as upu,`transport_company` as tc where upu.tpc_id=tc.id and upu.uid=%d and upu.status<2 limit %d,%d", this.User.UserId, begidx, counts)
		sqlc = ps("SELECT id from `userpdt_pickup` where uid='%d' and status<2", this.User.UserId)
	case 1:
		sql = ps("SELECT upu.*,tc.code,tc.name from `userpdt_pickup` as upu,`transport_company` as tc where upu.tpc_id=tc.id and upu.uid=%d and upu.status=2 limit %d,%d", this.User.UserId, begidx, counts)
		sqlc = ps("SELECT id from `userpdt_pickup` where uid='%d' and status=2", this.User.UserId)
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

	this.Rec = &Recv{3, "查询订单成功", &RecvEx{nums, result}}
	return
}

func (this *UserProductController) TransportQuery() {
	sql := "SELECT * from `transport_company`;"

	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询失败:[%v]", err)
		this.Rec = &Recv{5, "查询失败", nil}
		return
	}

	this.Rec = &Recv{3, "查询成功", result}
	return
}
