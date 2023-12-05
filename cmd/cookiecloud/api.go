package cookiecloud

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/sagan/ptool/util"
	"github.com/sagan/ptool/util/crypto"
)

type Ccdata_struct struct {
	Domain string
	Uuid   string
	Sites  []string
	Data   *CookiecloudData
}

type Cookie struct {
	Domain string
	Name   string
	Value  string
	Path   string
}

type CookieCloudBody struct {
	Uuid      string `json:"uuid,omitempty"`
	Encrypted string `json:"encrypted,omitempty"`
}

type CookiecloudData struct {
	// host => [{name,value,domain}...]
	Cookie_data map[string][]map[string]any `json:"cookie_data"`
}

func GetCookiecloudData(server string, uuid string, password string, proxy string) (*CookiecloudData, error) {
	if server == "" || uuid == "" || password == "" {
		return nil, fmt.Errorf("all params of server,uuid,password must be provided")
	}
	if !strings.HasSuffix(server, "/") {
		server += "/"
	}
	var httpClient *http.Client
	if proxy != "" {
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("failed to parse proxy %s: %v", proxy, err)
		}
		httpClient = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyUrl),
			},
		}
	}
	var data *CookieCloudBody
	err := util.FetchJson(server+"get/"+uuid, &data, httpClient, "", "", nil)
	if err != nil || data == nil {
		return nil, fmt.Errorf("failed to get cookiecloud data: err=%v, null data=%t", err, data == nil)
	}
	keyPassword := crypto.Md5String(uuid, "-", password)[:16]
	decrypted, err := crypto.DecryptCryptoJsAesMsg(keyPassword, data.Encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: err=%v", err)
	}
	var cookiecloudData *CookiecloudData
	err = json.Unmarshal(decrypted, &cookiecloudData)
	if err != nil || cookiecloudData == nil {
		return nil, fmt.Errorf("failed to parse decrypted data as json: err=%v", err)
	}
	return cookiecloudData, nil
}

func (cookiecloudData *CookiecloudData) GetEffectiveCookie(urlOrHost string) (string, error) {
	hostname := ""
	path := "/"
	if util.IsUrl(urlOrHost) {
		urlObj, err := url.Parse(urlOrHost)
		if err != nil {
			return "", fmt.Errorf("arg is not a valid url: %v", err)
		}
		hostname = urlObj.Hostname()
		path = urlObj.Path
	}
	if hostname == "" {
		return "", fmt.Errorf("hostname can not be empty")
	}
	effectiveCookies := []*Cookie{}
	keys := []string{hostname, "." + hostname}
	for _, key := range keys {
		cookies, ok := cookiecloudData.Cookie_data[key]
		if !ok {
			continue
		}
		for _, cookie := range cookies {
			if cookie == nil {
				continue
			}
			cookieDomain, _ := cookie["domain"].(string)
			if cookieDomain != hostname && cookieDomain != "."+hostname {
				continue
			}
			cookiePath, _ := cookie["path"].(string)
			if cookiePath == "" {
				cookiePath = "/"
			}
			if !strings.HasPrefix(path, cookiePath) {
				continue
			}
			// cookiecloud 导出的 cookies 里的 expirationDate 为 float 类型。意义不明确，暂不使用。
			cookieName, _ := cookie["name"].(string)
			cookieValue, _ := cookie["value"].(string)
			// RFC 似乎允许 empty cookie ?
			if cookieName == "" || cookieValue == "" {
				continue
			}
			effectiveCookies = append(effectiveCookies, &Cookie{
				Domain: cookieDomain,
				Path:   cookiePath,
				Name:   cookieName,
				Value:  cookieValue,
			})
		}
	}
	if len(effectiveCookies) == 0 {
		return "", nil
	}
	sort.SliceStable(effectiveCookies, func(i, j int) bool {
		a := effectiveCookies[i]
		b := effectiveCookies[j]
		if a.Domain != b.Domain {
			return false
		}
		// longest path first
		if len(a.Path) != len(b.Path) {
			return len(a.Path) > len(b.Path)
		}
		return false
	})
	effectiveCookies = util.UniqueSliceFn(effectiveCookies, func(cookie *Cookie) string {
		return cookie.Name
	})
	cookieStr := ""
	sep := ""
	for _, cookie := range effectiveCookies {
		cookieStr += sep + cookie.Name + "=" + cookie.Value
		sep = "; "
	}
	return cookieStr, nil
}
