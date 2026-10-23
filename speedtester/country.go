package speedtester

import (
	"regexp"
	"strings"
)

// GetCountryFlag 根据 2 位国家代码生成对应的国旗 Emoji
func GetCountryFlag(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "UK" {
		code = "GB"
	}
	if len(code) == 2 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= 'Z' {
		r1 := rune(0x1F1E6 + int(code[0]-'A'))
		r2 := rune(0x1F1E6 + int(code[1]-'A'))
		return string([]rune{r1, r2})
	}
	return "🏳️"
}

// GetCountryName 根据国家代码获取国家/地区的中文名称
func GetCountryName(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	if name, ok := CountryNames[code]; ok {
		return name
	}
	return "未知"
}

// ExtractFlagEmoji 从节点名称中提取 Unicode 国旗 Emoji 并转换为 ISO 2 位国家代码
func ExtractFlagEmoji(name string) string {
	runes := []rune(name)
	for i := 0; i < len(runes)-1; i++ {
		r1, r2 := runes[i], runes[i+1]
		if r1 >= 0x1F1E6 && r1 <= 0x1F1FF && r2 >= 0x1F1E6 && r2 <= 0x1F1FF {
			c1 := byte('A' + int(r1-0x1F1E6))
			c2 := byte('A' + int(r2-0x1F1E6))
			code := string([]byte{c1, c2})
			if code == "UK" {
				code = "GB"
			}
			return code
		}
	}
	return ""
}

