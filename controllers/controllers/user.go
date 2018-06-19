package controllers

import (
	"github.com/astaxie/beego/orm"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type UserController struct {
	OnlineController
}

type AppformController struct {
	BaseController
}

// 查询在线用户信息(这个有点莫名其妙,就是更新下内存中自己账户信息)
func (this *UserController) QueryUserInfo() {
	this.User.LastTime = TimeNow
	type TagUser struct {
		SessionId string
		UserId    int64
		Account   string
		Nick      string
		Intro     string
		Flag      int64
		Level     int64
		Wallet    float64
		Src       string
		City      string
		DealerAcc string
		Ptid      int32
		Phone     string
		Address   string
		Realname  string
		LastTime  int64
	}

	var user *TagUser
	user = &TagUser{
		SessionId: this.User.SessionId,
		UserId:    this.User.UserId,
		Account:   this.User.Account,
		Nick:      this.User.Nick,
		Intro:     this.User.Intro,
		Flag:      this.User.Flag,
		Level:     this.User.Level,
		Wallet:    this.User.Wallet,
		Src:       this.User.Src,
		City:      this.User.City,
		DealerAcc: this.User.DealerAcc,
		Ptid:      this.User.Ptid,
		Phone:     this.User.Phone,
		Address:   this.User.Address,
		Realname:  this.User.Realname,
		LastTime:  this.User.LastTime,
	}

	this.Rec = &Recv{3, "查询用户信息成功", user}
	return
}

// 直接返回内存中当前用户信息
func (this *UserController) LogCheck() {
	this.User.LastTime = TimeNow
	type User struct {
		SessionId string
		UserId    int64
		Account   string
		Nick      string
		Intro     string
		Flag      int64
		Level     int64
		Wallet    float64
		Src       string
		City      string
		LastTime  int64
	}

	var user *User
	user = &User{
		SessionId: this.User.SessionId,
		UserId:    this.User.UserId,
		Account:   this.User.Account,
		Nick:      this.User.Nick,
		Intro:     this.User.Intro,
		Flag:      this.User.Flag,
		Level:     this.User.Level,
		Wallet:    this.User.Wallet,
		Src:       this.User.Src,
		City:      this.User.City,
		LastTime:  this.User.LastTime,
	}
	this.Rec = &Recv{-1, "", user}
}

func (this *UserController) UploadHead() {
	f, _, err := this.GetFile("file")
	if f == nil {
		this.Rec = &Recv{5, "上传头像文件失败,上传图片为空", nil}
		return
	}
	defer f.Close()
	if err != nil {
		log("head上传文件传输失败:err[%v]", err)
		this.Rec = &Recv{5, "上传头像文件失败,请重新尝试", nil}
	} else {
		/***************开始业务逻辑****************/
		// 保存位置在 static/head,没有文件夹要先创建
		filename := ps("head%d", this.User.UserId)
		err = this.SaveToFile("file", filepath.Join(conf("headpath"), filename))
		if err != nil {
			log("head上传文件保存失败:err[%v]", err)
			this.Rec = &Recv{5, "上传头像文件失败", nil}
		} else {
			this.Rec = &Recv{3, "success", ps("https://%s/head%d", conf("headdown"), this.User.UserId)}
		}
	}
}

// 上传普通文件
func (this *UserController) UploadTmpFile() {
	/***************参数校验********************/
	f, h, err := this.GetFile("file")
	if f == nil {
		this.Rec = &Recv{5, "上传文件失败,上传图片为空", nil}
		return
	}
	defer f.Close()

	if err != nil {
		log("上传文件传输失败:err[%v]", err)
		this.Rec = &Recv{5, "上传文件失败,请检查文件是否合法", nil}
		return
	} else {
		// 保存位置在 static/upload,没有文件夹要先创建
		filename := GetSid()
		filename += filepath.Ext(h.Filename)
		date := time.Now().Format("20060102")
		dirpath := filepath.Join(conf("tmppath"), date)
		err = os.MkdirAll(dirpath, os.ModePerm)
		if err != nil {
			log("创建文件夹失败err[%v]", err)
		}
		err = this.SaveToFile("file", filepath.Join(dirpath, filename))
		if err != nil {
			log("上传文件保存失败:err[%v]", err)
			this.Rec = &Recv{5, "上传文件失败", nil}
		} else {
			downdir := filepath.Join(conf("tmpdown"), date)
			url := "https://" + filepath.Join(downdir, filename) //windows下路径是"\...\..."
			url = strings.Replace(url, "\\", "/", -1)
			this.Rec = &Recv{3, "success", url}
		}
	}
}

// sid,file(文件:*.apk或*.exe),img(二维码),platform(0-电脑端行情,1-电脑端实盘,2-android.,3-ios)
func (this *UserController) AppFileAdd() {
	platform, _ := this.GetInt32("platform")

	// 检测上传文件
	var imgurl, fileurl string
	f, h, err := this.GetFile("img")
	if f == nil { //空文件
		imgurl = ""
		log("上传二维码文件为空")
	} else {
		defer f.Close()
		if err != nil {
			log("上传文件传输失败:[%v]", err)
			this.Rec = &Recv{5, "文件传输失败,请检查", nil}
			return
		} else {
			// 文件名
			filename := GetSid()
			filename += filepath.Ext(h.Filename)

			err = this.SaveToFile("img", filepath.Join(conf("appfilepath"), filename))
			if err != nil {
				log("二维码保存失败:err[%v]", err)
				this.Rec = &Recv{5, "二维码传输失败,请重试", nil}
				return
			} else {
				imgurl = ps("https://%s/%s", conf("appfiledown"), filename)
			}
		}
	}

	f, h, err = this.GetFile("file")
	if f == nil { //空文件
		fileurl = ""
		log("上传压缩包文件为空")
	} else {
		defer f.Close()
		if err != nil {
			log("上传文件传输失败:[%v]", err)
			this.Rec = &Recv{5, "文件传输失败,请检查", nil}
			return
		} else {
			// 文件名
			var filename string = ""
			switch platform {
			case 0:
				filename = "PC-Market"
			case 1:
				filename = "PC-RealQuotes"
			}
			if CheckArg(filename) {
				filename += filepath.Ext(h.Filename)
				err = this.SaveToFile("file", filepath.Join(conf("appfilepath"), filename))
				if err != nil {
					log("压缩包保存失败:err[%v]", err)
					this.Rec = &Recv{5, "压缩包传输失败,请重试", nil}
					return
				} else {
					fileurl = ps("https://%s/%s", conf("appfiledown"), filename)
				}
			}
		}
	}

	var result []orm.Params
	db := orm.NewOrm()
	var sql string
	_, err = db.Raw("SELECT count(id) as num FROM app_form WHERE platform=?;", platform).Values(&result)
	if err == nil {
		if result[0]["num"].(string) == "0" {
			sql = ps("insert into app_form (fileurl,imgurl,platform,unix) values ('%s','%s','%d','%d')", fileurl, imgurl, platform, TimeNow)
		} else {
			sql = ps("update app_form set fileurl='%s',imgurl='%s',unix='%d' where platform=%d;", fileurl, imgurl, TimeNow, platform)
		}
	} else {
		this.Rec = &Recv{5, ps("[%s]添加应用失败", this.User.Account), nil}
		return
	}
	log("%s", sql)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("[%s]添加应用失败:err[%v]", this.User.Account, err)
		this.Rec = &Recv{5, ps("[%s]添加应用失败", this.User.Account), nil}
		return
	}
	this.Rec = &Recv{3, ps("[%s]添加应用成功!", this.User.Account), nil}
}

func (this *AppformController) AppformQuery() {
	//业务逻辑
	var sql string = "select * from app_form;"
	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		this.Rec = &Recv{5, "查询客户端下载链接失败", nil}
		return
	}

	this.Rec = &Recv{3, "查询客户端下载链接成功", result}
}
