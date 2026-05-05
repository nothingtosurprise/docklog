package main

import (
	"context"
	"fmt"
	"reflect"
	"github.com/moby/moby/client"
)

func main() {
	cli, _ := client.NewClientWithOpts(client.FromEnv)
	res, _ := cli.ContainerList(context.Background(), client.ContainerListOptions{})
	val := reflect.ValueOf(res)
	typ := val.Type()
	fmt.Printf("Type: %s\n", typ)
	if val.Kind() == reflect.Struct {
		for i := 0; i < val.NumField(); i++ {
			fmt.Printf("Field: %s, Value: %v\n", typ.Field(i).Name, val.Field(i))
		}
	} else {
		fmt.Printf("Value: %v\n", res)
	}
}