// 常见城市与地区别名映射到 ISO 2 位代码
var regionKeywordMap = map[string]string{
	"香港": "HK", "深港": "HK", "沪港": "HK", "京港": "HK", "广港": "HK", "穗港": "HK", "HKT": "HK", "HKBN": "HK",
	"台湾": "TW", "台北": "TW", "台中": "TW", "新北": "TW", "高雄": "TW", "彰化": "TW", "沪台": "TW", "广台": "TW", "深台": "TW",
	"日本": "JP", "东京": "JP", "大阪": "JP", "埼玉": "JP", "名古屋": "JP", "福冈": "JP", "横滨": "JP", "沪日": "JP", "京日": "JP", "广日": "JP", "深日": "JP",
	"美国": "US", "洛杉矶": "US", "圣何塞": "US", "硅谷": "US", "西雅图": "US", "纽约": "US", "沪美": "US", "京美": "US", "广美": "US", "深美": "US",
	"芝加哥": "US", "达拉斯": "US", "凤凰城": "US", "波特兰": "US", "迈阿密": "US", "旧金山": "US",
	"新加坡": "SG", "狮城": "SG", "沪新": "SG", "京新": "SG", "广新": "SG", "深新": "SG",
	"韩国": "KR", "首尔": "KR", "仁川": "KR", "春川": "KR", "沪韩": "KR", "京韩": "KR",
	"英国": "GB", "伦敦": "GB", "曼彻斯特": "GB", "沪英": "GB",
	"德国": "DE", "法兰克福": "DE", "柏林": "DE", "慕尼黑": "DE", "杜塞尔多夫": "DE", "沪德": "DE",
	"法国": "FR", "巴黎": "FR", "马赛": "FR",
	"俄罗斯": "RU", "莫斯科": "RU", "圣彼得堡": "RU", "新西伯利亚": "RU", "符拉迪沃斯托克": "RU", "海参崴": "RU",
	"加拿大": "CA", "多伦多": "CA", "温哥华": "CA", "蒙特利尔": "CA",
	"澳大利亚": "AU", "悉尼": "AU", "墨尔本": "AU", "布里斯班": "AU", "珀斯": "AU", "澳洲": "AU",
	"荷兰": "NL", "阿姆斯特丹": "NL", "鹿特丹": "NL",
	"意大利": "IT", "米兰": "IT", "罗马": "IT",
	"西班牙": "ES", "马德里": "ES", "巴塞罗那": "ES",
	"瑞士": "CH", "苏黎世": "CH", "日内瓦": "CH",
	"瑞典": "SE", "斯德哥尔摩": "SE",
	"挪威": "NO", "奥斯陆": "NO",
	"丹麦": "DK", "哥本哈根": "DK",
	"芬兰": "FI", "赫尔辛基": "FI",
	"奥地利": "AT", "维也纳": "AT",
	"比利时": "BE", "布鲁塞尔": "BE",
	"波兰": "PL", "华沙": "PL",
	"土耳其": "TR", "伊斯坦布尔": "TR", "安卡拉": "TR",
	"阿联酋": "AE", "迪拜": "AE", "阿布扎比": "AE",
	"印度": "IN", "孟买": "IN", "德里": "IN", "班加罗尔": "IN",
	"泰国": "TH", "曼谷": "TH",
	"马来西亚": "MY", "吉隆坡": "MY", "大马": "MY",
	"越南": "VN", "河内": "VN", "胡志明": "VN",
	"菲律宾": "PH", "马尼拉": "PH",
	"印度尼西亚": "ID", "雅加达": "ID", "印尼": "ID",
	"巴西": "BR", "圣保罗": "BR", "里约热内卢": "BR",
	"阿根廷": "AR", "布宜诺斯艾利斯": "AR",
	"智利": "CL", "圣地亚哥": "CL",
	"墨西哥": "MX", "墨西哥城": "MX",
	"南非": "ZA", "约翰内斯堡": "ZA", "开普敦": "ZA",
	"尼日利亚": "NG", "拉各斯": "NG",
	"埃及": "EG", "开罗": "EG",
	"以色列": "IL", "特拉维夫": "IL", "耶路撒冷": "IL",
	"沙特阿拉伯": "SA", "沙特": "SA", "利雅得": "SA",
	"乌克兰": "UA", "基辅": "UA",
	"哈萨克斯坦": "KZ", "阿拉木图": "KZ",
	"乌兹别克斯坦": "UZ", "塔什干": "UZ",
	"吉尔吉斯斯坦": "KG", "比什凯克": "KG",
	"格鲁吉亚": "GE", "第比利斯": "GE",
	"亚美尼亚": "AM", "埃里温": "AM",
	"阿塞拜疆": "AZ", "巴库": "AZ",
	"巴基斯坦": "PK", "伊斯兰堡": "PK", "卡拉奇": "PK",
	"孟加拉国": "BD", "孟加拉": "BD", "达卡": "BD",
	"柬埔寨": "KH", "金边": "KH",
	"老挝": "LA", "万象": "LA",
	"缅甸": "MM", "仰光": "MM",
	"蒙古": "MN", "乌兰巴托": "MN",
	"尼泊尔": "NP", "加德满都": "NP",
	"斯里兰卡": "LK", "科伦坡": "LK",
	"朝鲜": "KP", "平壤": "KP",
	"希腊": "GR", "雅典": "GR",
	"葡萄牙": "PT", "里斯本": "PT",
	"爱尔兰": "IE", "都柏林": "IE",
	"捷克": "CZ", "布拉格": "CZ",
	"匈牙利": "HU", "布达佩斯": "HU",
	"罗马尼亚": "RO", "布加勒斯特": "RO",
	"保加利亚": "BG", "索非亚": "BG",
	"克罗地亚": "HR", "萨格勒布": "HR",
	"塞尔维亚": "RS", "贝尔格莱德": "RS",
	"斯洛伐克": "SK", "布拉迪斯拉发": "SK",
	"斯洛文尼亚": "SI", "卢布尔雅那": "SI",
	"立陶宛": "LT", "维尔纽斯": "LT",
	"拉脱维亚": "LV", "里加": "LV",
	"爱沙尼亚": "EE", "塔林": "EE",
	"白俄罗斯": "BY", "明斯克": "BY",
	"摩尔多瓦": "MD", "基希讷乌": "MD",
	"冰岛": "IS", "雷克雅未克": "IS",
	"卢森堡":  "LU",
	"马耳他":  "MT",
	"塞浦路斯": "CY",
	"摩纳哥":  "MC",
	"新西兰":  "NZ", "奥克兰": "NZ", "惠灵顿": "NZ",
	"哥伦比亚": "CO", "波哥大": "CO",
	"秘鲁": "PE", "利马": "PE",
	"委内瑞拉": "VE", "加拉加斯": "VE",
	"厄瓜多尔": "EC", "基多": "EC",
	"乌拉圭": "UY", "蒙得维的亚": "UY",
	"巴拉圭": "PY", "亚松森": "PY",
	"玻利维亚": "BO", "拉巴斯": "BO",
	"哥斯达黎加": "CR",
	"巴拿马":   "PA",
	"危地马拉":  "GT",
	"古巴":    "CU", "哈瓦那": "CU",
	"多米尼加": "DO",
	"波多黎各": "PR",
	"中国":   "CN", "国内": "CN", "上海": "CN", "北京": "CN", "广州": "CN", "深圳": "CN", "杭州": "CN", "江苏": "CN", "广东": "CN", "浙江": "CN",
}

