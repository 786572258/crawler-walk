package crwaler

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	url      = "https://www.woyaogexing.com/shouji/"
	referImg = "img2.woyaogexing.com"
	isSetUserAgent = false
	referer = ""
)

func downloadUrl(url string) ([]byte, error) {

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if isSetUserAgent == true {
		req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.149 Safari/537.36")
	}
	if referer != "" {
		req.Header.Add("Referer", referer)
	}
	response, err := client.Do(req)
	defer response.Body.Close()

	if err != nil {
		return nil, err
	}

	if response == nil || response.StatusCode != 200 {
		return nil, errors.New("没找到")
	}

	byteContent, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return byteContent, nil
}


//调用os.MkdirAll递归创建文件夹
func CreateMutiDir(filePath string) error {
	if !isExist(filePath) {
		err := os.MkdirAll(filePath, os.ModePerm)
		if err != nil {
			fmt.Println("创建文件夹失败,error info:", err)
			return err
		}
		return err
	}
	return nil
}

// 判断所给路径文件/文件夹是否存在(返回true是存在)
func isExist(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func downloadImg(url string, wg *sync.WaitGroup) {
	defer wg.Done()

	content, err := downloadUrl(url)
	if err != nil {
		fmt.Printf("下载图片%s 失败：%s\n", url, err.Error())
		return
	}
	str1 := strings.Split(url, "/")
	fileName := str1[len(str1)-1]
	date := time.Now().Format("2006-01-02")

	dirPath := "./imgs/"+date+"/"
	err = CreateMutiDir(dirPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = ioutil.WriteFile(dirPath+fileName, content, 0777)
	if err != nil {
		fmt.Printf("下载图片%s 失败：%s\n", url, err.Error())
	}
	fmt.Printf("下载图片%s 成功\n", url)
}

func test(i int, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Println("结果：", i)
}

func start() {
	flagCmd()
}

func run(ruleImgUrl, rulePageUrl, regularImgUrl string, custom bool, ua bool, r string) {
	var wg sync.WaitGroup
	isSetUserAgent = ua
	referer = r
	if ruleImgUrl != "" {
		wg.Add(1)
		crawlByRuleImgUrl(ruleImgUrl, &wg)
		fmt.Println("done.")
		return
	}

	if rulePageUrl != "" {
		if regularImgUrl == "" {
			fmt.Println("缺少正则图片url.")
			return
		}
	}
	if regularImgUrl != "" {
		if rulePageUrl == "" {
			fmt.Println("缺少规则页面url.")
			return
		}
	}

	//页面抓元素获取图片的url
	if regularImgUrl != "" && rulePageUrl != "" {
		//规则页面url
		//rulePageUrl := "https://www.tupianzj.com/meinv/20201102/219671_[1,2].html"
		//正则图片url
		//regularImgUrl := "https://img.lianzhixiu.com/uploads/allimg/.*?.jpg"
		crawlByPage(rulePageUrl, regularImgUrl)
		fmt.Println("done.")
		return
	}

	//===============自定义代码爬取=======================
	if custom == true {
		var totalPage = 2
		wg.Add(totalPage)
		for j:=0; j<totalPage; j++ {
			go crawlByCustom(&wg)
		}
		wg.Wait()

		fmt.Println("")
		fmt.Println("done.")
		//time.Sleep(time.Second * 100)
		return
	}

	fmt.Println("请指定参数执行")

}

func test2(url string, wg *sync.WaitGroup) {
	fmt.Println("执行。")
	wg.Done()
}


//页面抓元素获取图片的url并下载
func crawlByPage(rulePageUrl string, regularImgUrl string) {
	var wg sync.WaitGroup
	first, last, numLength, count, err := parseRuleUrl(rulePageUrl)
	if err != nil {
		fmt.Println(rulePageUrl+"解析错误：", err)
		return
	}
	if count == 0 {
		wg.Add(1)
		go downloadImgBySearchPage(rulePageUrl, regularImgUrl, &wg)
		wg.Wait()
		return
	}
	wg.Add(count)
	for i := first; i<=last; i++ {
		re3, err := regexp.Compile("\\[.*?\\]")
		if err != nil {
			fmt.Println(err)
			continue
		}
		ruleNum := fmt.Sprintf("%0"+strconv.Itoa(numLength)+"d", i)
		pageUrl := re3.ReplaceAllString(rulePageUrl, ruleNum)
		go downloadImgBySearchPage(pageUrl, regularImgUrl, &wg)
		//go test(i, &wg)
	}
	wg.Wait()
}

func downloadImgBySearchPage(pageUrl, regularImgUrl string, wwg *sync.WaitGroup) {
	defer wwg.Done()
	pageContent, err := getPageContent(pageUrl)
	if err != nil {
		fmt.Println(err)
		return
	}
	reg := regexp.MustCompile(regularImgUrl)
	search := reg.FindAllSubmatch([]byte(pageContent), -1)
	if len(search) == 0 {
		fmt.Println(pageUrl + "未找到规则图片")
		return
	}
	var wg sync.WaitGroup
	wg.Add(len(search))
	for _,v:=range search {
		imgUrl := string(v[0])
		go downloadImg(imgUrl, &wg)
	}
	wg.Wait()
}

func getPageContent(pageUrl string) (string, error) {
	// 根据URL获取资源
	res, err := http.Get(pageUrl)

	if err != nil {
		return "", err
	}

	// 读取资源数据 body: []byte
	body, err := ioutil.ReadAll(res.Body)

	// 关闭资源流
	_ = res.Body.Close()
	if res.StatusCode != 200 {
		return "", errors.New("页面访问不到：" + pageUrl)
	}
	if err != nil {
		return "", err
	}
	pageContent := string(body)
	return pageContent, err

}

//规则图片地址，如：https://img-pre.ivsky.com/img/tupian/pre/202107/15/wenquan-[1,3].jpg
func crawlByRuleImgUrl(ruleImgUrl string, wwg *sync.WaitGroup) {
	defer wwg.Done()

	first, last, numLength, count, err := parseRuleUrl(ruleImgUrl)
	if err != nil {
		fmt.Println(ruleImgUrl+"解析错误：", err)
		return
	}
	if count == 0 {
		var wg sync.WaitGroup
		wg.Add(1)
		go downloadImg(ruleImgUrl, &wg)
		wg.Wait()
		return
	}

	var wg sync.WaitGroup
	wg.Add(count)
	for i := first; i<=last; i++ {
		re3, err := regexp.Compile("\\[.*?\\]");
		if err != nil {
			fmt.Println(err)
		}
		ruleNum := fmt.Sprintf("%0"+strconv.Itoa(numLength)+"d", i)
		imgUrl := re3.ReplaceAllString(ruleImgUrl, ruleNum);
		go downloadImg(imgUrl, &wg)
		//go test(i, &wg)
	}
	wg.Wait()
}

//解析类似"https://www.tupianzj.com/meinv/20201102/219671_[1,9].html"地址的规则
//返回如：first=1 last=9 numLength=1 count=9
func parseRuleUrl(ruleUrl string) (first int, last int, numLength int, count int, err error) {
	reg := regexp.MustCompile("\\[(.*?)\\]")
	search := reg.FindAllSubmatch([]byte(ruleUrl), -1)
	first = 0
	last = 0
	rule := ""
	numLength = 0
	var ruleSlice []string
	for _, m := range search {
		rule = string(m[1])
	}

	if rule != "" {
		ruleSlice = strings.Split(rule, ",")
	}
	if len(ruleSlice) == 2 {
		first,_ = strconv.Atoi(ruleSlice[0])
		last,_ = strconv.Atoi(ruleSlice[1])
	} else if len(ruleSlice) == 1 {
		last,_ = strconv.Atoi(ruleSlice[0])
	}

	if len(ruleSlice) == 0 {
		return 0,0,0, 0, nil
	}

	if ruleSlice[0] != ""{
		numLength = len(ruleSlice[0])
	}
	count = (last - first)+1
	return first, last, numLength, count, nil
}

//自定义代码执行
func crawlByCustom(wwg *sync.WaitGroup) {
	defer wwg.Done()

	first := 0
	last := 12
	var wg sync.WaitGroup
	//baseImgUrl = "https://img-pre.ivsky.com/img/tupian/pre/202107/15/wenquan-007.jpg"
	baseImgUrl := "https://img-pre.ivsky.com/img/tupian/pre/202107/15/"
	referImg := ""
	count := (last - first)+1
	fmt.Println("调试：", baseImgUrl, referImg)
	wg.Add(count)
	for i := first; i<=last; i++ {
		imgUrl := baseImgUrl + "wenquan-0"+strconv.Itoa(i)+".jpg"
		go downloadImg(imgUrl, &wg)
		//go test(i, &wg)
	}
	wg.Wait()
}


/**
命令行
*/
func flagCmd() {
	ruleImgUrl := flag.String("ruleImgUrl", "", "规则图片url，如：https://img-pre.ivsky.com/img/tupian/pre/202107/15/wenquan-[1,3].jpg 或者https://img-pre.ivsky.com/img/tupian/pre/202107/15/wenquan-[001,003].jpg")
	regularImgUrl := flag.String("regularImgUrl", "", "正则图片url，如：https://img.lianzhixiu.com/uploads/allimg/.*?.jpg")
	custom := flag.Bool("c", false, "自定义代码执行")
	rulePageUrl := flag.String("rulePageUrl", "", "规则页面url，如：https://www.tupianzj.com/meinv/20201102/219671_[1,2].html，需要配合regularImgUrl使用")
	ua := flag.Bool("ua", false, "是否设置user-agent")
	r := flag.String("r", "", "referer")

	// 解析命令行参数
	flag.Parse()
	run(*ruleImgUrl, *rulePageUrl, *regularImgUrl, *custom, *ua, *r)
}
