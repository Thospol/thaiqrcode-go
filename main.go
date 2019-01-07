package main

import (
	"thaiqr-go/internal/qr"
)

func main() {
	qr.ConvertStringToMap("000201010211021649570300000080620415520473000001046153134300764005204460000000000111565204530953037645802TH5918KBANK Merchant UAT6007bangkok62210505213460708709999955125000412340106416971020312363048169")
	// port := flag.String("p", "8031", "port number")

	// ro := handler.Routes{}
	// hdlr := ro.InitRoute()
	// srv := &http.Server{
	// 	Addr:    fmt.Sprint(":", *port),
	// 	Handler: hdlr,
	// }

	// if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
	// 	log.Panicf("listen: %s\n", err)
	// }

}
