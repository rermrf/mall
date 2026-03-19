package ioc

import "github.com/rermrf/mall/pkg/snowflake"

func InitSnowflakeNode() *snowflake.Node {
	node, err := snowflake.NewNode(3)
	if err != nil {
		panic(err)
	}
	return node
}
