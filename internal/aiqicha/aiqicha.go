package aiqicha

/* Aiqicha By Keac
 * admin@wgpsec.org
 */
import (
	"os"
	"fmt"
    "time"
	"strconv"
	"strings"
    "io/ioutil"

	"github.com/olekukonko/tablewriter"
    "github.com/antchfx/htmlquery"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/wgpsec/ENScan/common"
	"github.com/wgpsec/ENScan/common/utils"
	"github.com/wgpsec/ENScan/common/utils/gologger"
	"github.com/xuri/excelize/v2"
)

// pageParseJson 提取页面中的JSON字段
func pageParseJson(content string) gjson.Result {

	tag1 := "window.pageData ="
	tag2 := "window.isSpider ="
	//tag2 := "/* eslint-enable */</script><script data-app"
	idx1 := strings.Index(content, tag1)
	idx2 := strings.Index(content, tag2)
	if idx2 > idx1 {
		str := content[idx1+len(tag1) : idx2]
		str = strings.Replace(str, "\n", "", -1)
		str = strings.Replace(str, " ", "", -1)
		str = str[:len(str)-1]
		return gjson.Get(string(str), "result")
	}
	return gjson.Result{}
}

// GetEnInfoByPid 根据PID获取公司信息
func GetEnInfoByPid(options *common.ENOptions) {
	pid := ""
	if options.CompanyID == "" {
		SearchName(options)
	}
	pid = options.CompanyID
	if pid == "" {
		gologger.Fatalf("没有获取到PID\n")
	}
	gologger.Infof("查询PID %s\n", pid)

	//获取公司信息
	res := getCompanyInfoById(pid, true, options)

	//导出 【2021.11.7】 暂时还不能一起投资的企业和分支机构的其他信息
	outPutExcelByEnInfo(res, options)
    invest := res.infos["invest"]
    if options.GetAll && len(invest) > 0 {
		for _, t := range invest {
            name := t.Get("entName").String() + ".xlsx"
            var newOption common.ENOptions
            files, _ := ioutil.ReadDir("excels")
            var exists bool
            for _, file := range files {
                if file.IsDir() {
                    continue
                } else if file.Name() == name {
                    gologger.Infof("%s has been saved, ignore it\n", file.Name())
                    exists = true
                    break
                }
            }
            if exists {
                continue
            }
            gologger.Infof("wait %d seconds", options.Sleep)
            time.Sleep(time.Duration(options.Sleep)*time.Second)
            newOption.CompanyID = t.Get("pid").String()
            newOption.CookieInfo = options.CookieInfo
            newOption.ScanType = options.ScanType
            newOption.GetAll = options.GetAll
            newOption.IsGetBranch = options.IsGetBranch
            newOption.IsInvestRd = options.IsInvestRd
            newOption.InvestNum = options.InvestNum
            newOption.GetFlags = options.GetFlags
            newOption.Sleep = options.Sleep
            GetEnInfoByPid(&newOption)
        }
    }

}

func outPutExcelByEnInfo(enInfo EnInfo, options *common.ENOptions) {
	f := excelize.NewFile()
	//Base info
	baseHeaders := []string{"信息", "值"}
	baseData := [][]interface{}{
		{"PID", enInfo.Pid},
		{"企业名称", enInfo.EntName},
		{"法人代表", enInfo.legalPerson},
		{"开业状态", enInfo.openStatus},
		{"官网", enInfo.website},
		{"地址", enInfo.addr},
		{"成立日期", enInfo.startDate},
		{"注册资本", enInfo.regCapital},
		{"工商注册号", enInfo.licenseNumber},
		{"税号", enInfo.taxNo},
		{"电话", enInfo.telephone},
		{"邮箱", enInfo.email},
		{"经营范围", enInfo.scope},
	}
	f, _ = utils.ExportExcel("基本信息", baseHeaders, baseData, f)

    shHeaders := []string{"股东名称", "持股比例 (%)", "认缴出资额", "认缴出资日期", "pid"}
    var shData [][]interface{}
    for _, s := range enInfo.shareholders {
        var str []interface{}
        str = append(str, s.name)
        if s.subRatio == 0 {
            str = append(str, "不详")
        } else {
            str = append(str, s.subRatio)
        }
        str = append(str, s.subMoney)
        str = append(str, s.subDate)
        str = append(str, s.pid)
        shData = append(shData, str)
    }

	//ensInfoMap["shareholder"].api = "detail/shareholderAjax" // maybe not right
	//ensInfoMap["shareholder"].field = []string{"name", "subRatio", "subMoney", "subDate"}
	//ensInfoMap["shareholder"].keyWord = []string{"股东名称", "持股比例", "认缴出资额", "认缴出资日期"}
    //FIXME: better api -> stockchart/stockchartAjax?pid=13709418951495&drill=0
	f, _ = utils.ExportExcel("股东信息", shHeaders, shData, f)

	for k, s := range enInfo.ensMap {
		if s.total > 0 && s.api != "" {
			gologger.Infof("正在导出%s\n", s.name)
			headers := s.keyWord
			var data [][]interface{}
			for _, y := range enInfo.infos[k] {
				results := gjson.GetMany(y.Raw, s.field...)
				var str []interface{}
				for _, s := range results {
					str = append(str, s.String())
				}
				data = append(data, str)
			}
			f, _ = utils.ExportExcel(s.name, headers, data, f)
		}
	}

	f.DeleteSheet("Sheet1")
	// Save spreadsheet by the given path.
	savaPath := "excels/" + enInfo.EntName + ".xlsx"
	if err := f.SaveAs(savaPath); err != nil {
		gologger.Fatalf("导出失败：%s", err)
	}
	gologger.Infof("导出成功路径： %s\n", savaPath)

}

