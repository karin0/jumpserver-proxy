package main

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
)

func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}

type any = interface{}
type jObject = map[string]any
type jArray = []any

var gCookies string
var gExtraQs string
var urlBase string
var client = &http.Client{}

func expiredIf(err error) {
	if err != nil {
		panic("Cookie 可能已过期，请尝试重新获取: " + err.Error())
	}
}

func expiredIfNot(ok bool) {
	if !ok {
		panic("Cookie 可能已过期，请尝试重新获取")
	}
}

func requestJson(path string, qs string) any {
	req, err := http.NewRequest("GET", getUrl(path, qs), nil)
	expiredIf(err)
	req.Header.Add("Cookie", gCookies)
	resp, err := client.Do(req)
	expiredIf(err)
	var v any
	err = json.NewDecoder(resp.Body).Decode(&v)
	expiredIf(err)
	return v
}

func asArray(o any) jArray {
	r, err := o.(jArray)
	expiredIfNot(err)
	return r
}

func asObject(o any) jObject {
	r, err := o.(jObject)
	expiredIfNot(err)
	return r
}

func asString(o any) string {
	r, err := o.(string)
	expiredIfNot(err)
	return r
}

func getUrl(path string, qs string) string {
	return urlBase + path + "?" + gExtraQs + qs
}

func getUserId(targetId string) string {
	ra := asArray(requestJson("/api/v1/perms/users/assets/" + targetId + "/system-users/", "cache_policy=1"))
	ro := asObject(ra[0])
	return asString(ro["id"])
}

func getTargetId() string {
	ro := asObject(requestJson("/api/v1/perms/users/assets/", "offset=0&limit=15&display=1&draw=1"))
	ra := asArray(ro["results"])
	ro = asObject(ra[0])
	return asString(ro["id"])
}

func getWsConn(host, cookies, extraQs string, putUrl bool) *websocket.Conn {
	gCookies = cookies
	gExtraQs = extraQs + "&"
	hu, err := url.Parse(host)
	panicIf(err)
	if hu.Host == "" {
		hu, err = url.Parse("https://" + host)
		panicIf(err)
	}
	wsPath := "/koko/ws/terminal/"
	urlBase = "https://" + hu.Host
	if hu.Path != "" {
		lead := strings.TrimSuffix(hu.Path, "/")
		urlBase += lead
		wsPath = lead + wsPath
	}

	targetId := getTargetId()
	userId := getUserId(targetId)
	u := url.URL{Scheme: "wss", Host: hu.Host, Path: strings.Replace(wsPath, "http", "ws", -1), RawQuery: "target_id=" + targetId + "&type=asset&system_user_id=" + userId}
	rawUrl := u.String()
	if putUrl {
		log.Print("正在连接... ", rawUrl)
	} else {
		log.Print("正在连接...")
	}

	h := http.Header{}
	h.Set("Cookie", cookies)

	websocket.DefaultDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	conn, _, err := websocket.DefaultDialer.Dial(rawUrl, h)
	if err != nil {
		panic("连接失败，告辞。" + err.Error())
	}
	return conn
}
