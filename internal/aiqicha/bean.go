package aiqicha

import (
	"github.com/tidwall/gjson"
)

type EnBen struct {
	Pid           string `json:"pid"`
	EntName       string `json:"entName"`
	EntType       string `json:"entType"`
	ValidityFrom  string `json:"validityFrom"`
	Domicile      string `json:"domicile"`
	EntLogo       string `json:"entLogo"`
	OpenStatus    string `json:"openStatus"`
	LegalPerson   string `json:"legalPerson"`
	LogoWord      string `json:"logoWord"`
	TitleName     string `json:"titleName"`
	TitleLegal    string `json:"titleLegal"`
	TitleDomicile string `json:"titleDomicile"`
	RegCap        string `json:"regCap"`
	Scope         string `json:"scope"`
	RegNo         string `json:"regNo"`
	PersonTitle   string `json:"personTitle"`
	PersonID      string `json:"personId"`
}

type EnsGo struct {
	name      string
	total     int64
	available int64
	api       string
	field     []string
	keyWord   []string
}

type EnInfo struct {
	Pid         string `json:"pid"`
	EntName     string `json:"entName"`
	legalPerson string
	openStatus  string
	email       string
	telephone   string
	branchNum   int64
	investNum   int64
    website     string
    addr        string
    startDate   string
    regCapital  string
    licenseNumber string
    taxNo       string
    industry    string
    entType     string
    openTime    string
    scope       string
	//info
	infos  map[string][]gjson.Result
	ensMap map[string]*EnsGo
	//other
    shareholders []Shareholder
	investInfos map[string]EnInfo
	branchInfos map[string]EnInfo
}

type Shareholder struct {
    name        string
    pid         string
    subRatio    float32
    subMoney    string
    subDate     string
    //positionTitle   string
}