// 常见英文全称/别名映射
var englishKeywordMap = map[string]string{
	"HONGKONG": "HK", "HONG KONG": "HK",
	"TAIWAN": "TW",
	"JAPAN":  "JP", "TOKYO": "JP", "OSAKA": "JP",
	"UNITED STATES": "US", "AMERICA": "US", "LOS ANGELES": "US", "SAN JOSE": "US", "SILICON VALLEY": "US", "NEW YORK": "US", "SEATTLE": "US", "CHICAGO": "US", "DALLAS": "US",
	"SINGAPORE": "SG",
	"KOREA":     "KR", "SOUTH KOREA": "KR", "SEOUL": "KR",
	"UNITED KINGDOM": "GB", "BRITAIN": "GB", "LONDON": "GB",
	"GERMANY": "DE", "FRANKFURT": "DE", "BERLIN": "DE",
	"FRANCE": "FR", "PARIS": "FR",
	"RUSSIA": "RU", "MOSCOW": "RU",
	"CANADA": "CA", "TORONTO": "CA", "VANCOUVER": "CA",
	"AUSTRALIA": "AU", "SYDNEY": "AU", "MELBOURNE": "AU",
	"NETHERLANDS": "NL", "HOLLAND": "NL", "AMSTERDAM": "NL",
	"ITALY": "IT", "MILAN": "IT", "ROME": "IT",
	"SPAIN": "ES", "MADRID": "ES", "BARCELONA": "ES",
	"SWITZERLAND": "CH", "ZURICH": "CH",
	"SWEDEN": "SE", "STOCKHOLM": "SE",
	"NORWAY": "NO", "OSLO": "NO",
	"DENMARK": "DK", "COPENHAGEN": "DK",
	"FINLAND": "FI", "HELSINKI": "FI",
	"AUSTRIA": "AT", "VIENNA": "AT",
	"BELGIUM": "BE", "BRUSSELS": "BE",
	"POLAND": "PL", "WARSAW": "PL",
	"TURKEY": "TR", "ISTANBUL": "TR",
	"INDIA": "IN", "MUMBAI": "IN",
	"THAILAND": "TH", "BANGKOK": "TH",
	"MALAYSIA": "MY", "KUALA LUMPUR": "MY",
	"VIETNAM": "VN", "HANOI": "VN",
	"PHILIPPINES": "PH", "MANILA": "PH",
	"INDONESIA": "ID", "JAKARTA": "ID",
	"BRAZIL": "BR", "SAO PAULO": "BR",
	"ARGENTINA":    "AR",
	"CHILE":        "CL",
	"MEXICO":       "MX",
	"SOUTH AFRICA": "ZA",
	"EGYPT":        "EG", "CAIRO": "EG",
	"ISRAEL":       "IL",
	"SAUDI ARABIA": "SA",
	"UKRAINE":      "UA", "KYIV": "UA",
	"NEW ZEALAND": "NZ", "AUCKLAND": "NZ",
	"IRELAND": "IE", "DUBLIN": "IE",
	"PORTUGAL": "PT", "LISBON": "PT",
	"GREECE": "GR", "ATHENS": "GR",
	"CZECH": "CZ", "PRAGUE": "CZ",
	"HUNGARY": "HU", "BUDAPEST": "HU",
	"ROMANIA": "RO", "BUCHAREST": "RO",
	"BULGARIA": "BG", "SOFIA": "BG",
	"CROATIA":    "HR",
	"SERBIA":     "RS",
	"SLOVAKIA":   "SK",
	"SLOVENIA":   "SI",
	"LITHUANIA":  "LT",
	"LATVIA":     "LV",
	"ESTONIA":    "EE",
	"ICELAND":    "IS",
	"LUXEMBOURG": "LU",
	"CHINA":      "CN",
}