func getMoreInfos(pid string, options *common.ENOptions) gjson.Result {
	urls := "https://aiqicha.baidu.com/detail/basicAllDataAjax?pid=" + pid
	content := common.GetReq(urls, options)
    return gjson.Get(string(content), "data")
}

// getCompanyInfoById 获取公司基本信息
// pid 公司id
// isSearch 是否递归搜索信息【分支机构、对外投资信息】
// options options
func getCompanyInfoById(pid string, isSearch bool, options *common.ENOptions) EnInfo {
	var enInfo EnInfo
	enInfo.infos = make(map[string][]gjson.Result)
	urls := "https://aiqicha.baidu.com/company_detail_" + pid
	content := common.GetReq(urls, options)
	res := pageParseJson(string(content))
	//获取企业基本信息情况
	enInfo.Pid = res.Get("pid").String()
	enInfo.EntName = res.Get("entName").String()
	enInfo.legalPerson = res.Get("legalPerson").String()
	enInfo.openStatus = res.Get("openStatus").String()
	enInfo.telephone = res.Get("telephone").String()
	enInfo.email = res.Get("email").String()
	enInfo.website = res.Get("website").String()
	enInfo.addr = res.Get("addr").String()
	enInfo.startDate = res.Get("startDate").String()
	enInfo.regCapital = res.Get("regCapital").String()
	enInfo.licenseNumber = res.Get("licenseNumber").String()
	enInfo.taxNo = res.Get("taxNo").String()
	enInfo.scope = res.Get("scope").String()
	gologger.Infof("企业基本信息\n")
	data := [][]string{
		{"PID", enInfo.Pid},
		{"企业名称", enInfo.EntName},
		{"法人代表", enInfo.legalPerson},
		{"开业状态", enInfo.openStatus},
		{"官网", enInfo.website},
		{"地址", enInfo.addr},
		{"成立日期", enInfo.startDate},
		{"注册资本", enInfo.regCapital},
		{"工商注册号", enInfo.licenseNumber},
		{"税号", enInfo.taxNo},
		{"电话", enInfo.telephone},
		{"邮箱", enInfo.email},
		{"经营范围", enInfo.scope},
	}
    minfos := getMoreInfos(pid, options)
    enInfo.industry = minfos.Get("basicData.industry").String()
    enInfo.entType = minfos.Get("basicData.entType").String()
    enInfo.openTime = minfos.Get("basicData.openTime").String()
    data = append(data, []string{"所属行业", minfos.Get("basicData.industry").String()})
    data = append(data, []string{"企业类型", minfos.Get("basicData.entType").String()})
    data = append(data, []string{"营业期限", minfos.Get("basicData.openTime").String()})

    var shareholders []Shareholder
    for i := int64(0); i < minfos.Get("shareholdersData.list.#").Int(); i++ {
        var shareholder Shareholder
        shareholder.name = minfos.Get(fmt.Sprintf("shareholdersData.list.%d.name", i)).String()
        //shareholder.positionTitle = minfos.Get(fmt.Sprintf("shareholdersData.list.%d.positionTitle", i)).String()
        retio := strings.Split(minfos.Get(fmt.Sprintf("shareholdersData.list.%d.subRate", i)).String(), "%")[0]
        shareholder.subMoney = minfos.Get(fmt.Sprintf("shareholdersData.list.%d.subMoney", i)).String()
        shareholder.subDate = minfos.Get(fmt.Sprintf("shareholdersData.list.%d.subDate", i)).String()
        shareholder.pid = minfos.Get(fmt.Sprintf("shareholdersData.list.%d.pid", i)).String()
        percent, _ := strconv.ParseFloat(retio, 32)
        shareholder.subRatio = float32(percent)
        shareholders = append(shareholders, shareholder)
    }
	enInfo.shareholders = shareholders
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAlignment(tablewriter.ALIGN_CENTER)
	table.AppendBulk(data)
	table.Render()

	//判断企业状态，不然就可以跳过了
	if enInfo.openStatus == "注销" || enInfo.openStatus == "吊销" {
		return enInfo
	}

	//获取企业信息
	enInfoUrl := "https://aiqicha.baidu.com/compdata/navigationListAjax?pid=" + pid
	enInfoRes := common.GetReq(enInfoUrl, options)
	ensInfoMap := make(map[string]*EnsGo)
	if gjson.Get(string(enInfoRes), "status").String() == "0" {
		data := gjson.Get(string(enInfoRes), "data").Array()
		for _, s := range data {
			for _, t := range s.Get("children").Array() {
				ensInfoMap[t.Get("id").String()] = &EnsGo{
					t.Get("name").String(),
					t.Get("total").Int(),
					t.Get("avaliable").Int(),
					"",
					[]string{},
					[]string{},
				}
			}
		}
	}

	//赋值API数据
	ensInfoMap["webRecord"].api = "detail/icpinfoAjax"
	ensInfoMap["webRecord"].field = []string{"domain", "siteName", "homeSite", "icpNo"}
	ensInfoMap["webRecord"].keyWord = []string{"域名", "站点名称", "首页", "ICP备案号"}

	ensInfoMap["appinfo"].api = "c/appinfoAjax"
	ensInfoMap["appinfo"].field = []string{"name", "classify", "logoWord", "logoBrief", "entName"}
	ensInfoMap["appinfo"].keyWord = []string{"APP名称", "分类", "LOGO文字", "描述", "所属公司"}

	ensInfoMap["microblog"].api = "c/microblogAjax"
	ensInfoMap["microblog"].field = []string{"nickname", "weiboLink", "logo"}
	ensInfoMap["microblog"].keyWord = []string{"微博昵称", "链接", "LOGO"}

	ensInfoMap["wechatoa"].api = "c/wechatoaAjax"
	ensInfoMap["wechatoa"].field = []string{"wechatName", "wechatId", "wechatIntruduction", "wechatLogo", "qrcode", "entName"}
	ensInfoMap["wechatoa"].keyWord = []string{"名称", "ID", "描述", "LOGO", "二维码", "归属公司"}

	ensInfoMap["enterprisejob"].api = "c/enterprisejobAjax"
	ensInfoMap["enterprisejob"].field = []string{"jobTitle", "location", "salary", "education", "publishDate", "desc"}
	ensInfoMap["enterprisejob"].keyWord = []string{"职位名称", "工作地点", "薪资", "学历要求", "发布日期", "招聘描述"}

	ensInfoMap["copyright"].api = "detail/copyrightAjax"
	ensInfoMap["copyright"].field = []string{"softwareName", "shortName", "softwareType", "typeCode", "regDate"}
	ensInfoMap["copyright"].keyWord = []string{"软件名称", "软件简介", "分类", "行业", "日期"}

	ensInfoMap["supplier"].api = "c/supplierAjax"
	ensInfoMap["supplier"].field = []string{"supplier", "source", "principalNameClient", "cooperationDate"}
	ensInfoMap["supplier"].keyWord = []string{"供应商名称", "来源", "所属公司", "日期"}

	ensInfoMap["invest"].api = "detail/investajax" //对外投资
	ensInfoMap["invest"].field = []string{"entName", "openStatus", "regRate", "data", "pid"}
	ensInfoMap["invest"].keyWord = []string{"公司名称", "状态", "投资比例", "数据信息", "pid"}

	ensInfoMap["branch"].api = "detail/branchajax" //分支机构
	ensInfoMap["branch"].field = []string{"entName", "openStatus", "data"}
	ensInfoMap["branch"].keyWord = []string{"公司名称", "状态", "数据信息"}

	enInfo.ensMap = ensInfoMap

	//获取数据
	for k, s := range ensInfoMap {
		if s.total > 0 && s.api != "" {
			gologger.Infof("正在查询 %s\n", s.name)
			t := getInfoList(res.Get("pid").String(), s.api, options)

			//判断下网站备案，然后提取出来，留个坑看看有没有更好的解决方案
			if k == "webRecord" {
				var tmp []gjson.Result
				for _, y := range t {
					for _, o := range y.Get("domain").Array() {
						value, _ := sjson.Set(y.Raw, "domain", o.String())
						tmp = append(tmp, gjson.Parse(value))
					}
				}
				t = tmp
			}
			//保存数据
			enInfo.infos[k] = t

			//命令输出展示
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader(ensInfoMap[k].keyWord)
			for _, y := range t {
				results := gjson.GetMany(y.Raw, ensInfoMap[k].field...)
				var str []string
				for _, s := range results {
					str = append(str, s.String())
				}
				table.Append(str)
			}
			table.Render()

		}
	}

	// 查询对外投资详细信息
	// 对外投资>0 && 是否递归 && 参数投资信息大于0
	if ensInfoMap["invest"].total > 0 && isSearch && options.InvestNum > 0 {
		enInfo.investInfos = make(map[string]EnInfo)
		for _, t := range enInfo.infos["invest"] {
			gologger.Infof("企业名称：%s 投资占比：%s\n", t.Get("entName"), t.Get("regRate"))
			//openStatus := t.Get("openStatus").String()
			//if openStatus == "注销" || openStatus == "吊销" {
			//	continue
			//}
			investNum := 0.00
			if t.Get("regRate").String() == "-" {
				investNum = -1
			} else {
				str := strings.Replace(t.Get("regRate").String(), "%", "", -1)
				investNum, _ = strconv.ParseFloat(str, 2)
			}
			if investNum >= 100 {
				n := getCompanyInfoById(t.Get("pid").String(), false, options)
				enInfo.investInfos[t.Get("pid").String()] = n
			}
		}
	}

	// 查询分支机构公司详细信息
	// 分支机构大于0 && 是否递归模式 && 参数是否开启查询
	if ensInfoMap["branch"].total > 0 && isSearch && options.IsGetBranch {
		enInfo.branchInfos = make(map[string]EnInfo)
		for _, t := range enInfo.infos["branch"] {
			gologger.Infof("分支名称：%s 状态：%s\n", t.Get("entName"), t.Get("openStatus"))
			n := getCompanyInfoById(t.Get("pid").String(), false, options)
			enInfo.branchInfos[t.Get("pid").String()] = n
		}
	}

	return enInfo

}

