package controllers

import (
	"github.com/astaxie/beego/orm"
	"path/filepath"
)

type CarouselController struct {
	OnlineController
}

type CarouselBaseController struct {
	BaseController
}

// scene(1-商城,2-流转,3-首页)
func (this *CarouselBaseController) CarouselQuery() {
	scene, _ := this.GetInt32("scene")

	var sql string = ""
	if scene > 0 {
		sql = ps("SELECT * FROM carousel WHERE `scene`=%d;", scene)
	} else {
		sql = "SELECT * FROM carousel;"
	}

	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询失败:err[%v]", err)
		this.Rec = &Recv{5, "查询失败", nil}
		return
	}

	this.Rec = &Recv{3, "查询成功", result}
	return
}

// sid,text,img,idx(顺序,即为第几页显示,从1开始),gotourl(跳转网页),scene
func (this *CarouselController) CarouselAdd() {
	text := this.GetString("text")
	idx, _ := this.GetInt32("idx")
	gotourl := this.GetString("gotourl")
	scene, _ := this.GetInt32("scene")

	//检查参数
	if !CheckArg(idx, scene) {
		this.Rec = &Recv{5, "序号和平台号不能为空", nil}
		return
	}

	// 检测并存储图像
	f, h, err := this.GetFile("img")
	var imgurl string = ""
	if f == nil { //空文件
		this.Rec = &Recv{5, "图片为空,请检查", nil}
		return
	} else {
		defer f.Close()
		if err != nil {
			log("上传文件失败:err[%v]", err)
			this.Rec = &Recv{5, "文件传输失败,请检查", nil}
			return
		} else {
			// 文件名
			filename := GetSid()
			filename += filepath.Ext(h.Filename)

			// 保存位置在 static/carousel
			err = this.SaveToFile("img", filepath.Join(conf("carouselpath"), filename))
			if err != nil {
				log("文件保存失败:err[%v]", err)
				this.Rec = &Recv{5, "文件传输失败,请重试", nil}
				return
			} else {
				imgurl = ps("https://%s/%s", conf("carouseldown"), filename)
			}
		}
	}

	var sql string = ps("insert into carousel (text,imgurl,gotourl,idx,scene,unix) values ('%s','%s','%s','%d','%d','%d')", text, imgurl, gotourl, idx, scene, TimeNow)
	db := orm.NewOrm()
	_, err = db.Raw(sql).Exec()
	if err != nil {
		_, strerr := ChecSQLerr(err)
		log("添加轮播图失败:[%v]", err)
		this.Rec = &Recv{5, ps("添加轮播图失败:[%s]", strerr), nil}
		return
	}
	this.Rec = &Recv{3, "添加轮播图成功", nil}
}

// sid,id,text,img,idx,gotourl(跳转网页),scene
func (this *CarouselController) CarouselModify() {
	id, _ := this.GetInt32("id")
	text := this.GetString("text")
	idx, _ := this.GetInt32("idx")
	scene, _ := this.GetInt32("scene")
	gotourl := this.GetString("gotourl")

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	// 检测并存储图像
	f, h, err := this.GetFile("img")
	var imgurl string = ""
	if f == nil { //空文件
		imgurl = ""
	} else {
		defer f.Close()
		if err != nil {
			log("上传文件失败:err[%v]", err)
			this.Rec = &Recv{5, "文件传输失败,请检查", nil}
			return
		} else {
			// 文件名
			filename := GetSid()
			filename += filepath.Ext(h.Filename)

			// 保存位置在 static/carousel
			err = this.SaveToFile("img", filepath.Join(conf("carouselpath"), filename))
			if err != nil {
				log("文件保存失败:err[%v]", err)
				this.Rec = &Recv{5, "文件保存失败,请重试", nil}
				return
			} else {
				imgurl = ps("https://%s/%s", conf("carouseldown"), filename)
			}
		}
	}

	// 业务逻辑
	var sql = "update carousel set "
	if text != "" {
		sql += ps("text='%s',", text)
	}
	if imgurl != "" {
		sql += ps("imgurl='%s',", imgurl)
	}
	if gotourl != "" {
		sql += ps("gotourl='%s',", gotourl)
	}
	if idx > 0 {
		sql += ps("idx='%d',", idx)
	}
	if scene > 0 {
		sql += ps("scene='%d',", scene)
	}
	sql += ps("unix='%d' where id=%d", TimeNow, id)
	db := orm.NewOrm()
	_, err = db.Raw(sql).Exec()
	if err != nil {
		_, strerr := ChecSQLerr(err)
		log("修改轮播图失败:[%v]", err)
		this.Rec = &Recv{5, ps("修改轮播图失败:[%s]", strerr), nil}
		return
	}
	this.Rec = &Recv{3, ps("[%s]修改轮播图成功!", this.User.Account), nil}
}

// sid,id
func (this *CarouselController) CarouselDel() {
	id, _ := this.GetInt64("id")

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	// 业务逻辑
	var sql = ps("delete from carousel where id='%d';", id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("删除轮播图[%d]失败:err[%v]", id, err)
		this.Rec = &Recv{5, "删除轮播图失败", nil}
		return
	}
	this.Rec = &Recv{3, "删除轮播图成功", nil}
}
