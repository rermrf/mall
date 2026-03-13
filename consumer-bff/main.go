package main

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	initViper()
	app := InitApp()
	addr := viper.GetString("server.addr")
	fmt.Printf("consumer-bff 启动于 %s\n", addr)
	if err := app.Server.Run(addr); err != nil {
		panic(fmt.Errorf("启动服务失败: %w", err))
	}
}

func initViper() {
	configFile := pflag.String("config", "config/dev.yaml", "配置文件路径")
	pflag.Parse()
	viper.SetConfigFile(*configFile)
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("读取配置文件失败: %w", err))
	}
}
