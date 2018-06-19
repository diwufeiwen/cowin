package controllers

import (
	"github.com/astaxie/beego/orm"
	"path/filepath"
	"strconv"
	"strings"
)

type TalkaboutController struct {
	OnlineController
}

type TalkaboutBaseController struct {
	BaseController
}

// sid,text,imgs(总数),img1...imgn(图片参数)
func (this *TalkaboutController) TalkaboutPublish() {
	text := this.GetString("text")
	imgs, _ := this.GetInt32("imgs")

	//检查参数
	if !CheckArg(text) {
		this.Rec = &Recv{5, "请输入说说内容", nil}
		return
	}
	text = strings.Replace(text, "'", "''", -1)

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
					filename := GetSid()
					filename += filepath.Ext(h.Filename)
					err = this.SaveToFile(strimg, filepath.Join(conf("talkpath"), filename))
					if err != nil {
						log("文件%s保存失败:err[%v]", strimg, err)
						continue
					} else {
						imgurl += ps("https://%s/%s;", conf("talkdown"), filename)
					}
				}
			}
		}
	}

	var sql string = ps("insert into talkabout (uid,text,headurl,imgurl,unix) values ('%d','%s','%s','%s','%d');",
		this.User.UserId, text, ps("https://api.yddtv.cn:10032/cowin/head/head%d", this.User.UserId), imgurl, TimeNow)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("添加说说失败:err[%v]", err)
		this.Rec = &Recv{5, "添加说说失败", nil}
		return
	}
	this.Rec = &Recv{3, "添加说说成功!", nil}
}

// sid,id
func (this *TalkaboutController) TalkaboutDel() {
	id, _ := this.GetInt64("id")

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "说说id不能为空", nil}
		return
	}

	// 业务逻辑
	sql := ps("select id from talkabout where id='%d' and uid=%d;", id, this.User.UserId)
	db := orm.NewOrm()
	var result []orm.Params
	nums, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询说说出错:%s", err.Error())
		this.Rec = &Recv{5, "查询说说失败", nil}
		return
	} else {
		if nums <= 0 {
			this.Rec = &Recv{5, "无法删除别人发表的说说", nil}
			return
		}
	}

	sql = ps("delete from talkabout where id='%d' and uid=%d;", id, this.User.UserId)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("删除说说[%d]失败:[%v]", id, err)
		this.Rec = &Recv{5, "删除说说失败", nil}
		return
	}
	this.Rec = &Recv{3, "删除说说成功", nil}
	return
}

// uid(不传查询所有),begidx(开始索引),counts(条数)
func (this *TalkaboutBaseController) TalkaboutQuery() {
	begidx, _ := this.GetInt32("begidx")
	counts, _ := this.GetInt32("counts")
	uid, _ := this.GetInt64("uid")
	id, _ := this.GetInt64("id")

	//检查参数
	if !CheckArg(counts) {
		this.Rec = &Recv{5, "总数不能为空", nil}
		return
	}

	//业务逻辑
	db := orm.NewOrm()
	var result []orm.Params
	var (
		sql    string
		sqlc   string
		err    error
		totals int = 0
	)

	if id >0 {
		totals = 1
		sql = ps("select ta.id,ta.uid,ta.text,ta.headurl,u.nick,ta.imgurl,ta.viewers,ta.unix from talkabout as ta,user as u where ta.uid=u.id and ta.id=%d;", id)
	}else {

		if uid > 0 {
			sql = ps("select ta.id,ta.uid,ta.text,ta.headurl,u.nick,ta.imgurl,ta.viewers,ta.unix from talkabout as ta,user as u where ta.uid=u.id and ta.uid=%d and ta.status!=2 and ta.status!=0 order by ta.unix desc limit %d,%d;", uid, begidx, counts)
			sqlc = ps("select count(id) as num from talkabout where uid=%d and status!=0 and status!=2;", uid)
		} else {
			sql = ps("select ta.id,ta.uid,ta.text,ta.headurl,u.nick,ta.imgurl,ta.viewers,ta.unix from talkabout as ta,user as u where ta.uid=u.id and ta.status!=2 and ta.status!=0 order by ta.unix desc limit %d,%d;", begidx, counts)
			sqlc = "select count(id) as num from talkabout where status!=0 and status!=2;"
		}

		_, err = db.Raw(sqlc).Values(&result)
		if err == nil {
			totals, _ = strconv.Atoi(result[0]["num"].(string))
		}
	}


	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询说说失败:err[%v]", err)
		this.Rec = &Recv{5, ps("查询说说失败"), nil}
		return
	}

	//评论数和点赞数
	for idx := range result {
		item := result[idx]
		var restmp []orm.Params
		num, err := db.Raw("SELECT id from talkabout_review where tid=?;", item["id"]).Values(&restmp)
		if err == nil {
			item["review"] = num
		} else {
			item["review"] = 0
		}

		num, err = db.Raw("SELECT id from talkabout_fans where tid=?;", item["id"]).Values(&restmp)
		if err == nil {
			item["fans"] = num
		} else {
			item["fans"] = 0
		}
	}

	type RecvEx struct {
		Total  int
		Detail interface{}
	}
	this.Rec = &Recv{3, ps("查询说说成功!"), &RecvEx{totals, result}}
}

