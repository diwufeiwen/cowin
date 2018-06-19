package controllers

import (
	"github.com/astaxie/beego/orm"
	"path/filepath"
	"strconv"
)

type ActivityController struct {
	OnlineController
}

// sid,title,img(封面图),linkurl,qr_code,content,status(0-草稿,1-提交审核)
func (this *ActivityController) ActivityAdd() {
	// 身份判断
	if this.User.Flag < 1 || this.User.Flag > 4 {
		this.Rec = &Recv{5, ps("仅平台管理员和经销商有次权限"), nil}
		return
	}
	title := this.GetString("title")
	linkurl := this.GetString("linkurl")
	content := this.GetString("content")
	status,_ := this.GetInt32("status")

	//检查参数
	if !CheckArg(title, linkurl, content) {
		this.Rec = &Recv{5, "标题,链接和内容均不可为空,请检查.", nil}
		return
	} 

	// 图片参数
	var qr_code, coverurl string
	f, h, err := this.GetFile("qr_code")
	if f != nil {
		defer f.Close()
		if err != nil {
			log("文件上传失败:err[%v]", err)
		} else {
			// 保存位置在 static/dealer
			filename := GetSid()
			filename += filepath.Ext(h.Filename)
			err = this.SaveToFile("qr_code", filepath.Join(conf("activitypath"), filename))
			if err != nil {
				log("文件保存失败:err[%v]", err)
			} else {
				qr_code = ps("https://%s/%s;", conf("activitydown"), filename)
			}
		}
	}

	f, h, err = this.GetFile("img")
	if f != nil {
		defer f.Close()
		if err != nil {
			log("文件上传失败:err[%v]", err)
		} else {
			// 保存位置在 static/dealer
			filename := GetSid()
			filename += filepath.Ext(h.Filename)
			err = this.SaveToFile("img", filepath.Join(conf("activitypath"), filename))
			if err != nil {
				log("文件保存失败:err[%v]", err)
			} else {
				coverurl = ps("https://%s/%s;", conf("activitydown"), filename)
			}
		}
	}

	//业务逻辑
	var sql string = ps("insert into `activity` (title,coverurl,linkurl,qr_code,content,status,uid,unix) values ('%s','%s','%s','%s','%s','%d','%d','%d')",
		title, coverurl, linkurl,qr_code,content,status,this.User.UserId, TimeNow)
	db := orm.NewOrm()
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("添加活动出错:[%v]", err)
		this.Rec = &Recv{5, ps("添加活动失败[%s]", err.Error()), nil}
		return
	}
	this.Rec = &Recv{3, "添加活动成功", nil}
}

// sid,id,title,img(封面图),linkurl,qr_code,content,status(0-草稿,1-提交审核)
func (this *ActivityController) ActivityModify() {
	id, _ := this.GetInt64("id")

	// 检查权限
	var sql string = ps("SELECT uid,`status` from `activity` where id='%d';", id)
	db := orm.NewOrm()
	var result []orm.Params
	cnts, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询activity表出错:[%v]", err)
		this.Rec = &Recv{5, "修改失败", nil}
		return
	}else if cnts<=0 {
		this.Rec = &Recv{5, "待修改活动不存在", nil}
		return
	}else{
		uid,_ := strconv.Atoi(result[0]["uid"].(string))
		if this.User.UserId != int64(uid) {
			this.Rec = &Recv{5, "仅能修改自己添加的活动", nil}
			return
		}
	}

	title := this.GetString("title")
	linkurl := this.GetString("linkurl")
	content := this.GetString("content")
	status,_ := this.GetInt32("status")
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	// 图片参数
	var qr_code, coverurl string
	f, h, err := this.GetFile("qr_code")
	if f != nil {
		defer f.Close()
		if err != nil {
			log("文件上传失败:err[%v]", err)
		} else {
			// 保存位置在 static/dealer
			filename := GetSid()
			filename += filepath.Ext(h.Filename)
			err = this.SaveToFile("qr_code", filepath.Join(conf("activitypath"), filename))
			if err != nil {
				log("文件保存失败:err[%v]", err)
			} else {
				qr_code = ps("https://%s/%s;", conf("activitydown"), filename)
			}
		}
	}

	f, h, err = this.GetFile("img")
	if f != nil {
		defer f.Close()
		if err != nil {
			log("文件上传失败:err[%v]", err)
		} else {
			// 保存位置在 static/dealer
			filename := GetSid()
			filename += filepath.Ext(h.Filename)
			err = this.SaveToFile("img", filepath.Join(conf("activitypath"), filename))
			if err != nil {
				log("文件保存失败:err[%v]", err)
			} else {
				coverurl = ps("https://%s/%s;", conf("activitydown"), filename)
			}
		}
	}

	// 业务逻辑
	sql = "update `activity` set "
	if title != "" {
		sql += ps("title='%s',", title)
	}
	if linkurl != "" {
		sql += ps("linkurl='%s',", linkurl)
	}
	if content != "" {
		sql += ps("content='%s',", content)
	}
	if qr_code != "" {
		sql += ps("qr_code='%s',", qr_code)
	}
	if coverurl != "" {
		sql += ps("coverurl='%s',", coverurl)
	}
	sql += ps("unix='%d',status='%d' where id='%d';", TimeNow, status, id)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("修改活动出错:[%v]", err)
		_, str := ChecSQLerr(err)
		this.Rec = &Recv{5, ps("修改失败:%s", str), nil}
		return
	}

	this.Rec = &Recv{3, "修改成功", nil}
	return
}

