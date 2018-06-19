package main

import (
	_ "cowin/routers"
	"github.com/astaxie/beego"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	beego.Run()
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		s := <-c
		switch s {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			break
		default:
			break
		}
	}
}
