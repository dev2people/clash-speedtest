package speedtester

import (
	"errors"
	"testing"
)

func TestGetCountryFlag(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{"US", "🇺🇸"},
		{"CN", "🇨🇳"},
		{"JP", "🇯🇵"},
		{"HK", "🇭🇰"},
		{"TW", "🇹🇼"},
		{"UK", "🇬🇧"},
		{"GB", "🇬🇧"},
		{"DE", "🇩🇪"},
		{"FR", "🇫🇷"},
		{"SG", "🇸🇬"},
		{"UNKNOWN", "🏳️"},
		{"", "🏳️"},
		{"INVALID", "🏳️"},
	}

	for _, tc := range tests {
		got := GetCountryFlag(tc.code)
		if got != tc.expected {
			t.Errorf("GetCountryFlag(%q) = %q, expected %q", tc.code, got, tc.expected)
		}
	}
}

func TestGetCountryName(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{"US", "美国"},
		{"CN", "中国"},
		{"JP", "日本"},
		{"HK", "香港"},
		{"TW", "台湾"},
		{"GB", "英国"},
		{"UK", "英国"},
		{"DE", "德国"},
		{"SG", "新加坡"},
		{"UNKNOWN", "未知"},
		{"INVALID_XYZ", "未知"},
	}

	for _, tc := range tests {
		got := GetCountryName(tc.code)
		if got != tc.expected {
			t.Errorf("GetCountryName(%q) = %q, expected %q", tc.code, got, tc.expected)
		}
	}
}

func TestExtractCountryFromNodeName(t *testing.T) {
	tests := []struct {
		nodeName string
		expCode  string
		expName  string
		expFlag  string
	}{
		// 1. Unicode 国旗测试
		{"🇺🇸 美国 01 节点 [BGP]", "US", "美国", "🇺🇸"},
		{"🇯🇵 Tokyo 02 - Hysteria2", "JP", "日本", "🇯🇵"},
		{"🇭🇰 香港专线 01", "HK", "香港", "🇭🇰"},
		{"🇹🇼 台湾 BGP [03]", "TW", "台湾", "🇹🇼"},
		{"🇸🇬 新加坡 05", "SG", "新加坡", "🇸🇬"},

		// 2. 中文城市与别名测试
		{"深港专线 01 | 1000M", "HK", "香港", "🇭🇰"},
		{"沪日 IEPL - 02", "JP", "日本", "🇯🇵"},
		{"台北 - 01 广港中继", "TW", "台湾", "🇹🇼"},
		{"东京 01 - 高速节点", "JP", "日本", "🇯🇵"},
		{"洛杉矶 CN2 GIA", "US", "美国", "🇺🇸"},
		{"圣何塞 9929", "US", "美国", "🇺🇸"},
		{"法兰克福 01 节点", "DE", "德国", "🇩🇪"},
		{"伦敦 01 [VIP]", "GB", "英国", "🇬🇧"},
		{"首尔 01 KT", "KR", "韩国", "🇰🇷"},
		{"狮城 01 流媒体解锁", "SG", "新加坡", "🇸🇬"},

		// 3. 英文城市与关键词测试
		{"Frankfurt-01-Vless", "DE", "德国", "🇩🇪"},
		{"Tokyo-Osaka-01", "JP", "日本", "🇯🇵"},
		{"Singapore-SG-01", "SG", "新加坡", "🇸🇬"},
		{"Hong Kong Premium 01", "HK", "香港", "🇭🇰"},
		{"United States West 01", "US", "美国", "🇺🇸"},

		// 4. 纯代码或关键词
		{"[HK] IEPL 01", "HK", "香港", "🇭🇰"},
		{"[US] Trojan 02", "US", "美国", "🇺🇸"},
		{"[JP] Shadowsocks", "JP", "日本", "🇯🇵"},
		{"[SG] Vmess", "SG", "新加坡", "🇸🇬"},

		// 5. 无法识别
		{"Node-12345-Direct", "UNKNOWN", "未知", "🏳️"},
		{"", "UNKNOWN", "未知", "🏳️"},
	}

	for _, tc := range tests {
		code, name, flag := ExtractCountryFromNodeName(tc.nodeName)
		if code != tc.expCode || name != tc.expName || flag != tc.expFlag {
			t.Errorf("ExtractCountryFromNodeName(%q) = (%q, %q, %q), expected (%q, %q, %q)",
				tc.nodeName, code, name, flag, tc.expCode, tc.expName, tc.expFlag)
		}
	}
}

func TestResolveCountry(t *testing.T) {
	// 1. API 正常返回
	loc := &IPLocation{Country: "United States", CountryCode: "US"}
	code, name, flag := ResolveCountry(loc, nil, "Random-Node-Name")
	if code != "US" || name != "美国" || flag != "🇺🇸" {
		t.Errorf("expected US, got %s, %s, %s", code, name, flag)
	}

	// 2. API 报错，从节点名中提取 flag
	code, name, flag = ResolveCountry(nil, errors.New("timeout"), "🇯🇵 日本 01")
	if code != "JP" || name != "日本" || flag != "🇯🇵" {
		t.Errorf("expected fallback to JP, got %s, %s, %s", code, name, flag)
	}

	// 3. API 报错，从节点名中提取城市名
	code, name, flag = ResolveCountry(nil, errors.New("api rate limited"), "香港 BGP 01")
	if code != "HK" || name != "香港" || flag != "🇭🇰" {
		t.Errorf("expected fallback to HK, got %s, %s, %s", code, name, flag)
	}

	// 4. API 报错且原名无特征 -> UNKNOWN
	code, name, flag = ResolveCountry(nil, errors.New("fail"), "Unknown-Server-X")
	if code != "UNKNOWN" || name != "未知" || flag != "🏳️" {
		t.Errorf("expected UNKNOWN, got %s, %s, %s", code, name, flag)
	}
}
