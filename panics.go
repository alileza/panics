package panics

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"

	"github.com/julienschmidt/httprouter"
	"strings"
)

var (
	client *http.Client
	file   *os.File

	env             string
	ipAddress       string
	filepath        string
	slackWebhookURL string
	tagString       string
)

type Tags map[string]string

type Options struct {
	EnvKey          string
	ShowIP          bool
	Filepath        string
	SentryDSN       string
	SlackWebhookURL string
	Tags            Tags
}

func SetOptions(o *Options) {
	filepath = o.Filepath
	slackWebhookURL = o.SlackWebhookURL

	env = o.EnvKey

	var err error

	if o.ShowIP {
		ipAddress, err = findMyIP()
		if err != nil {
			log.Printf("[panics] cannot find IP, %s", err.Error())
		}
	}

	fp := filepath + "/panics.log"
	file, err = os.OpenFile(fp, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		log.Printf("[panics] failed to open file %s", fp)
	}

	var tmp []string
	for key, val := range o.Tags {
		tmp = append(tmp, fmt.Sprintf("`%s: %s`", key, val))
	}
	tagString = strings.Join(tmp, " | ")
}

func findMyIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}

		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("not connected to networks")
}

func init() {
	client = new(http.Client)
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

// Capture will publish any errors
func Capture(err string, message string) {
	publishError(errors.New(err), []byte(message), false)
}

func publishError(errs error, reqBody []byte, withStackTrace bool) {
	errorStack := debug.Stack()
	t := fmt.Sprintf(`[%s|%s] *%s*`, env, ipAddress, errs.Error())

	if len(tagString) > 0 {
		t = t + " | " + tagString
	}

	if reqBody != nil {
		t = t + (" ```" + string(reqBody) + "```")
	}
	if errorStack != nil && withStackTrace {
		t = t + (" ```" + string(errorStack) + "```")
	}
	if slackWebhookURL != "" {
		go postToSlack(t)
	}
	if file != nil {
		go file.Write([]byte(t + "\r\n"))
	}
}

func postToSlack(t string) {
	b, _ := json.Marshal(map[string]string{
		"text": t,
	})

	req, err := http.NewRequest("POST", slackWebhookURL, bytes.NewBuffer(b))
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