// sid,begidx(开始索引),counts(条数),status
func (this *TalkaboutController) TalkaboutBsQuery() {
	begidx, _ := this.GetInt32("begidx")
	counts, _ := this.GetInt32("counts")
	status, _ := this.GetInt32("status")

	//检查参数
	if !CheckArg(counts) {
		this.Rec = &Recv{5, "总数不能为空", nil}
		return
	}

	//业务逻辑
	db := orm.NewOrm()
	var result []orm.Params
	var (
		sql, sqlc string
		totals    int = 0
	)

	if status < 0 {
		sql = ps("select ta.*,u.nick from talkabout as ta,user as u where ta.uid=u.id order by ta.unix desc limit %d,%d;", begidx, counts)
		sqlc = "select count(id) as num from talkabout;"
	} else {
		sql = ps("select ta.*,u.nick from talkabout as ta,user as u where ta.uid=u.id and ta.status='%d' order by ta.unix desc limit %d,%d;", status, begidx, counts)
		sqlc = ps("select count(id) as num from talkabout where status='%d';", status)
	}

	_, err := db.Raw(sqlc).Values(&result)
	if err == nil {
		totals, _ = strconv.Atoi(result[0]["num"].(string))
	}

	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询说说失败:err[%v]", err)
		this.Rec = &Recv{5, ps("查询说说失败"), nil}
		return
	}

	type RecvEx struct {
		Total  int
		Detail interface{}
	}
	if status != 3 {
		this.Rec = &Recv{3, ps("查询说说成功!"), &RecvEx{totals, result}}
		return
	} else {
		type TalkaboutTag struct {
			Talkabout interface{}
			Report    interface{}
		}

		var tbarr []TalkaboutTag
		tbarr = make([]TalkaboutTag, len(result))
		for idx := range result {
			item := result[idx]
			report_num, _ := strconv.Atoi(item["report_num"].(string))
			if report_num > 0 {
				var res []orm.Params
				_, err := db.Raw("select tr.report_uid,tr.report_id,tr.report_reason,u.nick,r.text as report_nick from talkabout_report as tr,user as u,report as r where r.id=tr.report_id and tr.report_uid=u.id and tr.tid=? and tr.status=0;", item["id"].(string)).Values(&res)
				if err == nil {
					tbarr[idx].Report = res
				} else {
					log("查询举报内容失败:%v", err)
				}
			}
			tbarr[idx].Talkabout = item
		}
		this.Rec = &Recv{3, ps("查询说说成功!"), &RecvEx{totals, tbarr}}
		return
	}

	this.Rec = &Recv{3, ps("查询说说成功!"), nil}
}

// sid,id,status,reason
func (this *TalkaboutController) TalkaboutCheck() {
	id, _ := this.GetInt32("id")
	status, _ := this.GetInt32("status")
	reason := this.GetString("reason")

	//检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	// 通知
	sql := ps("select uid,text from `talkabout` where id='%d';", id)
	var result, res []orm.Params
	db := orm.NewOrm()
	nums, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询说说失败[%v]", err)
	} else if nums <= 0 {
		this.Rec = &Recv{5, "待审核说说不存在", nil}
		return
	}

	contintro := result[0]["text"].(string)
	if len(contintro) > 50 {
		contintro = contintro[0:50]
		contintro += "..."
	}
	contintro = strings.Replace(contintro, "'", "''", -1)
	var str string = ""
	if status == 2 {
		str = ps("你的说说[%s]经系统核实不符合要求,已被屏蔽,以后请注意.", contintro)
		sql = ps("insert into `letter` (title,content,send_uid,recv_uid,unix) values ('%s','%s','0','%s','%d')", "通知", str, result[0]["uid"].(string), TimeNow)
		_, err = db.Raw(sql).Exec()
		if err != nil {
			log("添加通知失败:[%v]", err)
			this.Rec = &Recv{5, "审核失败", nil}
			return
		}

		// 查询举报人id
		sql = ps("select report_uid from `talkabout_report` where id='%d';", id)
		_, err = db.Raw(sql).Values(&res)
		str = ps("你举报的说说[%s]已经被处理,感谢你对系统的贡献.", contintro)
		for _, item := range res {
			sql = ps("insert into `letter` (title,content,send_uid,recv_uid,unix) values ('%s','%s','0','%s','%d')", "通知", str, item["report_uid"].(string), TimeNow)
			_, err = db.Raw(sql).Exec()
			if err != nil {
				log("添加通知失败:[%v]", err)
				continue
			}
		}
	} else {
		// 查询举报人id
		sql = ps("select report_uid from `talkabout_report` where id='%d';", id)
		_, err = db.Raw(sql).Values(&res)
		str = ps("你举报的说说[%s]经系统核实符合要求,请注意不要恶意举报.", contintro)
		for _, item := range res {
			sql = ps("insert into `letter` (title,content,send_uid,recv_uid,unix) values ('%s','%s','0','%s','%d')", "通知", str, item["report_uid"].(string), TimeNow)
			_, err = db.Raw(sql).Exec()
			if err != nil {
				log("添加通知失败:[%v]", err)
				continue
			}
		}
	}

	// 设置举报已处理
	sql = ps("update talkabout_report set status=1 where tid=%d", id)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("设置举报已处理失败:[%v]", err)
		this.Rec = &Recv{5, "审核失败", nil}
		return
	}

	// 业务逻辑
	if status == 1 { //正常
		sql = ps("update talkabout set status='%d',reason='%s',report_num='0' where id='%d';", status, reason, id)
	} else {
		sql = ps("update talkabout set status='%d',reason='%s' where id='%d';", status, reason, id)
	}
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("审核说说失败:err[%v]", err)
		this.Rec = &Recv{5, ps("审核说说失败"), nil}
		return
	}

	this.Rec = &Recv{3, ps("审核说说成功!"), nil}
}

