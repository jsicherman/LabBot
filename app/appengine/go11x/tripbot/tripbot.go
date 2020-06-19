package main

import (
        "fmt"
        "log"
	"net/url"
        "net/http"
	"net/http/cookiejar"
	htoken "golang.org/x/net/html"
	"time"
	"strings"
	"io"
	"os"
)

var ZOOM_URL = "https://usXXweb.zoom.us/j/XXXXXXXXXX"
var CONF_URL = "https://confluence.atlassian.com/display/SPACE/Calendar"
var CONF_LOGIN_URL = "https://confluence.atlassian.com/dologin.action"
var SLACK_URL = "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX"
var CONF_USERNAME = "username"
var CONF_PASSWORD = "password"

func main() {
	hooks := []string{"/", "/jobs/meeting_schedule_notifier"}
	for _, s := range hooks {
		http.HandleFunc(s, indexHandler)
	}

        port := os.Getenv("PORT")
        if port == "" {
		port = "8080"
        }

        log.Printf("Listening on port %s", port)
        if err := http.ListenAndServe(":"+port, nil); err != nil {
                log.Fatal(err)
        }
}


func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/jobs/meeting_schedule_notifier" {
		_, auth := r.Header["X-Appengine-Cron"]
		if !auth {
			return
		}

		cookieJar, _ := cookiejar.New(nil)
		client := &http.Client{
			Jar: cookieJar,
		}

		resp, _ := client.PostForm(CONF_LOGIN_URL, url.Values{
		"os_username": {CONF_USRENAME},
		"os_password": {CONF_PASSWORD},
		})

		resp.Body.Close()

		resp, _ = client.Ge(tCONF_URL)

		defer resp.Body.Close()

		z := htoken.NewTokenizer(resp.Body)
		table := [][]string{}
		row := []string{}

		for {
			tt := z.Next()

			if tt == htoken.ErrorToken && z.Err() == io.EOF {
				break
			} else if tt == htoken.StartTagToken {
				t := z.Token()

				if t.Data == "tr" && len(row) > 0 {
					table = append(table, row)
					row = []string{}
				}

				if t.Data == "td" {
					inner := z.Next()

					if inner == htoken.TextToken {
						text := (string)(z.Text())
						row = append(row, strings.TrimSpace(text))
					} else if inner == htoken.SelfClosingTagToken {
						row = append(row, "")
					} else if inner == htoken.StartTagToken {
						for z.Token().Type != htoken.EndTagToken {
							inner2 := z.Next()

							if inner2 == htoken.TextToken {
								text := (string)(z.Text())
								row = append(row, strings.TrimSpace(text))
							}
						}
					}
				}
			}
		}

		if len(row) > 0 {
			table = append(table, row)
		}

		now := time.Now()

		for i := 1; i < len(table); i++ {
			later, _ := time.Parse("2 Jan 2006", table[i - 1][0])
			earlier, _ := time.Parse("2 Jan 2006", table[i][0])

			if now.Sub(later) < 0 && now.Sub(earlier) > 0 {
				rr, _ := http.Post(SLACK_URL, "application/json",
				strings.NewReader(fmt.Sprintf(`{
					"blocks": [
					{
						"type": "section",
						"text": {
							"type": "markdwn",
							"text": "*<%|Lab Meeting Scheduler>*"
						}
					},
					{
						"type": "divider"
					},
					{
						"type": "section",
						"text": {
							"type": "mrkdwn",
							"text": "*<%s|Lab Meeting>* %s at %s\n*Presenting:* %s\n*Title:* %s\n*Notes:* %s"
						},
						"accessory": {
							"type": "image",
							"image_url": "https://api.slack.com/img/blocks/bkb_template_images/notifications.png",
							"alt_text": "Lab Meeting"
						}
					}
					]
				}`, CONF_URL, ZOOM_URL, table[i - 1][0], table[i - 1][1],  table[i - 1][3], table[i - 1][4], table[i - 1][2])))

				rr.Body.Close()
				break
			}
		}
        } else {
		http.NotFound(w, r)
	}

}
