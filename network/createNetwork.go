package network

import "fmt"

func Create() {
	client := GetClient()
	fmt.Println(client)
}