// sid,key(搜索关键字),counts(条数),status
func (this *TalkaboutController) TalkaboutSearch() {
	key := this.GetString("key")

	//检查参数
	if !CheckArg(key) {
		this.Rec = &Recv{5, "搜索关键字不能为空", nil}
		return
	}

	//业务逻辑
	var sql string = ps("select * from talkabout where `text` like '%%%s%%';", key)
	db := orm.NewOrm()
	var result []orm.Params
	nums, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("搜索说说失败:err[%v]", err)
		this.Rec = &Recv{5, ps("搜索说说失败"), nil}
		return
	}

	type RecvEx struct {
		Totals int64
		Detail interface{}
	}

	this.Rec = &Recv{3, ps("搜索说说成功"), &RecvEx{nums, result}}
}

// sid,id
func (this *TalkaboutController) TalkaboutBsDel() {
	id, _ := this.GetInt64("id")

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "说说id不能为空", nil}
		return
	}

	// 业务逻辑
	var sql = ps("delete from talkabout where id='%d';", id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("删除说说[%d]失败:err[%v]", id, err)
		this.Rec = &Recv{5, "删除说说失败", nil}
		return
	}
	this.Rec = &Recv{3, "删除说说失败", nil}
	return
}

// sid,key(搜索关键字),counts(条数),status
func (this *TalkaboutController) TalkReviewSearch() {
	key := this.GetString("key")

	//检查参数
	if !CheckArg(key) {
		this.Rec = &Recv{5, "搜索关键字不能为空", nil}
		return
	}

	//业务逻辑
	var sql string = ps("select * from talkabout_review where `content` like '%%%s%%';", key)
	db := orm.NewOrm()
	var result []orm.Params
	nums, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("搜索评论失败:err[%v]", err)
		this.Rec = &Recv{5, ps("搜索评论失败"), nil}
		return
	}

	type RecvEx struct {
		Totals int64
		Detail interface{}
	}

	this.Rec = &Recv{3, ps("搜索评论成功"), &RecvEx{nums, result}}
}

// sid
func (this *TalkaboutController) TalkReviewBsDel() {
	id, _ := this.GetInt64("id")

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "评论id不能为空", nil}
		return
	}

	// 业务逻辑
	var sql = ps("delete from `talkabout_review` where id='%d';", id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("删除评论[%d]失败:err[%v]", id, err)
		this.Rec = &Recv{5, "删除评论失败", nil}
		return
	}
	this.Rec = &Recv{3, "删除评论失败", nil}
	return
}

// sid,key(搜索关键字),counts(条数),status
func (this *TalkaboutController) TalkSecreviewSearch() {
	key := this.GetString("key")

	//检查参数
	if !CheckArg(key) {
		this.Rec = &Recv{5, "搜索关键字不能为空", nil}
		return
	}

	//业务逻辑
	var sql string = ps("select * from `talkabout_sec_review` where `content` like '%%%s%%';", key)
	db := orm.NewOrm()
	var result []orm.Params
	nums, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("搜索二级评论失败:err[%v]", err)
		this.Rec = &Recv{5, ps("搜索二级评论失败"), nil}
		return
	}

	type RecvEx struct {
		Totals int64
		Detail interface{}
	}

	this.Rec = &Recv{3, ps("搜索二级评论成功"), &RecvEx{nums, result}}
}

// sid
func (this *TalkaboutController) TalkSecreviewBsDel() {
	id, _ := this.GetInt64("id")

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "评论id不能为空", nil}
		return
	}

	// 业务逻辑
	var sql = ps("delete from `talkabout_sec_review` where id='%d';", id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("删除二级评论[%d]失败:err[%v]", id, err)
		this.Rec = &Recv{5, "删除二级评论失败", nil}
		return
	}
	this.Rec = &Recv{3, "删除二级评论失败", nil}
	return
}

