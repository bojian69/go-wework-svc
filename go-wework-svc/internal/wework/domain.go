package wework

import (
	"encoding/xml"
	"errors"
)

// CallbackQuery 回调请求的 URL 查询参数
type CallbackQuery struct {
	MsgSignature string
	Timestamp    string
	Nonce        string
	Echostr      string // 仅 GET 验证时使用
}

// EncryptedBody POST 请求的加密 XML 消息体
type EncryptedBody struct {
	XMLName    xml.Name `xml:"xml"`
	ToUserName string   `xml:"ToUserName"`
	AgentID    string   `xml:"AgentID"`
	Encrypt    string   `xml:"Encrypt"`
}

// Message 解密后的企业微信消息
type Message struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
	MsgID        string   `xml:"MsgId"`
	AgentID      int64    `xml:"AgentID"`
}

// MsgType 消息类型常量
const (
	MsgTypeText  = "text"
	MsgTypeImage = "image"
	MsgTypeEvent = "event"
)

// ErrInvalidSignature 签名验证失败错误
var ErrInvalidSignature = errors.New("invalid signature")
