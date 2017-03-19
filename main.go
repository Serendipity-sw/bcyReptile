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
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var (
	debugFlag      = flag.Bool("d", false, "debug mode")
	initUrlPath    string //初始化目录地址
	urlHost        string //抓取网站域名
	createFilePath string //文件生成路径
	isFirstStart   = true //是否为第一次启动
	cosPageUrl     []string
	cosPageObj     map[string]int = make(map[string]int)
	zanNumberSort  []int
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

	getUrlPage(initUrlPath)

	var threadZanProcess sync.WaitGroup

	pageZanProcess(&threadZanProcess)
	//threadZanProcess.Wait()

	for _, value := range cosPageObj {
		zanNumberSort = append(zanNumberSort, value)
	}
	sort.Ints(zanNumberSort)
	f, err := os.OpenFile(createFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		glog.Error("fileCreateAndWrite os openFile error! fileName: %s err: %s \n", createFilePath, err.Error())
		return
	}
	defer f.Close()

	for _, value := range zanNumberSort {
		for key, item := range cosPageObj {
			if item == value {
				_, err = f.Write([]byte(fmt.Sprintf("%s %d \n", key, item)))
				if err != nil {
					glog.Error("fileCreateAndWrite write error! content: %v fileName: %s err: %s \n", key, createFilePath, err.Error())
					return
				}
			}
		}
	}
}

/**
每个帖子赞的数量获取
创建人：邵炜
创建时间：2017年03月10日21:16:21
*/
func pageZanProcess(threadZanProcess *sync.WaitGroup) {
	for _, item := range cosPageUrl {
		//threadZanProcess.Add(1)
		coserZanNumberProcess(item, threadZanProcess)
	}

}

/**
coser帖子赞的数量
创建人：邵炜
创建时间：2017年03月10日21:24:32
*/
func coserZanNumberProcess(urlPathStr string, threadZanProcess *sync.WaitGroup) {
	defer func() {
		//threadZanProcess.Done()
	}()
	httpClient, err := http.Get(urlPathStr)
	if err != nil {
		glog.Error("coserZanNumberProcess http get err! urlPathStr: %s err: %s \n", urlPathStr, err.Error())
		return
	}
	defer httpClient.Body.Close()
	docQuery, err := goquery.NewDocumentFromReader(httpClient.Body)
	if err != nil {
		glog.Error("coserZanNumberProcess NewDocumentFromReader run err! urlPath: %s err: %s\n", urlPathStr, err.Error())
		return
	}
	zanNumberStr, bo := docQuery.Find("#js-detailZanTuijian-zan").Attr("data-zan")
	if bo {
		zanNumber, err := strconv.Atoi(zanNumberStr)
		if err != nil {
			glog.Error("coserZanNumberProcess zanNumberStr can't convert string to int! zanNumberStr: %s err: %s \n", zanNumberStr, err.Error())
			return
		}
		cosPageObj[urlPathStr] = zanNumber
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
func pageProcess(bodyByte *[]byte, urlPathStr string) {

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
			arrayList = append(arrayList, fmt.Sprintf("http://%s%s", urlHost, href))
		}
	})
	if len(arrayList) == 0 {
		fmt.Println(urlPathStr)
	}
	cosPageUrl = append(cosPageUrl, arrayList...)
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
	pageProcess(&bodyByte, urlPathStr)
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
		getUrlPage(fmt.Sprintf("%s&p=%d", urlPathStr, pageNumber))
		pageNumber--
	}
}