// id
func (this *TalkaboutBaseController) TalkaboutView() {
	id, _ := this.GetInt32("id")

	//检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "说说id不能为空", nil}
		return
	}

	//业务逻辑
	var sql string = ps("update talkabout set viewers=viewers+1 where id='%d';", id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("查看说说失败:err[%v]", err)
		this.Rec = &Recv{5, ps("查看说说失败"), nil}
		return
	}

	this.Rec = &Recv{3, ps("查看说说成功!"), nil}
}

// sid,id,rid(举报原因id,其他传0或不传),reason(其他原因)
func (this *TalkaboutController) ReportTalk() {
	id, _ := this.GetInt32("id")
	rid, _ := this.GetInt32("rid")
	reason := this.GetString("reason")

	//检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "说说id不能为空", nil}
		return
	}

	if rid <= 0 && reason == "" {
		this.Rec = &Recv{5, "请填写举报原因.", nil}
		return
	}

	sql := ps("select uid,text from talkabout where id=%d", id)
	db := orm.NewOrm()
	var result []orm.Params
	nums, err := db.Raw(sql).Values(&result)
	if err == nil {
		if nums <= 0 {
			this.Rec = &Recv{5, "你举报的说说不存在", nil}
			return
		}
	} else {
		log("查询说说错误:[%s]", err.Error())
		this.Rec = &Recv{5, "举报失败", nil}
		return
	}

	// 添加通知
	contintro := result[0]["text"].(string)
	if len(contintro) > 50 {
		contintro = contintro[0:50]
		contintro += "..."
	}
	contintro = strings.Replace(contintro, "'", "''", -1)
	sql = ps("insert into `letter` (title,content,send_uid,recv_uid,unix) values ('%s','%s','0','%s','%d')", "通知", ps("你的说说[%s]被举报,系统将会核实.", contintro), result[0]["uid"].(string), TimeNow)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("添加通知失败:[%v]", err)
		this.Rec = &Recv{5, "举报说说失败", nil}
		return
	}

	// 业务逻辑
	sql = ps("update talkabout set report_num=report_num+1,status='3' where id='%d';", id)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("举报说说失败:err[%v]", err)
		this.Rec = &Recv{5, ps("举报说说失败"), nil}
		return
	}

	// 添加举报内容
	sql = ps("insert into `talkabout_report` (tid,report_uid,report_id,report_reason,unix) values ('%d','%d','%d','%s','%d')", id, this.User.UserId, rid, reason, TimeNow)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("举报说说失败:err[%v]", err)
		this.Rec = &Recv{5, ps("举报说说失败"), nil}
		return
	}

	this.Rec = &Recv{3, ps("举报说说成功!"), nil}
}

// sid,id,status(0-未处理,1-已处理),begidx,counts
func (this *TalkaboutController) ReportTalkQuery() {
	id, _ := this.GetInt32("id")
	begidx, _ := this.GetInt64("begidx")
	counts, _ := this.GetInt32("counts")
	status, _ := this.GetInt32("status")

	//检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "说说id不能为空", nil}
		return
	}

	//业务逻辑
	sql := ps("select * from `talkabout_report` where tid=%d and status=%d order by unix desc limit %d,%d;", id, status, begidx, counts)
	sqlc := ps("select id from `talkabout_report` where tid=%d and status=%d;", id, status)
	db := orm.NewOrm()
	var result []orm.Params
	nums, err := db.Raw(sqlc).Values(&result)
	if err != nil {
		log("查询说说举报内容失败:err[%v]", err)
		this.Rec = &Recv{5, ps("查询说说举报内容失败"), nil}
		return
	}

	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询说说举报内容失败:err[%v]", err)
		this.Rec = &Recv{5, ps("查询说说举报内容失败"), nil}
		return
	}

	type RecvEx struct {
		Totals int64
		Detail interface{}
	}

	this.Rec = &Recv{3, ps("搜索二级评论成功"), &RecvEx{nums, result}}
}

