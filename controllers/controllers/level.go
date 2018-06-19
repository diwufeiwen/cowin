package controllers

import (
	"github.com/astaxie/beego/orm"
	"path/filepath"
)

type LevelController struct {
	OnlineController
}

type LevelBaseController struct {
	BaseController
}

//
func (this *LevelBaseController) QueryLevel() {
	// 业务逻辑
	var sql string = "SELECT * FROM `user_level`;"

	db := orm.NewOrm()
	db.Using("default")
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询数据库失败:err[%v]", err)
		this.Rec = &Recv{5, "查询失败", nil}
		return
	}

	this.Rec = &Recv{3, "查询成功!", result}
}

// sid,fid,lid,role_name,roleimg
func (this *LevelController) AddLevel() {
	role_name := this.GetString("role_name")
	fid, _ := this.GetInt64("fid")
	lid, _ := this.GetInt64("lid")

	// 参数检测
	if !CheckArg(role_name) {
		this.Rec = &Recv{5, "角色名不能为空", nil}
		return
	}

	// 存储图像
	var role_css string = ""
	f, h, err := this.GetFile("roleimg")
	if f == nil { //空文件
		role_css = ""
	} else {
		defer f.Close()
		if err != nil {
			log("上传文件传输失败:err[%v]", err)
			this.Rec = &Recv{5, "文件传输失败,请检查", nil}
			return
		} else {
			filename := GetSid()
			filename += filepath.Ext(h.Filename)
			err = this.SaveToFile("roleimg", filepath.Join(conf("levelpath"), filename))
			if err != nil {
				log("上传文件保存失败:err[%v]", err)
				this.Rec = &Recv{5, "上传文件失败", nil}
				return
			} else {
				role_css = ps("https://%s/%s", conf("leveldown"), filename)
			}
		}
	}

	// 业务逻辑
	var sql string = ps("INSERT INTO `user_level` (fid,lid,role_name,role_css) values('%d','%d','%s','%s');", fid, lid, role_name, role_css)
	db := orm.NewOrm()
	db.Using("default")
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("添加等级失败:err[%v]", err)
		this.Rec = &Recv{5, "添加等级失败", nil}
		return
	}

	this.Rec = &Recv{3, "添加等级成功!", nil}
}

// sid,id
func (this *LevelController) DelLevel() {
	id, _ := this.GetInt64("id")

	// 参数检测
	if !CheckArg(id) {
		this.Rec = &Recv{5, "等级id不能为空", nil}
		return
	}

	// 业务逻辑
	var sql string = ps("DELETE FROM `user_level` WHERE id='%d';", id)
	db := orm.NewOrm()
	db.Using("default")
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("删除等级失败:err[%v]", err)
		this.Rec = &Recv{5, "删除等级失败", nil}
		return
	}

	this.Rec = &Recv{3, "删除等级成功!", nil}
}

// sid,id,role_name,roleimg
func (this *LevelController) ModifyLevel() {
	role_name := this.GetString("role_name")
	id, _ := this.GetInt64("id")

	// 参数检测
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	// 存储图像
	var role_css string = ""
	f, h, err := this.GetFile("roleimg")
	if f == nil { //空文件
		role_css = ""
	} else {
		defer f.Close()
		if err != nil {
			log("上传文件传输失败:err[%v]", err)
			this.Rec = &Recv{5, "文件传输失败,请检查", nil}
			return
		} else {
			filename := GetSid()
			filename += filepath.Ext(h.Filename)
			err = this.SaveToFile("roleimg", filepath.Join(conf("levelpath"), filename))
			if err != nil {
				log("上传文件保存失败:err[%v]", err)
				this.Rec = &Recv{5, "上传文件失败", nil}
				return
			} else {
				role_css = ps("https://%s/%s", conf("leveldown"), filename)
			}
		}
	}

	var sql = "update user_level set "
	if role_name != "" {
		sql += ps("role_name='%s',", role_name)
	}
	if role_css != "" {
		sql += ps("role_css='%s',", role_css)
	}
	sql = sql[0 : len(sql)-1]
	sql += ps(" where id='%d'", id)
	db := orm.NewOrm()
	db.Using("default")
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("修改等级失败:err[%v]", err)
		this.Rec = &Recv{5, "修改等级失败", nil}
		return
	}
	this.Rec = &Recv{3, "修改等级成功", nil}
}
