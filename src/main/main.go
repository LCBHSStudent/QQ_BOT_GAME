package main

import (
	"database/sql"
	"log"
	"strconv"
	"strings"
	"sync"
	
	"CQApp/src/dbTransition"
	"CQApp/src/homo"
	"CQApp/src/lottery"
	"github.com/catsworld/qq-bot-api"
	_ "github.com/go-sql-driver/mysql"
)

var bot  *qqbotapi.BotAPI
var db   *sql.DB
var keyWords = [9]string {
	"转蛋单抽", "转蛋十连", "转蛋奖池", "HOMOSPACE", "编辑HOMO", "我的转蛋券", "HOMO图鉴", "准备对战", "Document",
}


//const Host = "39.106.219.180"
const Host = "127.0.0.1"

const (
	userName = "root"
	password = "password"
	//ip       = "39.106.219.180"
	ip       = "127.0.0.1"
	port     = "3306"
	dbName   = "homospace"
)


var ChanList  []chan qqbotapi.Update
var ChanMutex sync.RWMutex

func main() {
	var err error
	bot, err = qqbotapi.NewBotAPI("", "ws://"+Host+":6700", "CQHTTP_SECRET")
	if err != nil {
		log.Fatal(err)
	}
	bot.Debug = true
	
	conf := qqbotapi.NewUpdate(0)
	conf.PreloadUserInfo = true
	updates, err := bot.GetUpdatesChan(conf)
	
	path := strings.Join([]string{
		userName, ":", password, "@tcp(",ip, ":", port, ")/", dbName, "?charset=utf8"},
	"")
		db, err = sql.Open("mysql", path)
	//	"root:password@/homospace?charset=utf8")
	//连接数据库，格式 用户名：密码@/数据库名？charset=编码方式
	if err != nil {
		log.Println(err)
		panic("open database-MySql failed.")
	}
	defer db.Close()
	
	dbTransition.Init(db)
	lottery.Init(bot)
	homo.Init(bot)
	
	for update := range updates {
		// 向下一级分发消息
		for _, ch := range ChanList {
			ch <- update
		}
		// 判断消息属性
		if update.Message == nil || update.MessageType != "group" {
			continue
		}
		log.Println(update.Message)
		// detect is const operation
		var flag int = -1
		for index, str := range keyWords {
			if str == update.Message.Text {
				flag = index
				break
			}
		}
		switch flag {
		case 0:
			lottery.SingleDraw(update)
			break
		case 1:
			lottery.MultiDraw(update)
			break
		case 2:
			lottery.ShowDrawPool(update.GroupID)
			break
		case 3:
			homo.DisplayAsset(update)
			break
		case 4:
			addMissionChan := make(chan struct{}, 1)
			go func() {
				homo.EditHomo(
					updates, addMissionChan,
					update.Message.From.ID,
					update.GroupID,
				)
				lottery.GetHomoList()
			}()
			<-addMissionChan
			break
		case 5:
			lottery.ShowTicketCnt(
				update.Message.From.ID,
				update.GroupID,
			)
			break
		case 6:
			homo.DisplayAllHomo(update.GroupID)
			break
		case 7:
			go func() {
				updateChan := make(chan qqbotapi.Update, 1)
				
				ChanMutex.Lock()
				pos := len(ChanList)
				ChanList = append(ChanList, updateChan)
				ChanMutex.Unlock()
				
				homo.Prepare4Battle(
					updateChan, //addMissionChan,
					update.Message.From.ID,
					update.GroupID,
				)
				close(updateChan)
				ChanMutex.Lock()
				ChanList = append(ChanList[:pos], ChanList[pos+1:len(ChanList)]...)
				ChanMutex.Unlock()
			}()
			break
		case 8:
			PrintHelpInfo(update.GroupID)
			break
		default:
			handleMsg(update)
			break
		}
	}
}

func handleMsg(update qqbotapi.Update) {
	if update.GroupID == 930378083 {
		go func() {
			dbTransition.AddUser(update.Message.From.ID)
			if !dbTransition.DetectDailyLimit(update.Message.From.ID) {
				dbTransition.IncreaseUserTicket(update.Message.From.ID, 1)
			}
		}()
	}
	list := strings.Split(update.Message.Text, " ")
	if len(list) == 2 && list[0] == "查询" {
		bot.NewMessage(update.GroupID, "group").
			Text(dbTransition.DisplaySingleHomoInfo(list[1])).Send()
	}
}

func PrintHelpInfo(groupID int64) {
	msg := bot.NewMessage(groupID, "group").Text("")
	for index, api := range keyWords {
		msg = msg.Text(strconv.Itoa(index+1)+"."+api)
		msg = msg.NewLine()
	}
	msg.Text("10.查询 角色名")
	msg.Send()
}