// getInfoList 获取信息列表
func getInfoList(pid string, types string, options *common.ENOptions) []gjson.Result {
	urls := "https://aiqicha.baidu.com/" + types + "?size=100&pid=" + pid
	content := common.GetReq(urls, options)
	var listData []gjson.Result
	if gjson.Get(string(content), "status").String() == "0" {
		data := gjson.Get(string(content), "data")
		//判断一个获取的特殊值
		if types == "relations/relationalMapAjax" {
			data = gjson.Get(string(content), "data.investRecordData")
		}
		//判断是否多页，遍历获取所有数据
		pageCount := data.Get("pageCount").Int()
		if pageCount > 1 {
			for i := 1; int(pageCount) >= i; i++ {
				gologger.Infof("当前：%s,%d\n", types, i)
				reqUrls := urls + "&p=" + strconv.Itoa(i)
				content = common.GetReq(reqUrls, options)
				listData = append(listData, gjson.Get(string(content), "data.list").Array()...)
			}
		} else {
			listData = data.Get("list").Array()
		}
	}
	return listData

}

// SearchName 根据企业名称搜索信息
func SearchName(options *common.ENOptions) []gjson.Result {
	name := options.KeyWord
	urls := "https://aiqicha.baidu.com/s?q=" + name + "&p=1&s=10&t=0"
	content := common.GetReq(urls, options)

	page, _ := htmlquery.Parse(strings.NewReader(string(content)))
    list := htmlquery.Find(page, "//script")
    //fmt.Println(htmlquery.InnerText(list[4])) // output @href value
    data := strings.SplitAfterN(htmlquery.InnerText(list[4]), "\n", 2)
    js := data[0][0:len(data[0])-2]

	enList := gjson.Get(js, "result.resultList").Array()

	if len(enList) == 0 {
		gologger.Errorf("没有查询到关键词 “%s” ", name)
		return enList
	} else {
		gologger.Infof("关键词：“%s” 查询到 %d 个结果，默认选择第一个 \n", name, len(enList))
	}
	options.CompanyID = enList[0].Get("pid").String()
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"PID", "企业名称", "法人代表"})
	for _, v := range enList {
		table.Append([]string{v.Get("pid").String(), v.Get("titleName").String(), v.Get("titleLegal").String()})
	}
	table.Render()
	return enList
}
