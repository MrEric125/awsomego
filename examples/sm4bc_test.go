package main

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/tjfoc/gmsm/sm4"
)

func TestSm4Bc(t *testing.T) {
	// 16 字节密钥（128 位），实际项目中绝不能硬编码
	key := []byte("1234567890abcdef")
	// 16 字节 IV，加密时每次都应不同，解密时使用加密时的 IV
	//iv := []byte("fedcba0987654321")
	stry := "{\\\"auth_type\\\":1,\\\"common_params\\\":{\\\"1\\\":[],\\\"2\\\":[],\\\"3\\\":[],\\\"4\\\":[{\\\"name\\\":\\\"User-Agent\\\",\\\"value\\\":\\\"Coze/1.0\\\"}]},\\\"oauth_info\\\":\\\"\\\",\\\"public_plugin_name\\\":\\\"7637020739819798528\\\",\\\"sub_auth_type\\\":0}"

	plaintext := []byte(stry)

	// 加密：最后一个参数 true 表示加密
	ciphertext, err := sm4.Sm4Cbc(key, plaintext, true)
	if err != nil {
		panic(err)
	}
	encodeStr := hex.EncodeToString(ciphertext)

	fmt.Println("密文(hex):", encodeStr)

	ciphertext, err = hex.DecodeString(encodeStr)

	// 解密：最后一个参数 false 表示解密
	decrypted, err := sm4.Sm4Cbc(key, ciphertext, false)
	if err != nil {
		panic(err)
	}
	fmt.Println("解密后原文:", string(decrypted))
}
