package types

import (
	"time"
)

type Arguments struct {
	LimsHost          string
	LimsUser          string
	LimsPW            string
	LimsPubTop        string
	DateMode          bool
	StartDate         time.Time
	EndDate           time.Time
	ReqIdMode         bool
	ReqIds            []string
	JSONFileMode      bool
	JSONFilePath      string
	PublisherFileMode bool
	PublisherFilePath string
	SmileServiceMode  bool
	CMOReqs           bool // Only fetch CMO Requests
	NatsUrl           string
	NatsConName       string
	NatsConPw         string
	NatsKeyPath       string
	NatsTrustPath     string
	SmileRequestUrl   string
	SmilePubTop       string
}

type Config struct {
	Name string
	Type string
	Path string
}