// sid,id,content,imgs(总数),img1...imgn(图片参数)
func (this *TalkaboutController) TalkaboutReview() {
	id, _ := this.GetInt64("id")
	content := this.GetString("content")
	imgs, _ := this.GetInt32("imgs")
	content = strings.Replace(content, "'", "''", -1)

	//检查参数
	if !CheckArg(id, content) {
		this.Rec = &Recv{5, "说说id或评论内容不能为空", nil}
		return
	}

	//开始业务逻辑
	db := orm.NewOrm()
	//检测评论是否存在
	sql := ps("select uid,text from talkabout where id=%d;", id)
	var result []orm.Params
	nums, err := db.Raw(sql).Values(&result)
	if err == nil {
		if nums <= 0 {
			this.Rec = &Recv{5, "你要评论的说说不存在", nil}
			return
		}
	} else {
		log("查询说说错误:[%s]", err.Error())
		this.Rec = &Recv{5, "评论失败", nil}
		return
	}
	tbuid, _ := strconv.Atoi(result[0]["uid"].(string))

	// 添加通知
	contintro := result[0]["text"].(string)
	if len(contintro) > 50 {
		contintro = contintro[0:50]
		contintro += "..."
	}
	contintro = strings.Replace(contintro, "'", "''", -1)
	sql = ps("insert into `letter` (title,content,send_uid,recv_uid,unix) values ('%s','%s','0','%d','%d')", "通知", ps("有人评论了你的说说[%s],请去查看.", contintro), tbuid, TimeNow)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("添加通知失败:[%v]", err)
		this.Rec = &Recv{5, "评论说说失败", nil}
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
					filename := GetSid()
					filename += filepath.Ext(h.Filename)
					err = this.SaveToFile(strimg, filepath.Join(conf("talkpath"), filename))
					if err != nil {
						log("文件%s保存失败:err[%v]", strimg, err)
						continue
					} else {
						imgurl += ps("https://%s/%s;", conf("talkdown"), filename)
					}
				}
			}
		}
	}

	// 添加评论
	sql = ps("insert into talkabout_review (uid,headurl,tid,content,imgurl,unix) values ('%d','%s','%d','%s','%s','%d');",
		this.User.UserId, ps("https://api.yddtv.cn:10032/cowin/head/head%d", this.User.UserId), id, content, imgurl, TimeNow)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("评论说说[%d]失败:err[%v]", id, err)
		this.Rec = &Recv{5, "评论说说失败", nil}
	} else {
		this.Rec = &Recv{3, "评论说说成功", nil}
	}

	return
}

// sid,id,begidx,counts
func (this *TalkaboutBaseController) TalkaboutReviewQuery() {
	id, _ := this.GetInt64("id")
	begidx, _ := this.GetInt64("begidx")
	counts, _ := this.GetInt64("counts")

	//检查参数
	if !CheckArg(id, counts) {
		this.Rec = &Recv{5, "参数存在空值", nil}
		return
	}

	//开始业务逻辑
	var totals int
	var results, res, rescont []orm.Params
	db := orm.NewOrm()
	_, err := db.Raw("SELECT count(id) as num FROM talkabout_review WHERE tid=? and status=1;", id).Values(&results)
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

	var sql string = ps("SELECT tr.id,tr.uid,u.nick,tr.headurl,tr.tid,tr.content,tr.imgurl,tr.unix from talkabout_review as tr,user as u WHERE tr.status=1 AND tr.tid='%d' AND tr.uid=u.id ORDER BY tr.unix desc limit %d,%d;",
		id, begidx, counts)
	_, err = db.Raw(sql).Values(&results)
	if err != nil {
		log("查询评论失败:[%v]", err)
		this.Rec = &Recv{5, "查询评论失败", nil}
	} else {
		type TagReview struct {
			Review    interface{}
			SecReview interface{}
		}
		type RecvEx struct {
			Total  int
			Detail []*TagReview
		}
		data := make([]*TagReview, len(results)) // 分配内存
		for idx := range results {
			item := results[idx]
			// 查询二级评论
			sql := ps("SELECT a.id,a.uid,u.nick,a.headurl,a.imgurl,a.tid,a.trid,a.tsrid,a.content,a.unix from talkabout_sec_review AS a,user AS u WHERE a.status=1 AND a.tid='%d' AND a.trid='%s' AND a.uid=u.id ORDER BY a.unix desc;",
				id, item["id"].(string))
			_, err = db.Raw(sql).Values(&res)
			if err != nil {
				log("查询评论失败:[%v]", err)
				continue
			} else {
				for j := range res {
					tsrid, _ := strconv.Atoi(res[j]["tsrid"].(string))
					trid, _ := strconv.Atoi(res[j]["trid"].(string))
					if tsrid > 0 { // 三级评论
						sql := ps("SELECT a.uid as srcuid,u.nick as srcnick,a.content as srccont from talkabout_sec_review AS a,user AS u WHERE a.uid=u.id AND a.id='%d';", tsrid)
						cnts, err := db.Raw(sql).Values(&rescont)
						if err != nil {
							log("查询评论失败:[%v]", rescont)
							continue
						} else if cnts > 0 {
							res[j]["srccont"] = rescont[0]["srccont"].(string)
							res[j]["srcuid"] = rescont[0]["srcuid"].(string)
							res[j]["srcnick"] = rescont[0]["srcnick"].(string)
						}
					} else {
						sql := ps("SELECT a.uid as srcuid,u.nick as srcnick,a.content as srccont from talkabout_review AS a,user AS u WHERE a.uid=u.id AND a.id='%d';", trid)
						cnts, err := db.Raw(sql).Values(&rescont)
						if err != nil {
							log("查询评论失败:[%v]", rescont)
							continue
						} else if cnts > 0 {
							res[j]["srccont"] = rescont[0]["srccont"].(string)
							res[j]["srcuid"] = rescont[0]["srcuid"].(string)
							res[j]["srcnick"] = rescont[0]["srcnick"].(string)
						}
					}
				}

				data[idx] = &TagReview{item, res}
			}
		}
		this.Rec = &Recv{3, "查询评论成功", &RecvEx{totals, data}}
	}
}

