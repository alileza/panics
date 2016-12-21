```go
import "github.com/alileza/panics"
```

## Configuration
```go
panics.SetOptions(&panics.Options{
	Env:             "TEST",
	SlackWebhookURL: "https://hooks.slack.com/services/blablabla/blablabla/blabla",
})
```


## Capture Panic on HTTP Handler
```go
http.HandleFunc("/", panics.CaptureHandler(func(w http.ResponseWriter, r *http.Request) {
	panic("Duh aku panik nih guys")
}))
```

## Capture Panic on httprouter handler
```go
router := httprouter.New()
router.POST("/", panics.CaptureHTTPRouterHandler(func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
    panic("Duh httprouter aku panik nih guys")
}))
```

## Capture Custom Error
```go
panics.Capture(
    "Deposit Anomaly",
    `{"user_id":123, "deposit_amount" : -100000000}`,
)
```
