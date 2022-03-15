package common

type ENOptions struct {
	KeyWord     string // Keyword of Search
	CompanyID   string // Company ID
	InputFile   string // Scan Input File
	Output      string
	CookieInfo  string
	ScanType    string
    GetAll      bool
	IsGetBranch bool
	IsInvestRd  bool
	InvestNum   int
    Sleep       int
	GetFlags    string
	Version     bool
}
