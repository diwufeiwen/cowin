package controllers

import (
	"sync"
	"time"
)

var UserSessions *Buckets

type Buckets struct {
	bLock   sync.RWMutex
	bucket  map[string]*Loginuser
	account map[string]*Loginuser
}

func init() {
	UserSessions = new(Buckets)
	UserSessions.bucket = make(map[string]*Loginuser)
	UserSessions.account = make(map[string]*Loginuser)

	ticker := time.NewTicker(time.Hour * 1)
	go func() {
		for range ticker.C {
			if time.Now().Format("15") == "00" {
				UserSessions.bLock.Lock()
				for k, v := range UserSessions.bucket {
					if TimeNow-v.LastTime > SidLife {
						delete(UserSessions.bucket, k)
						delete(UserSessions.account, k)
					}
				}
				UserSessions.bLock.Unlock()
				log("Session clear 24 hour 00:00:00")
			}
		}
	}()
}

func (b *Buckets) Adduser(usr *Loginuser) {
	if uu, ok := b.account[usr.Account]; ok {
		log("Account:[%s]已经在线,另一端将会下线", usr.Account)
		b.Deluser(uu.SessionId)
	}
	log("Account:[%s] add in UserSessions", usr.Account)
	b.bLock.Lock()
	b.bucket[usr.SessionId] = usr
	b.account[usr.Account] = usr
	b.bLock.Unlock()
	return
}

func (b *Buckets) Deluser(sion string) {
	if uu, ok := b.bucket[sion]; ok {
		b.bLock.Lock()
		delete(b.bucket, sion)
		delete(b.account, uu.Account)
		b.bLock.Unlock()
	} else {
		log("SessionId:[%s]用户不在线", sion)
	}
	return
}

func (b *Buckets) QueryloginS(sion string) (usr *Loginuser, ok bool) {
	if usr, ok = b.bucket[sion]; ok {
		log("SessionId:[%s]用户在线", sion)
	} else {
		log("SessionId:[%s]用户不在线", sion)
	}
	return
}

func (b *Buckets) QueryloginA(account string) (usr *Loginuser, ok bool) {
	if usr, ok = b.account[account]; ok {
		log("SessionId:[%s]用户在线", account)
	} else {
		log("SessionId:[%s]用户不在线", account)
	}
	return
}

func (b *Buckets) QueryloginB(sion string) bool {
	if _, ok := b.bucket[sion]; ok {
		return ok
	} else {
		return false
	}
}
