package main

import (
	e_taobao_sdk "eplugs/e-taobao-sdk"
	"fmt"
)

func main() {
	e_taobao_sdk.AppKey = ""
	e_taobao_sdk.AppSecret = ""
	e_taobao_sdk.Router = "http://gw.api.taobao.com/router/rest"
	res, err := e_taobao_sdk.Execute("taobao.tbk.item.get", e_taobao_sdk.Parameter{
		"fields": "num_iid,title",
		"q":      "制服",
		"cat":    "69,96",
	})

	if err != nil {
		panic(err)
	}

	fmt.Println("商品数量:", res.Get("tbk_item_get_response").Get("total_results").MustInt())
	var imtes []interface{}
	imtes, _ = res.Get("tbk_item_get_response").Get("results").Get("n_tbk_item").Array()
	for _, v := range imtes {
		fmt.Println("======")
		item := v.(map[string]interface{})
		fmt.Println("商品名称:", item["title"])
		fmt.Println("商品价格:", item["reserve_price"])
		fmt.Println("商品链接:", item["item_url"])
	}
}
