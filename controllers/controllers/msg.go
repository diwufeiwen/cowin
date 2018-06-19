package controllers

import (
	"github.com/astaxie/beego/orm"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type MsgController struct {
	OnlineController
}

type MsgBaseController struct {
	BaseController
}

// sid,title,img(图片),content(内容),audience(受众),type(0-存为草稿,1-发送)
func (this *MsgController) MsgAdd() {
	title := this.GetString("title")
	content := this.GetString("content")
	audience, _ := this.GetInt32("audience")
	types, _ := this.GetInt32("type")

	//检查参数
	if !CheckArg(title, content) {
		this.Rec = &Recv{5, "参数存在空值", nil}
		return
	}

	// 存储图像
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

	var sql string = ps("insert into msg (title,content,imgurl,audience,status,unix) values ('%s','%s','%s','%d','%d','%d');",
		title, content, imgurl, audience, types, TimeNow)
	db := orm.NewOrm()
	_, err = db.Raw(sql).Exec()
	if err != nil {
		_, str := ChecSQLerr(err)
		log("[%s]添加消息失败:[%v]", this.User.Account, str)
		this.Rec = &Recv{5, "添加消息失败", nil}
		return
	}
	this.Rec = &Recv{3, "添加消息成功", nil}
}

// sid,id,title,img(图片),content(内容),audience(受众)
func (this *MsgController) MsgModify() {
	id, _ := this.GetInt32("id")
	title := this.GetString("title")
	content := this.GetString("content")
	audience, _ := this.GetInt32("audience")

	//检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "参数存在空值", nil}
		return
	}

	// 存储图像
	f, h, err := this.GetFile("img")
	var imgurl string = ""
	if f == nil { //空文件
		imgurl = ""
	} else {
		defer f.Close()
		if err != nil {
			log("上传文件传输失败:err[%v]", err)
			this.Rec = &Recv{5, "文件传输失败,请检查.", nil}
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
			err = this.SaveToFile("img", filepath.Join(dirpath, filename))
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

	//业务逻辑
	var sql = "update `msg` set "
	if title != "" {
		sql += ps("title='%s',", title)
	}
	if content != "" {
		sql += ps("content='%s',", content)
	}
	if audience > 0 {
		sql += ps("audience='%d',", audience)
	}
	if imgurl != "" {
		sql += ps("imgurl='%s',", imgurl)
	}

	sql += ps("unix='%d' where id=%d and status=0;", TimeNow, id)
	db := orm.NewOrm()
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("修改消息失败:[%v]", err)
		this.Rec = &Recv{5, "修改消息失败", nil}
		return
	}
	this.Rec = &Recv{3, "修改消息成功", nil}
}

// sid,id
func (this *MsgController) MsgDel() {
	id, _ := this.GetInt64("id")

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "消息id不能为空", nil}
		return
	}

	// 业务逻辑
	var sql = ps("delete from msg where id='%d';", id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("删除消息[%d]失败:err[%v]", id, err)
		this.Rec = &Recv{5, "删除消息失败", nil}
		return
	}
	this.Rec = &Recv{3, "删除消息成功", nil}
	return
}

// sid,id
func (this *MsgController) MsgSend() {
	id, _ := this.GetInt64("id")

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "消息id不能为空", nil}
		return
	}

	// 业务逻辑
	var sql = ps("update msg set status=1 where id='%d';", id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("发送消息[%d]失败:err[%v]", id, err)
		this.Rec = &Recv{5, "发送消息失败", nil}
		return
	}
	this.Rec = &Recv{3, "发送消息成功", nil}
	return
}

// sid,begidx(开始索引),counts(条数),flag(0-前端,1-后端),status(-1-全部;0-草稿;1-已发送,此参数仅后端有意义)
func (this *MsgController) MsgQuery() {
	begidx, _ := this.GetInt32("begidx")
	counts, _ := this.GetInt32("counts")
	flag, _ := this.GetInt32("flag")
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
		sql, sqlc string = "", ""
		totals    int    = 0
	)

	switch flag {
	case 0:
		sql = ps("select id,title,content,imgurl,unix from msg where status=1 order by unix desc limit %d,%d;", begidx, counts)
		sqlc = "select count(id) as num from msg where status=1;"
	case 1:
		sql = "select * from msg"
		sqlc = "select count(id) as num from msg"
		if status >= 0 {
			sql += ps(" where status=%d", status)
			sqlc += ps(" where status=%d", status)
		}
		sql += ps(" order by unix desc limit %d,%d;", begidx, counts)
		sqlc += ";"
	}

	_, err := db.Raw(sqlc).Values(&result)
	if err == nil {
		totals, _ = strconv.Atoi(result[0]["num"].(string))
	}

	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询消息失败:[%v]", err)
		this.Rec = &Recv{5, ps("查询消息失败"), nil}
		return
	}

	type RecvEx struct {
		Total  int
		Detail interface{}
	}
	this.Rec = &Recv{3, ps("查询消息成功!"), &RecvEx{totals, result}}
}
