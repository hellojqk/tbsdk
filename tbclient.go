package tbsdk

import (
	"crypto/hmac"
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"hash"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// Client 客户端对象
type Client struct {
	appKey     string        //key
	appSecret  string        //secret
	APIAddr    string        //接口地址
	Timeout    time.Duration //请求超时时间
	Formart    string        //返回数据结构
	SignMethod string        //签名方法
}

// BaseRequest 基础请求接口
type BaseRequest interface {
	GetAPIName() string
	GetParams() map[string]interface{}
}

const (
	APIAddr         = "http://gw.api.taobao.com/router/rest"    //正式服务器地址
	APIAddrTest     = "http://gw.api.tbsandbox.com/router/rest" //沙箱服务器地址
	Formart_Json    = "json"                                    //返回数据类型
	Formart_XML     = "xml"                                     //返回数据类型
	SignMethod_HMAC = "hmac"                                    //请求签名方法
	SignMethod_MD5  = "md5"                                     //请求签名方法
)

// NewClientWithAddr 创建自定义地址客户端
func NewClientWithAddr(appkey, appsecret, apiAddr string) *Client {
	client := new(Client)
	client.appKey = appkey
	client.appSecret = appsecret
	client.APIAddr = apiAddr
	client.Timeout = 30 * time.Second
	client.Formart = Formart_Json
	client.SignMethod = SignMethod_MD5
	return client
}

// NewClient 创建淘宝客户端
func NewClient(appkey, appsecret string) *Client {
	return NewClientWithAddr(appkey, appsecret, APIAddr)
}

// DoPost post数据
func (cli *Client) DoPost(req BaseRequest, session string) ([]byte, error) {
	return cli.DoPostObj(req, session, nil)
}

var clientPool = sync.Pool{New: func() interface{} {
	return interface{}(&http.Client{})
}}

func ClientPoolPut(c *http.Client) {
	c.Transport = nil
	c.Timeout = 0
	clientPool.Put(c)
}

// var paramPool = sync.Pool{New: func() interface{} {
// 	m := make(map[string]string, 20)
// 	return interface{}(m)
// }}

// func ParamPoolPut(m map[string]string) {
// 	for k := range m {
// 		delete(m, k)
// 	}
// 	paramPool.Put(m)
// }

// DoPostObj 请求方法
func (cli *Client) DoPostObj(req BaseRequest, session string, v interface{}) ([]byte, error) {
	tr := &http.Transport{
		DisableCompression: true,
	}

	httpCli := http.Client{}
	httpCli.Transport = tr
	httpCli.Timeout = cli.Timeout

	var param = make(map[string]string, 10)
	param["method"] = req.GetAPIName()
	param["app_key"] = cli.appKey
	// if session != "" {
	param["session"] = session
	// }
	param["timestamp"] = time.Now().Format("2006-01-02 15:04:05")
	param["format"] = cli.Formart
	param["v"] = "2.0"
	// partner_id	String	否	合作伙伴身份标识。
	// target_app_key	String	否	被调用的目标AppKey，仅当被调用的API为第三方ISV提供时有效。
	if cli.Formart == Formart_Json {
		param["simplify"] = "true"
	}
	param["sign_method"] = cli.SignMethod

	var reqParam = req.GetParams()
	for k, v := range reqParam {
		param[k] = GetValueStr(v)
	}

	param["sign"] = SignStringMap(param, cli.appSecret, cli.SignMethod)

	var postData = GetParamStr(param)
	// log.Printf("postData\t%s\n\n", postData)
	httpReq, err := http.NewRequest("POST", cli.APIAddr, strings.NewReader(postData))
	if err != nil {
		return nil, err
	}
	err = httpReq.ParseForm()
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// resp, err := httpCli.Do(httpReq)
	// if err != nil {
	// 	return nil, err
	// }
	// defer resp.Body.Close()
	// byteResult, err := ioutil.ReadAll(resp.Body)
	// if v == nil || err != nil {
	// 	return byteResult, err
	// }
	// log.Printf("byteResult:%s\n", byteResult)
	byteResult := make([]byte, 2000)
	byteResult[0] = '{'
	firstStrIndex := 0
	for _, bt := range byteResult {
		if bt != ' ' && firstStrIndex == 0 {
			if bt == '{' {
				err = json.Unmarshal(byteResult, v)
			} else {
				err = xml.Unmarshal(byteResult, v)
			}
			return nil, err
		}
	}
	return byteResult, err
}

func (cli *Client) DoPostObjPool(req BaseRequest, session string, v interface{}) ([]byte, error) {
	tr := &http.Transport{
		DisableCompression: true,
	}
	httpCli := clientPool.Get().(*http.Client)
	httpCli.Transport = tr
	httpCli.Timeout = cli.Timeout
	defer ClientPoolPut(httpCli)

	// param := paramPool.Get().(map[string]string)
	// defer ParamPoolPut(param)
	var param = make(map[string]string, 10)
	param["method"] = req.GetAPIName()
	param["app_key"] = cli.appKey
	// if session != "" {
	param["session"] = session
	// }
	param["timestamp"] = time.Now().Format("2006-01-02 15:04:05")
	param["format"] = cli.Formart
	param["v"] = "2.0"
	// partner_id	String	否	合作伙伴身份标识。
	// target_app_key	String	否	被调用的目标AppKey，仅当被调用的API为第三方ISV提供时有效。
	if cli.Formart == Formart_Json {
		param["simplify"] = "true"
	}
	param["sign_method"] = cli.SignMethod

	var reqParam = req.GetParams()
	for k, v := range reqParam {
		param[k] = GetValueStr(v)
	}

	param["sign"] = SignStringMapPool(param, cli.appSecret, cli.SignMethod)

	var postData = GetParamStr(param)
	// log.Printf("postData\t%s\n\n", postData)
	httpReq, err := http.NewRequest("POST", cli.APIAddr, strings.NewReader(postData))
	if err != nil {
		return nil, err
	}
	err = httpReq.ParseForm()
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// resp, err := httpCli.Do(httpReq)
	// if err != nil {
	// 	return nil, err
	// }
	// defer resp.Body.Close()
	// byteResult, err := ioutil.ReadAll(resp.Body)
	// if v == nil || err != nil {
	// 	return byteResult, err
	// }
	// log.Printf("byteResult:%s\n", byteResult)
	byteResult := make([]byte, 2000)
	byteResult[0] = '{'
	firstStrIndex := 0
	for _, bt := range byteResult {
		if bt != ' ' && firstStrIndex == 0 {
			if bt == '{' {
				err = json.Unmarshal(byteResult, v)
			} else {
				err = xml.Unmarshal(byteResult, v)
			}
			return nil, err
		}
	}
	return byteResult, err
}

// GetValueStr 将interface{}数据转换成string
func GetValueStr(v interface{}) string {
	if t, ok := v.(time.Time); ok {
		return t.Format("2006-01-02 15:04:05")
	}
	return fmt.Sprint(v)
}

// GetParamStr 将map[string]string数据转换成string
func GetParamStr(params map[string]string) string {
	var sb strings.Builder
	for k, v := range params {
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(v)
		sb.WriteString("&")
	}
	return sb.String()[0 : sb.Len()-1]
}

// SignStringMap 对map进行淘宝签名
func SignStringMap(params map[string]string, appSecret string, signMethod string) string {
	var keys = make([]string, 0, len(params))
	for k, _ := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	// log.Printf("SignStringKeys\t%+v\n\n", keys)
	var sb strings.Builder
	for _, key := range keys {
		sb.WriteString(key)
		sb.WriteString(params[key])
	}
	// log.Printf("SignString\t%s\n\n", sb.String())
	return SignString(sb.String(), appSecret, signMethod)
}

// SignStringMap 对map进行淘宝签名
func SignStringMapPool(params map[string]string, appSecret string, signMethod string) string {
	var keys = make([]string, 0, 20)
	for k, _ := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	// log.Printf("SignStringKeys\t%+v\n\n", keys)
	var sb strings.Builder
	for _, key := range keys {
		sb.WriteString(key)
		sb.WriteString(params[key])
	}
	// log.Printf("SignString\t%s\n\n", sb.String())
	return SignStringPool(sb.String(), appSecret, signMethod)
}

// SignString 字符串签名发放
func SignString(params string, appSecret string, signMethod string) string {
	if signMethod == SignMethod_MD5 {
		h := md5.New()
		h.Write([]byte(appSecret))
		h.Write([]byte(params))
		h.Write([]byte(appSecret))
		return strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
	} else {
		h := hmac.New(md5.New, []byte(appSecret))
		h.Write([]byte(params))
		return strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
	}
}

var md5Pool = sync.Pool{
	New: func() interface{} {
		return interface{}(md5.New())
	}}

func SignStringPool(params string, appSecret string, signMethod string) string {

	if signMethod == SignMethod_MD5 {
		h := md5Pool.Get().(hash.Hash)
		defer func() {
			h.Reset()
			md5Pool.Put(h)
		}()
		h.Write([]byte(appSecret))
		h.Write([]byte(params))
		h.Write([]byte(appSecret))
		return strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
	} else {
		h := hmac.New(md5.New, []byte(appSecret))
		h.Write([]byte(params))
		return strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
	}
}

// func GetResonseString(bytes []byte, err error) (string, error) {
// 	if err != nil {
// 		return "", err
// 	}
// 	return string(bytes), err
// }

// func GetResonseObject(v interface{}, bytes []byte, err error) error {
// 	if err != nil {
// 		return err
// 	}
// 	firstStrIndex := 0
// 	for _, bt := range bytes {
// 		if bt != ' ' && firstStrIndex == 0 {
// 			if bt == '{' {
// 				err = json.Unmarshal(bytes, v)
// 			} else {
// 				err = xml.Unmarshal(bytes, v)
// 			}
// 			return err
// 		}
// 	}
// 	return nil
// }