// sid,id
func (this *ActivityController) ActivityDel() {
	id, _ := this.GetInt64("id")

	// 检查权限
	var sql string = ps("SELECT uid from `activity` where id='%d';", id)
	db := orm.NewOrm()
	var result []orm.Params
	cnts, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询activity表出错:[%v]", err)
		this.Rec = &Recv{5, "删除失败", nil}
		return
	}else if cnts<=0 {
		this.Rec = &Recv{5, "待删除活动不存在", nil}
		return
	}else{
		uid,_ := strconv.Atoi(result[0]["uid"].(string))
		if this.User.UserId != int64(uid) {
			this.Rec = &Recv{5, "仅能删除自己添加的活动", nil}
			return
		}
	}

	// 业务逻辑
	sql = ps("delete from `activity` where id='%d';", id)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("删除活动出错:[%v]", id, err)
		this.Rec = &Recv{5, "删除失败", nil}
		return
	}

	this.Rec = &Recv{3, "删除成功", nil}
}

// sid,status(1-待审核,2-通过,3-未通过,0-管理员查询除草稿之外的全部,经销商查询自己添加的全部) 管理员可查所有,经销商只能查询自己添加的
func (this *ActivityController) ActivityQuery() {
	// 身份判断
	if this.User.Flag < 1 || this.User.Flag > 4 {
		this.Rec = &Recv{5, ps("仅平台管理员和经销商有次权限"), nil}
		return
	}

	status,_ := this.GetInt32("status")

	// 业务逻辑
	var sql,sqlc string = "",""
	switch this.User.Flag {
	case 1:
		if status>0 {
			sql = ps("SELECT a.*,u.nick,u.account from `activity` as a,user as u where a.status=%d order by a.unix desc;", status)
			sqlc = ps("SELECT count(id) as num from `activity` where status=%d;", status)
		}else{
			sql = "SELECT a.*,u.nick,u.account from `activity` as a,user as u where a.status>0 order by a.unix desc;"
			sqlc = "SELECT count(id) as num from `activity` where status>0;"
		}	
	default:
		if status>0 {
			sql = ps("SELECT a.*,u.nick,u.account from `activity` as a,user as u where a.status=%d and a.uid=%d order by a.unix desc;", status, this.User.UserId)
			sqlc = ps("SELECT count(id) as num from `activity` where status=%d and uid=%d;", status,this.User.UserId)
		}else{
			sql = ps("SELECT a.*,u.nick,u.account from `activity` as a,user as u where a.uid=%d order by a.unix desc;",this.User.UserId)
			sqlc = ps("SELECT count(id) as num from `activity` where uid=%d;", this.User.UserId)
		}
	}
	

	db := orm.NewOrm()
	var result []orm.Params
	var totals int = 0
	_, err := db.Raw(sqlc).Values(&result)
	if err != nil {
		log("查询activity表出错:[%v]", err)
		this.Rec = &Recv{5, "查询失败", nil}
		return
	}else{
		totals,_ = strconv.Atoi(result[0]["num"].(string))
	}

	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询activity表出错:[%v]", err)
		this.Rec = &Recv{5, "查询失败", nil}
		return
	}

	type RecvEx struct{
		Total int
		Detail interface{}
	}
	this.Rec = &Recv{3, "查询角色成功", &RecvEx{totals,result}}
	return
}

// sid,id,status(2-通过,3-未通过),reason
func (this *ActivityController) ActivityCheck() {
	// 身份判断
	if this.User.Flag != 1 {
		this.Rec = &Recv{5, ps("仅平台管理员有此权限"), nil}
		return
	}

	status,_ := this.GetInt32("status")
	id,_ := this.GetInt64("id")
	reason := this.GetString("reason")
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	if status==3 && reason=="" {
		this.Rec = &Recv{5, ps("不通过原因不能为空"), nil}
		return
	}

	// 业务逻辑
	var sql string = ps("update `activity` set status='%d',reason='%s' where id=%d;", status,reason,id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("审核活动出错:[%v]", err)
		this.Rec = &Recv{5, "审核失败", nil}
		return
	}

	this.Rec = &Recv{3, "审核成功", nil}
	return
}