// sid,tid,id
func (this *TalkaboutController) TalkaboutReviewDel() {
	id, _ := this.GetInt64("id")
	tid, _ := this.GetInt64("tid")

	//检查参数
	if !CheckArg(id, tid) {
		this.Rec = &Recv{5, "说说id和评论id不能为空", nil}
		return
	}

	// 判断删除权限
	var results []orm.Params
	db := orm.NewOrm()
	nums, err := db.Raw("SELECT tr.uid as tuid,t.uid as uid FROM `talkabout_review` as tr,`talkabout` as t WHERE tr.tid=t.id and tr.tid=? and tr.id=?;", tid, id).Values(&results)
	if err != nil {
		log("查询评论失败:[%v]", err)
		this.Rec = &Recv{5, "查询评论失败", nil}
		return
	} else {
		if nums > 0 {
			tuid, _ := strconv.Atoi(results[0]["tuid"].(string))
			uid, _ := strconv.Atoi(results[0]["uid"].(string))
			if int64(tuid) != this.User.UserId && int64(uid) != this.User.UserId {
				this.Rec = &Recv{5, "无权删除此评论.", nil}
				return
			}
		} else {
			this.Rec = &Recv{5, "待删除评论不存在", nil}
			return
		}
	}

	// 删除评论
	var sql string = ps("DELETE from talkabout_review WHERE id=%d AND tid=%d;", id, tid)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("删除评论失败:[%v]", err)
		this.Rec = &Recv{5, "删除评论失败", nil}
	} else {
		this.Rec = &Recv{3, "删除评论成功", nil}
	}
}

// sid,id(评论id),rid(举报原因id,其他传0或不传),reason(其他原因)
func (this *TalkaboutController) ReportReview() {
	id, _ := this.GetInt32("id")
	rid, _ := this.GetInt32("rid")
	reason := this.GetString("reason")

	//检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	sql := ps("select uid,content from talkabout_review where id=%d", id)
	db := orm.NewOrm()
	var result []orm.Params
	nums, err := db.Raw(sql).Values(&result)
	if err == nil {
		if nums <= 0 {
			this.Rec = &Recv{5, "你举报的评论不存在", nil}
			return
		}
	} else {
		log("查询评论错误:[%s]", err.Error())
		this.Rec = &Recv{5, "举报失败", nil}
		return
	}

	// 添加通知
	contintro := result[0]["content"].(string)
	if len(contintro) > 50 {
		contintro = contintro[0:50]
		contintro += "..."
	}
	sql = ps("insert into `letter` (title,content,send_uid,recv_uid,unix) values ('%s','%s','0','%s','%d')", "通知", ps("你的一级评论[%s]被举报,系统将会核实.", contintro), result[0]["uid"].(string), TimeNow)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("添加通知失败:[%v]", err)
		this.Rec = &Recv{5, "举报说说失败", nil}
		return
	}

	//业务逻辑
	sql = ps("update talkabout_review set report_id='%d',report_reason='%s',status='3' where id='%d';", rid, reason, id)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("举报失败:err[%v]", err)
		this.Rec = &Recv{5, ps("举报失败"), nil}
		return
	}

	this.Rec = &Recv{3, ps("举报成功!"), nil}
}

