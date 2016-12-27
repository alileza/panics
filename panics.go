package panics

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"

	"strings"

	"github.com/julienschmidt/httprouter"
)

var (
	client *http.Client
	file   *os.File

	env          string
	filepath     string
	slackWebhook SlackWebhook
	tagString    string
)

type Tags map[string]string
type SlackWebhook struct {
	URL     string
	Channel string
}

type Options struct {
	Env          string
	Filepath     string
	SentryDSN    string
	SlackWebhook SlackWebhook
	Tags         Tags
}

func SetOptions(o *Options) {
	filepath = o.Filepath
	slackWebhook = o.SlackWebhook

	env = o.Env

	if filepath != "" {
		var err error
		fp := filepath + "/panics.log"
		file, err = os.OpenFile(fp, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			log.Printf("[panics] failed to open file %s", fp)
		}
	}

	var tmp []string
	for key, val := range o.Tags {
		tmp = append(tmp, fmt.Sprintf("`%s: %s`", key, val))
	}
	tagString = strings.Join(tmp, " | ")
}

func init() {
	client = new(http.Client)

	env = os.Getenv("TKPENV")
}

// CaptureHandler handle panic on http handler.
func CaptureHandler(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		request, _ := httputil.DumpRequest(r, true)
		defer func() {
			r := recover()

			if r != nil {
				switch t := r.(type) {
				case string:
					err = errors.New(t)
				case error:
					err = t
				default:
					err = errors.New("Unknown error")
				}

				publishError(err, request, true)

				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}()
		h.ServeHTTP(w, r)
	}
}

// CaptureHTTPRouterHandler handle panic on httprouter handler.
func CaptureHTTPRouterHandler(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		var err error
		request, _ := httputil.DumpRequest(r, true)
		defer func() {
			r := recover()

			if r != nil {
				switch t := r.(type) {
				case string:
					err = errors.New(t)
				case error:
					err = t
				default:
					err = errors.New("Unknown error")
				}

				publishError(err, request, true)

				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}()
		h(w, r, ps)
	}
}

// CaptureNegroniHandler handle panic on negroni handler.
func CaptureNegroniHandler(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	var err error
	request, _ := httputil.DumpRequest(r, true)
	defer func() {
		r := recover()

		if r != nil {
			switch t := r.(type) {
			case string:
				err = errors.New(t)
			case error:
				err = t
			default:
				err = errors.New("Unknown error")
			}

			publishError(err, request, true)

			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}()
	next(w, r)
}

// Capture will publish any errors
func Capture(err string, message ...string) {
	var tmp string
	for i, val := range message {
		if i == 0 {
			tmp += val
		} else {
			tmp += fmt.Sprintf("\n\n%s", val)
		}
	}

	publishError(errors.New(err), []byte(tmp), false)
}

func publishError(errs error, reqBody []byte, withStackTrace bool) {
	var text string
	var snip string
	var buffer bytes.Buffer
	errorStack := debug.Stack()
	buffer.WriteString(fmt.Sprintf(`[%s] *%s*`, env, errs.Error()))

	if len(tagString) > 0 {
		buffer.WriteString(" | " + tagString)
	}

	if reqBody != nil {
		buffer.WriteString(fmt.Sprintf(" ```%s``` ", string(reqBody)))
	}
	text = buffer.String()

	if errorStack != nil && withStackTrace {
		snip = fmt.Sprintf("```\n%s```", string(errorStack))
	}

	if slackWebhook.URL != "" {
		go postToSlack(buffer.String(), snip)
	}
	if file != nil {
		go func() {
			file.Write([]byte(text))
			file.Write([]byte(snip + "\r\n"))
		}()
	}
}

func postToSlack(text, snip string) {
	payload := map[string]interface{}{
		"text": text,
		"attachments": []map[string]interface{}{
			map[string]interface{}{
				"text":      snip,
				"mrkdwn_in": []string{"text"},
			},
		},
	}
	if slackWebhook.Channel != "" {
		payload["channel"] = slackWebhook.Channel
	}
	b, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", slackWebhook.URL, bytes.NewBuffer(b))
	if err != nil {
		log.Printf("[panics] error on capturing error : %s \n", err.Error())
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[panics] error on capturing error : %s \n", err.Error())
	}

	if resp.StatusCode >= 300 {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("[panics] error on capturing error : %s \n", err)
			return
		}
		log.Printf("[panics] error on capturing error : %s \n", string(b))
	}
}