var englishCodePattern = regexp.MustCompile(`(?i)\b(HK|TW|JP|US|USA|SG|KR|UK|GB|DE|FR|RU|CA|AU|NL|IT|ES|CH|SE|NO|DK|FI|AT|BE|PL|TR|AE|IN|TH|MY|VN|PH|ID|BR|AR|CL|MX|ZA|NG|EG|IL|SA|UA|NZ|IE|PT|GR|CZ|HU|RO|BG|HR|RS|SK|SI|LT|LV|EE|IS|LU|MT|CY|KZ|PK|BD|KH|MM|MN|LK|CO|PE|VE|EC|UY|CR|PA|DO|CU|BS|CN)\b`)

// ExtractCountryCodeFromName 尝试从节点原名称中提取国家/地区代码
func ExtractCountryCodeFromName(nodeName string) string {
	if nodeName == "" {
		return ""
	}

	// 1. 优先提取 Unicode 国旗 Emoji
	if flagCode := ExtractFlagEmoji(nodeName); flagCode != "" {
		return flagCode
	}

	// 2. 匹配中文城市与地区别名
	for kw, code := range regionKeywordMap {
		if strings.Contains(nodeName, kw) {
			return code
		}
	}

	// 3. 匹配中文国家全称
	for code, name := range CountryNames {
		if code == "UNKNOWN" {
			continue
		}
		if len(name) >= 2 && strings.Contains(nodeName, name) {
			return code
		}
	}

	upperName := strings.ToUpper(nodeName)

	// 4. 匹配英文全称/英文城市
	for kw, code := range englishKeywordMap {
		if strings.Contains(upperName, kw) {
			return code
		}
	}

	// 5. 匹配独立英文单词代码 (如 HK, US, JP, SG 等)
	if match := englishCodePattern.FindString(upperName); match != "" {
		code := strings.ToUpper(match)
		if code == "USA" {
			return "US"
		}
		if code == "UK" {
			return "GB"
		}
		if _, ok := CountryNames[code]; ok {
			return code
		}
	}

	return ""
}

// ExtractCountryFromNodeName 从节点名称中提取完整国家信息 (代码、中文名称、国旗 Emoji)
func ExtractCountryFromNodeName(nodeName string) (countryCode, countryName, countryFlag string) {
	code := ExtractCountryCodeFromName(nodeName)
	if code == "" {
		return "UNKNOWN", "未知", "🏳️"
	}
	return code, GetCountryName(code), GetCountryFlag(code)
}

// ResolveCountry 综合 API 定位与节点原始名称解析国家信息
func ResolveCountry(location *IPLocation, err error, originalNodeName string) (countryCode, countryName, countryFlag string) {
	if err == nil && location != nil {
		code := strings.ToUpper(strings.TrimSpace(location.CountryCode))
		if code != "" && code != "UNKNOWN" {
			return code, GetCountryName(code), GetCountryFlag(code)
		}
	}

	// API 无法获取或返回未知时，回退从节点原名提取
	return ExtractCountryFromNodeName(originalNodeName)
}

