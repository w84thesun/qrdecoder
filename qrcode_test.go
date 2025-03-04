package qrcode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		in, out string
	}{
		// {in: "qrcode.jpg", out: "http://www.imdb.com/title/tt2948356/"},
		// {in: "qrcode.png", out: "http://weixin.qq.com/r/2fKmvj-EkmLtrXvd96fL"},
		// {in: "qrcode1.png", out: "http://weixin.qq.com/r/2fKmvj-EkmLtrXvd96fL"},
		// {in: "qrcode4.png", out: "http://www.example.org"},
		{in: "qrcode5.png", out: "a"},
		// 		{in: "qrcode6.png", out: "abcdefg"},
		// 		{in: "qrcode7.png", out: "abcdefg"},
		// 		{in: "qrcode8.png", out: "中文"},
		// 		{in: "qrcode9.png", out: "abcdefg"},
		// 		{in: "qrcode10.png", out: "abcdefghijklmnopqrstuvwxyz"},
		// 		{in: "qrcode11.png", out: `PProf是一个CPU分析器( cpu profiler)， 它是gperftools工具的一个组件， 由Google工程师为分析多线程的程序所开发。
		// Go标准库中的pprof package通过HTTP的方式为pprof工具提供数据。
		// (译者注：不止这个包，runtime/pprof还可以为控制`},
		// 		{in: "qrcode13.png", out: `PProf是一个CPU分析器( cpu profiler)， 它是gperftools工具的一个组件， 由Google工程师为分析多线程的程序所开发。
		// Go标准库中的pprof package通过HTTP的方式为pprof工具提供数据。
		// (译者注：不止这个包，runtime/pprof还可以为控制`},
		// 		{in: "qrcode14.jpeg", out: "AEL-10007-78402-01XXB45EBF1163C414B24AFD062B008024605AA3AB554463147C78A4B0ECA23B1DA80"},
		// 		{in: "qrcode15.jpeg", out: "AEL-10007-78379-02XX524DBEEF63C414A830F3062A0047E2404ECEAF6E8C1DCCF9E0ED2484355C22EF0"},
		// {in: "qrcode16.png", out: "otpauth://totp/MLX-1c17dc67-5475-4f3a-9a0b-c26166a6276e"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			f, err := os.Open(filepath.Join("example", tt.in))
			if err != nil {
				t.Fatal(err)
			}

			qr, err := Decode(f)

			require.NoError(t, err)

			require.Equal(t, tt.out, qr.Content)
		})
	}
}
