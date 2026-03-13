package ioc

import "github.com/rermrf/mall/pkg/snowflake"

func InitSnowflakeNode() *snowflake.Node {
	node, err := snowflake.NewNode(1)
	if err != nil {
		panic(err)
	}
	return node
}