// sid,tid(说说id),trid(评论id),id(二级评论id),content,imgs(总数),img1...imgn(图片参数)
func (this *TalkaboutController) TalkaboutSecReview() {
	tid, _ := this.GetInt64("tid")
	trid, _ := this.GetInt64("trid")
	id, _ := this.GetInt64("id")
	content := this.GetString("content")
	imgs, _ := this.GetInt32("imgs")
	content = strings.Replace(content, "'", "''", -1)

	//检查参数
	if !CheckArg(tid, trid) {
		this.Rec = &Recv{5, "说说id和评论id不能为空", nil}
		return
	}
	if !CheckArg(content) {
		this.Rec = &Recv{5, "评论内容不能为空", nil}
		return
	}

	//开始业务逻辑
	db := orm.NewOrm()
	var sql string = ""
	//检测评论是否存在
	if id > 0 {
		sql = ps("select uid,content from talkabout_sec_review where tid=%d and trid=%d and id=%d;", tid, trid, id)
	} else {
		sql = ps("select uid,content from talkabout_review where tid=%d and id=%d;", tid, trid)
	}

	var result []orm.Params
	nums, err := db.Raw(sql).Values(&result)
	if err == nil {
		if nums <= 0 {
			this.Rec = &Recv{5, "你要评论的评论不存在", nil}
			return
		}
	} else {
		log("查询评论错误:[%s]", err.Error())
		this.Rec = &Recv{5, "评论失败", nil}
		return
	}

	// 添加通知
	contintro := result[0]["content"].(string)
	if len(contintro) > 50 {
		contintro = contintro[0:50]
		contintro += "..."
	}
	sql = ps("insert into `letter` (title,content,send_uid,recv_uid,unix) values ('%s','%s','0','%s','%d')", "通知", ps("你发表的评论[%s]有了最新动态.", contintro), result[0]["uid"].(string), TimeNow)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("添加通知失败:[%v]", err)
		this.Rec = &Recv{5, "评论失败", nil}
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
					filename := GetSid()
					filename += filepath.Ext(h.Filename)
					err = this.SaveToFile(strimg, filepath.Join(conf("talkpath"), filename))
					if err != nil {
						log("文件%s保存失败:err[%v]", strimg, err)
						continue
					} else {
						imgurl += ps("https://%s/%s;", conf("talkdown"), filename)
					}
				}
			}
		}
	}

	//添加评论
	sql = ps("insert into talkabout_sec_review (uid,headurl,tid,trid,tsrid,content,imgurl,unix) values ('%d','%s','%d','%d','%d','%s','%s','%d');",
		this.User.UserId, ps("https://api.yddtv.cn:10032/cowin/head/head%d", this.User.UserId), tid, trid, id, content, imgurl, TimeNow)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("评论说说[%d]失败:err[%v]", tid, err)
		this.Rec = &Recv{5, "评论失败", nil}
	} else {
		this.Rec = &Recv{3, "评论成功", nil}
	}
	return
}

// sid,tid,trid(一级评论id),id(评论id)
func (this *TalkaboutController) TalkaboutSecReviewDel() {
	tid, _ := this.GetInt64("tid")
	trid, _ := this.GetInt64("trid")
	id, _ := this.GetInt64("id")

	//检查参数
	if !CheckArg(tid, trid, id) {
		this.Rec = &Recv{5, "说说id,评论id不能为空", nil}
		return
	}
	log("tid：[%d],trid:[%d],id:[%d]", tid, trid, id)

	//开始业务逻辑
	var results []orm.Params
	db := orm.NewOrm()
	nums, err := db.Raw("SELECT tsr.uid as tsruid,tr.uid as truid,t.uid as tuid FROM `talkabout_sec_review` as tsr,`talkabout_review` as tr,`talkabout` as t WHERE tsr.trid=tr.id and tr.tid=t.id and tsr.trid=? and tsr.tid=? and tsr.id=?;",
		trid, tid, id).Values(&results)
	if err != nil {
		log("查询评论失败:[%v]", err)
		this.Rec = &Recv{5, "查询评论失败", nil}
		return
	} else {
		if nums > 0 {
			tuid, _ := strconv.Atoi(results[0]["tuid"].(string))
			truid, _ := strconv.Atoi(results[0]["truid"].(string))
			tsruid, _ := strconv.Atoi(results[0]["tsruid"].(string))
			if int64(tuid) != this.User.UserId && int64(tsruid) != this.User.UserId && int64(truid) != this.User.UserId {
				this.Rec = &Recv{5, "无权删除此评论.", nil}
				return
			}
		} else {
			this.Rec = &Recv{5, "待删除评论不存在", nil}
			return
		}
	}

	var sql string = ps("DELETE FROM talkabout_sec_review WHERE id=%d AND tid=%d AND trid=%d;", id, tid, trid)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("删除评论失败:[%v]", err)
		this.Rec = &Recv{5, "删除评论失败", nil}
	} else {
		this.Rec = &Recv{3, "删除评论成功", nil}
	}
}

// sid,id,rid,reason
func (this *TalkaboutController) ReportSecReview() {
	id, _ := this.GetInt32("id")
	rid, _ := this.GetInt32("rid")
	reason := this.GetString("reason")

	//检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	sql := ps("select uid,content from talkabout_sec_review where id=%d", id)
	db := orm.NewOrm()
	var result []orm.Params
	nums, err := db.Raw(sql).Values(&result)
	if err == nil {
		if nums <= 0 {
			this.Rec = &Recv{5, "你举报的评论不存在", nil}
			return
		}
	} else {
		log("查询评论错误:[%s]", err.Error())
		this.Rec = &Recv{5, "举报失败", nil}
		return
	}

	// 添加通知
	contintro := result[0]["content"].(string)
	if len(contintro) > 50 {
		contintro = contintro[0:50]
		contintro += "..."
	}

	sql = ps("insert into `letter` (title,content,send_uid,recv_uid,unix) values ('%s','%s','0','%s','%d')", "通知", ps("你的二级评论[%s]被举报,系统将会核实.", contintro), result[0]["uid"].(string), TimeNow)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("添加通知失败:[%v]", err)
		this.Rec = &Recv{5, "举报说说失败", nil}
		return
	}

	//业务逻辑
	sql = ps("update talkabout_sec_review set report_id='%d',report_reason='%s',status='3' where id='%d';", rid, reason, id)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("举报失败:err[%v]", err)
		this.Rec = &Recv{5, ps("举报失败"), nil}
		return
	}

	this.Rec = &Recv{3, ps("举报成功!"), nil}
}