// CountryNames 全球全部 ISO 3166-1 2位国家/地区代码与中文名称完整映射
var CountryNames = map[string]string{
	"AD":      "安道尔",
	"AE":      "阿联酋",
	"AF":      "阿富汗",
	"AG":      "安提瓜和巴布达",
	"AI":      "安圭拉",
	"AL":      "阿尔巴尼亚",
	"AM":      "亚美尼亚",
	"AO":      "安哥拉",
	"AQ":      "南极洲",
	"AR":      "阿根廷",
	"AS":      "美属萨摩亚",
	"AT":      "奥地利",
	"AU":      "澳大利亚",
	"AW":      "阿鲁巴",
	"AX":      "奥兰群岛",
	"AZ":      "阿塞拜疆",
	"BA":      "波斯尼亚和黑塞哥维那",
	"BB":      "巴巴多斯",
	"BD":      "孟加拉国",
	"BE":      "比利时",
	"BF":      "布基纳法索",
	"BG":      "保加利亚",
	"BH":      "巴林",
	"BI":      "布隆迪",
	"BJ":      "贝宁",
	"BL":      "圣巴泰勒米",
	"BM":      "百慕大",
	"BN":      "文莱",
	"BO":      "玻利维亚",
	"BQ":      "荷属加勒比区",
	"BR":      "巴西",
	"BS":      "巴哈马",
	"BT":      "不丹",
	"BV":      "布韦岛",
	"BW":      "博茨瓦纳",
	"BY":      "白俄罗斯",
	"BZ":      "伯利兹",
	"CA":      "加拿大",
	"CC":      "科科斯（基林）群岛",
	"CD":      "刚果（金）",
	"CF":      "中非共和国",
	"CG":      "刚果（布）",
	"CH":      "瑞士",
	"CI":      "科特迪瓦",
	"CK":      "库克群岛",
	"CL":      "智利",
	"CM":      "喀麦隆",
	"CN":      "中国",
	"CO":      "哥伦比亚",
	"CR":      "哥斯达黎加",
	"CU":      "古巴",
	"CV":      "佛得角",
	"CW":      "库拉索",
	"CX":      "圣诞岛",
	"CY":      "塞浦路斯",
	"CZ":      "捷克",
	"DE":      "德国",
	"DJ":      "吉布提",
	"DK":      "丹麦",
	"DM":      "多米尼克",
	"DO":      "多米尼加",
	"DZ":      "阿尔及利亚",
	"EC":      "厄瓜多尔",
	"EE":      "爱沙尼亚",
	"EG":      "埃及",
	"EH":      "西撒哈拉",
	"ER":      "厄立特里亚",
	"ES":      "西班牙",
	"ET":      "埃塞俄比亚",
	"FI":      "芬兰",
	"FJ":      "斐济",
	"FK":      "福克兰群岛",
	"FM":      "密克罗尼西亚",
	"FO":      "法罗群岛",
	"FR":      "法国",
	"GA":      "加蓬",
	"GB":      "英国",
	"UK":      "英国",
	"GD":      "格林纳达",
	"GE":      "格鲁吉亚",
	"GF":      "法属圭亚那",
	"GG":      "根西岛",
	"GH":      "加纳",
	"GI":      "直布罗陀",
	"GL":      "格陵兰",
	"GM":      "冈比亚",
	"GN":      "几内亚",
	"GP":      "瓜德罗普",
	"GQ":      "赤道几内亚",
	"GR":      "希腊",
	"GS":      "南乔治亚和南桑威奇群岛",
	"GT":      "危地马拉",
	"GU":      "关岛",
	"GW":      "几内亚比绍",
	"GY":      "圭亚那",
	"HK":      "香港",
	"HM":      "赫德岛和麦克唐纳群岛",
	"HN":      "洪都拉斯",
	"HR":      "克罗地亚",
	"HT":      "海地",
	"HU":      "匈牙利",
	"ID":      "印度尼西亚",
	"IE":      "爱尔兰",
	"IL":      "以色列",
	"IM":      "马恩岛",
	"IN":      "印度",
	"IO":      "英属印度洋领地",
	"IQ":      "伊拉克",
	"IR":      "伊朗",
	"IS":      "冰岛",
	"IT":      "意大利",
	"JE":      "泽西岛",
	"JM":      "牙买加",
	"JO":      "约旦",
	"JP":      "日本",
	"KE":      "肯尼亚",
	"KG":      "吉尔吉斯斯坦",
	"KH":      "柬埔寨",
	"KI":      "基里巴斯",
	"KM":      "科摩罗",
	"KN":      "圣基茨和尼维斯",
	"KP":      "朝鲜",
	"KR":      "韩国",
	"KW":      "科威特",
	"KY":      "开曼群岛",
	"KZ":      "哈萨克斯坦",
	"LA":      "老挝",
	"LB":      "黎巴嫩",
	"LC":      "圣卢西亚",
	"LI":      "列支敦士登",
	"LK":      "斯里兰卡",
	"LR":      "利比里亚",
	"LS":      "莱索托",
	"LT":      "立陶宛",
	"LU":      "卢森堡",
	"LV":      "拉脱维亚",
	"LY":      "利比亚",
	"MA":      "摩洛哥",
	"MC":      "摩纳哥",
	"MD":      "摩尔多瓦",
	"ME":      "黑山",
	"MF":      "法属圣马丁",
	"MG":      "马达加斯加",
	"MH":      "马绍尔群岛",
	"MK":      "北马其顿",
	"ML":      "马里",
	"MM":      "缅甸",
	"MN":      "蒙古",
	"MO":      "澳门",
	"MP":      "北马里亚纳群岛",
	"MQ":      "马提尼克",
	"MR":      "毛里塔尼亚",
	"MS":      "蒙特塞拉特",
	"MT":      "马耳他",
	"MU":      "毛里求斯",
	"MV":      "马尔代夫",
	"MW":      "马拉维",
	"MX":      "墨西哥",
	"MY":      "马来西亚",
	"MZ":      "莫桑比克",
	"NA":      "纳米比亚",
	"NC":      "新喀里多尼亚",
	"NE":      "尼日尔",
	"NF":      "诺福克岛",
	"NG":      "尼日利亚",
	"NI":      "尼加拉瓜",
	"NL":      "荷兰",
	"NO":      "挪威",
	"NP":      "尼泊尔",
	"NR":      "瑙鲁",
	"NU":      "纽埃",
	"NZ":      "新西兰",
	"OM":      "阿曼",
	"PA":      "巴拿马",
	"PE":      "秘鲁",
	"PF":      "法属波利尼西亚",
	"PG":      "巴布亚新几内亚",
	"PH":      "菲律宾",
	"PK":      "巴基斯坦",
	"PL":      "波兰",
	"PM":      "圣皮埃尔和密克隆",
	"PN":      "皮特凯恩群岛",
	"PR":      "波多黎各",
	"PS":      "巴勒斯坦",
	"PT":      "葡萄牙",
	"PW":      "帕劳",
	"PY":      "巴拉圭",
	"QA":      "卡塔尔",
	"RE":      "留尼汪",
	"RO":      "罗马尼亚",
	"RS":      "塞尔维亚",
	"RU":      "俄罗斯",
	"RW":      "卢旺达",
	"SA":      "沙特阿拉伯",
	"SB":      "所罗门群岛",
	"SC":      "塞舌尔",
	"SD":      "苏丹",
	"SE":      "瑞典",
	"SG":      "新加坡",
	"SH":      "圣赫勒拿",
	"SI":      "斯洛文尼亚",
	"SJ":      "斯瓦尔巴和扬马延",
	"SK":      "斯洛伐克",
	"SL":      "塞拉利昂",
	"SM":      "圣马力诺",
	"SN":      "塞内加尔",
	"SO":      "索马里",
	"SR":      "苏里南",
	"SS":      "南苏丹",
	"ST":      "圣多美和普林西比",
	"SV":      "萨尔瓦多",
	"SX":      "荷属圣马丁",
	"SY":      "叙利亚",
	"SZ":      "斯威士兰",
	"TC":      "特克斯和凯科斯群岛",
	"TD":      "乍得",
	"TF":      "法属南部领地",
	"TG":      "多哥",
	"TH":      "泰国",
	"TJ":      "塔吉克斯坦",
	"TK":      "托克劳",
	"TL":      "东帝汶",
	"TM":      "土库曼斯坦",
	"TN":      "突尼斯",
	"TO":      "汤加",
	"TR":      "土耳其",
	"TT":      "特立尼达和多巴哥",
	"TV":      "图瓦卢",
	"TW":      "台湾",
	"TZ":      "坦桑尼亚",
	"UA":      "乌克兰",
	"UG":      "乌干达",
	"UM":      "美国本土外小岛屿",
	"US":      "美国",
	"UY":      "乌拉圭",
	"UZ":      "乌兹别克斯坦",
	"VA":      "梵蒂冈",
	"VC":      "圣文森特和格林纳丁斯",
	"VE":      "委内瑞拉",
	"VG":      "英属维尔京群岛",
	"VI":      "美属维尔京群岛",
	"VN":      "越南",
	"VU":      "瓦努阿图",
	"WF":      "瓦利斯和富图纳",
	"WS":      "萨摩亚",
	"YE":      "也门",
	"YT":      "马约特",
	"ZA":      "南非",
	"ZM":      "赞比亚",
	"ZW":      "津巴布韦",
	"UNKNOWN": "未知",
}
