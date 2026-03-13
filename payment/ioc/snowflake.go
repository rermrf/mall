package ioc

import "github.com/rermrf/mall/pkg/snowflake"

func InitSnowflakeNode() *snowflake.Node {
	node, err := snowflake.NewNode(2)
	if err != nil {
		panic(err)
	}
	return node
}
