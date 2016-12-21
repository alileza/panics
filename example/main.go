package main

import (
	"log"
	"net/http"

	"github.com/alileza/panics"
)

func main() {
	panics.SetOptions(&panics.Options{
		Env:      "TEST",
		Filepath: "./",
		// SlackWebhookURL: "https://hooks.slack.com/services/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",
	})
	panics.Capture("test capture", `ulala one two three`)
	// router := httprouter.New()
	// router.POST("/", panics.CaptureHTTPRouterHandler(func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// 	panic("Duh httprouter aku panik nih guys")
	// }))
	//
	// http.HandleFunc("/", panics.CaptureHandler(func(w http.ResponseWriter, r *http.Request) {
	// 	panic("Duh httprouter aku panik nih guys")
	// }))

	log.Fatal(http.ListenAndServe(":9233", nil))
}