// sid,id
func (this *TalkaboutController) TalkaboutFans() {
	id, _ := this.GetInt64("id")

	//检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "参数存在空值", nil}
		return
	}

	//开始业务逻辑
	var results []orm.Params
	db := orm.NewOrm()
	var sql string
	num, err := db.Raw("SELECT id FROM talkabout_fans WHERE tid=? and uid=?;", id, this.User.UserId).Values(&results)
	if err == nil {
		if num == 0 {
			sql = ps("insert into talkabout_fans (uid,tid,unix) values ('%d','%d','%d');", this.User.UserId, id, TimeNow)
		} else {
			sql = ps("delete from talkabout_fans where tid='%d' and uid='%d';", id, this.User.UserId)
		}
	} else {
		this.Rec = &Recv{5, ps("[%s]点赞失败", this.User.Account), nil}
		return
	}

	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("[%s]点赞说说[%d]失败:err[%v]", this.User.Account, id, err)
		this.Rec = &Recv{5, "点赞失败", nil}
	} else {
		if num == 0 {
			this.Rec = &Recv{3, ps("[%s]点赞成功", this.User.Account), nil}
		} else {
			this.Rec = &Recv{3, ps("[%s]取消点赞成功", this.User.Account), nil}
		}
	}
}

// id,begidx,counts
func (this *TalkaboutBaseController) TalkaboutFansQuery() {
	id, _ := this.GetInt64("id")
	begidx, _ := this.GetInt64("begidx")
	counts, _ := this.GetInt64("counts")

	//检查参数
	if !CheckArg(id, counts) {
		this.Rec = &Recv{5, "参数存在空值", nil}
		return
	}

	//开始业务逻辑
	var totals string
	var results []orm.Params
	db := orm.NewOrm()
	_, err := db.Raw("SELECT count(id) as num FROM talkabout_fans WHERE tid=?;", id).Values(&results)
	if err != nil {
		log("查询点赞人数失败:[%v]", err)
		this.Rec = &Recv{5, "查询点赞人数失败", nil}
		return
	} else {
		totals = results[0]["num"].(string)
	}

	var sql string = ps("SELECT a.unix,b.id as uid,b.nick from talkabout_fans AS a,user AS b WHERE a.tid='%d' AND a.uid=b.id ORDER BY a.unix desc limit %d,%d;", id, begidx, counts)
	_, err = db.Raw(sql).Values(&results)
	if err != nil {
		log("查询点赞失败:[%v]", err)
		this.Rec = &Recv{5, "查询点赞失败", nil}
	} else {
		type RecvEx struct {
			Total  string
			Detail interface{}
		}
		this.Rec = &Recv{3, "查询点赞成功", &RecvEx{totals, results}}
	}
}

//
func (this *TalkaboutBaseController) ReportQuery() {
	//开始业务逻辑
	var results []orm.Params
	db := orm.NewOrm()

	var sql string = "SELECT * from report;"
	_, err := db.Raw(sql).Values(&results)
	if err != nil {
		log("查询失败:[%v]", err)
		this.Rec = &Recv{5, "查询失败", nil}
	} else {
		this.Rec = &Recv{3, "查询成功", results}
	}
}

// sid,text
func (this *TalkaboutController) ReportAdd() {
	text := this.GetString("text")

	//检查参数
	if !CheckArg(text) {
		this.Rec = &Recv{5, "参数存在空值", nil}
		return
	}

	var sql string = ps("insert into report (text) values ('%s');", text)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("[%s]添加举报原因失败:err[%v]", this.User.Account, err)
		this.Rec = &Recv{5, ps("[%s]添加举报原因失败", this.User.Account), nil}
		return
	}
	this.Rec = &Recv{3, ps("[%s]添加举报原因成功!", this.User.Account), nil}
}

// sid,id,text
func (this *TalkaboutController) ReportModify() {
	id, _ := this.GetInt32("id")
	text := this.GetString("text")

	//检查参数
	if !CheckArg(id, text) {
		this.Rec = &Recv{5, "参数存在空值", nil}
		return
	}

	//业务逻辑
	var sql = ps("update report set text='%s' where id=%d;", text, id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("修改举报原因失败:err[%v]", err)
		this.Rec = &Recv{5, "修改举报原因失败", nil}
		return
	}
	this.Rec = &Recv{3, "修改举报原因成功!", nil}
}

// sid,id
func (this *TalkaboutController) ReportDel() {
	id, _ := this.GetInt64("id")

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	// 业务逻辑
	var sql = ps("delete from report where id='%d';", id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("删除举报原因[%d]失败:err[%v]", id, err)
		this.Rec = &Recv{5, "删除举报原因失败", nil}
		return
	}
	this.Rec = &Recv{3, "删除举报原因成功", nil}
	return
}
