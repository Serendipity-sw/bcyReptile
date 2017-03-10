package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/guotie/config"
	"github.com/smtc/glog"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

var (
	debugFlag        = flag.Bool("d", false, "debug mode")
	initUrlPath      string //初始化目录地址
	urlHost          string //抓取网站域名
	createFilePath   string //文件生成路径
	isFirstStart     = true //是否为第一次启动
	cosPageUrlLock   sync.RWMutex
	cosPageUrl       []string
	threadNumberLock sync.RWMutex
	threadNumber     = 0
	threadRun        = make(chan bool, 1)
	isRun            int
)

/**
日志文件加载
创建人：邵炜
输入参数：调试模式是否为
*/
func logInit(debug bool) {
	var option = make(map[string]interface{})

	option["typ"] = "file"
	if debug {
		glog.InitLogger(glog.DEV, option)
	} else {
		glog.InitLogger(glog.PRO, option)
	}
}

/**
主函数
创建人：邵炜
创建时间：2017年03月10日10:50:38
*/
func main() {
	flag.Parse()
	config.ReadCfg("./config.json")
	readConfig()
	logInit(*debugFlag)
	urlObj, err := url.Parse(initUrlPath)
	if err != nil {
		glog.Error("main urlParse")
	}
	urlHost = urlObj.Host

	threadNumber++
	getUrlPage(initUrlPath)

	select {
	case <-threadRun:
		threadNumberLock.RLock()
		isRun = threadNumber
		threadNumberLock.RUnlock()
		if isRun == 0 {
			//文件输出处理代码 待完善
		}
		break
	}
}

/**
配置文件解析
创建人：邵炜
创建时间：2017年03月10日14:15:53
*/
func readConfig() {
	initUrlPath = config.GetStringMust("initUrlPath")
	createFilePath = config.GetStringMust("createFilePath")
}

/**
每一页网页源码处理
创建人：邵炜
创建时间：2017年03月10日11:19:51
输入参数： bodyStr 网页html源码
*/
func pageProcess(bodyByte *[]byte) {
	defer func() {
		threadNumberLock.Lock()
		threadNumber--
		threadNumberLock.Unlock()
		threadRun <- true
	}()
	var (
		href      string
		arrayList []string
		bo        bool
	)
	docQuery, err := goquery.NewDocumentFromReader(bytes.NewReader(*bodyByte))
	if err != nil {
		glog.Error("pageProcess NewDocumentFromReader err! bodyByte: %s err: %s \n", string(*bodyByte), err.Error())
		return
	}
	if isFirstStart {
		isFirstStart = false
		pageNumberProcess(docQuery)
	}
	docQuery.Find(".grid__inner.gallery.gallery--5 ._box.imageCard.pd10>a").Each(func(i int, elem *goquery.Selection) {
		href, bo = elem.Attr("href")
		if bo {
			arrayList = append(arrayList, fmt.Sprintf("%s%s", urlHost, href))
		}
	})
	cosPageUrlLock.Lock()
	cosPageUrl = append(cosPageUrl, arrayList...)
	cosPageUrlLock.Unlock()
}

/**
页码处理
创建人：邵炜
创建时间：2017年03月10日14:27:12
输入参数：html文档对象
*/
func pageNumberProcess(docQuery *goquery.Document) {
	pageNumberHref, bo := docQuery.Find(".pager a").Last().Attr("href")
	if bo {
		pageNumberArray := strings.Split(pageNumberHref, "&p=")
		if len(pageNumberArray) == 2 {
			pageNumber, err := strconv.Atoi(pageNumberArray[1])
			if err != nil {
				glog.Error("pageNumberProcess string can't convert int! pageNumberArray: %v err: %s\n", pageNumberArray, err.Error())
				return
			}
			pageNumberUrlProcess(pageNumber)
		}
	}
}

/**
获取网页源码
创建人：邵炜
创建时间：2017年03月10日11:20:35
输入参数： 网页地址
*/
func getUrlPage(urlPathStr string) {
	httpClient, err := http.Get(urlPathStr)
	if err != nil {
		glog.Error("getUrlPage http get send err! urlPathStr: %s err: %s \n", urlPathStr, err.Error())
		return
	}
	defer httpClient.Body.Close()
	bodyByte, err := ioutil.ReadAll(httpClient.Body)
	if err != nil {
		glog.Error("getUrlPage read body err! urlPathStr: %s err: %s \n", urlPathStr, err.Error())
		return
	}
	go pageProcess(&bodyByte)
}

/**
网页每页url生成处理
创建人：邵炜
创建时间：2017年03月10日14:53:24
*/
func pageNumberUrlProcess(pageNumber int) {
	var urlPathStr string
	if strings.LastIndex(initUrlPath, "&p=") > 0 {
		urlPathStr = initUrlPath[:strings.LastIndex(initUrlPath, "&p=")]
	}
	for pageNumber > 1 {
		threadNumberLock.Lock()
		go getUrlPage(fmt.Sprintf("%s&p=%d", urlPathStr, pageNumber))
		threadNumber++
		threadNumberLock.Unlock()
		pageNumber--
	}
}
