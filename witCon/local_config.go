package main

import (
	"fmt"
	"github.com/naoina/toml"
	"os"
)

type LocalConfig struct {
	Nodes              uint   `json:"nodes"`
	LogLvl             uint   `json:"logLvl"`
	Consensus          uint   `json:"consensus"`
	TxAmount           uint   `json:"txAmount"`
	TxSize             uint   `json:"txSize"`
	ViewChangeDuration uint   `json:"viewChangeDuration"`
	Rtt                uint   `json:"rtt"`
	Path               string `json:"path"`
	NodePath           string `json:"nodePath"`
	IP                 string `json:"IP"`
	NodeName           string `json:"nodeName"`
	PrePack            bool   `json:"prePack"`
	ShardVerify        bool   `json:"ShardVerify"`
	Schnorr            bool   `json:"schnorr"`
	SignVerifyCore     uint   `json:"signVerifyCore"`
	ShardCount         uint   `json:"shardCount"`
	ShardVerifyCore    uint   `json:"shardVerifyCore"`
	TxPath             string `json:"txPath"`
	VerifyCount        uint   `json:"verifyCount"`
}

func GetLocalConfig(path string) *LocalConfig {
	_, err := os.Stat(path)
	config := new(LocalConfig)
	if err == nil {
		file, err := os.Open(path)
		if err != nil {
			fmt.Printf("Failed to read config file: %v", "err", err)
		}
		defer file.Close()
		if err := toml.NewDecoder(file).Decode(&config); err != nil {
			fmt.Printf("invalid config file: %v", "err", err)
		}
		return config
	} else {
		fmt.Printf("nil config file: %v", "err", err)
	}
	return nil
}
