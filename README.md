# Panics
Simple package to catch & notify your panic or exceptions via slack or save into files.

```go
import "github.com/tokopedia/panics"
```

## Configuration
```go
panics.SetOptions(&panics.Options{
	Env:             "TEST",
	SlackWebhookURL: "https://hooks.slack.com/services/blablabla/blablabla/blabla",
	Filepath: "/var/log/myapplication", // it'll generate panics.log
})
```

## Capture Custom Error
```go
panics.Capture(
    "Deposit Anomaly",
    `{"user_id":123, "deposit_amount" : -100000000}`,
)
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

## Example
### Slack Notification
![Notification Example](https://monosnap.com/file/Pjkw1uxjV8p0GnjevDwhHesUnTC2Ru.png)

# Authors

* [Ali Reza](mailto:alirezayahya@gmail.com)
* [Afid Eri](mailto:afid.eri@gmail.com) 
