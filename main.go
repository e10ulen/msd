//	Package Overview.
//	mastodon cli tool
package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/comail/colog"
	"github.com/fatih/color"
	m "github.com/mattn/go-mastodon"
	"github.com/spf13/viper"
	"golang.org/x/net/html"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

//	MainFunction
//	ここで設定ファイルを読み込み、各コマンドに読み込んだ設定を渡している。
func main() {
	colog.Register()
	app := kingpin.New("md", "a Mastodon Application")
	viper.SetConfigName(".mastodon")
	viper.AddConfigPath("./")
	viper.AddConfigPath("$HOME/")
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	if err != nil {
		log.Print("w: ", err)
	}
	config := &m.Config{
		Server:       viper.GetString("server"),
		ClientID:     viper.GetString("clientid"),
		ClientSecret: viper.GetString("clientsecret"),
	}
	email := viper.GetString("emailaddress")
	pass := viper.GetString("password")

	cfg := m.NewClient(config)
	cfg.Authenticate(context.Background(), email, pass)
	timelineMastodon(app, cfg)
	tootMastodon(app, cfg)
	streamMastodon(app, cfg)
	updatenameMastodon(app, cfg)
	kingpin.MustParse(app.Parse(os.Args[1:]))
}

//	一言トゥートする事ができる。ダブルクオーテーションをつける必要はない。
func tootMastodon(app *kingpin.Application, cfg *m.Client) {
	cmd := app.Command("toot", "toot to mastodon")
	text := cmd.Arg("text", "text to toot").Strings()
	cmd.Action(func(c *kingpin.ParseContext) error {
		toot := strings.Join(*text, " ")
		cfg.PostStatus(context.Background(), &m.Toot{
			Status: toot,
		})
		return nil
	})
}

//	20件程トゥートを取得し整形し描画する。
func timelineMastodon(app *kingpin.Application, cfg *m.Client) {
	cmd := app.Command("tl", "TimeLine for mastodon")
	cmd.Action(func(c *kingpin.ParseContext) error {
		timeline, err := cfg.GetTimelinePublic(context.Background(), true, nil)
		if err != nil {
			return err
		}
		for i := len(timeline) - 1; i >= 0; i-- {
			displayStatus(timeline[i])
		}
		return nil
	})
}

func updatenameMastodon(app *kingpin.Application, cfg *m.Client) {

	cmd := app.Command("un", "Update Name")
	name := cmd.Arg("text", "username update").Strings()
	cmd.Action(func(c *kingpin.ParseContext) error {
		account, err := cfg.AccountUpdate(context.Background(), &m.Profile{})
		if err != nil {
			return err
		}

		log.Print("w: ", account.DisplayName)
		log.Print("d: ", name)
		log.Print("w: ", account)
		//user := strings.Join(*name, " ")
		//account.DisplayName = user

		return nil

	})
}

//	Stram対応箇所
func streamMastodon(app *kingpin.Application, cfg *m.Client) {
	cmd := app.Command("ltl", "Streaming Local TimeLine")
	cmd.Action(func(c *kingpin.ParseContext) error {
		wsc := cfg.NewWSClient()
		q, err := wsc.StreamingWSPublic(context.Background(), true)
		if err != nil {
			return err
		}
		green := color.New(color.FgHiGreen).SprintFunc()
		red := color.New(color.FgHiRed).SprintFunc()
		for e := range q {
			if t, ok := e.(*m.UpdateEvent); ok {
				s := t.Status.Content
				s = strings.Replace(s, "<p>", "", -1)
				s = strings.Replace(s, "</p>", "", -1)
				fmt.Printf("%s %s %s\n",
					red(t.Status.Account.Acct),
					green(t.Status.Account.DisplayName),
					s,
				)

			}
		}

		return nil
	})
}

func acct(a string) string {
	return a
}
func displayStatus(t *m.Status) {
	if t == nil {
		return
	}
	if t.Reblog != nil {
		color.Set(color.FgHiRed)
		fmt.Printf(acct(t.Account.Acct))
		color.Set(color.Reset)
		fmt.Printf(" reblogged ")
		color.Set(color.FgHiBlue)
		fmt.Println(acct(t.Reblog.Account.Acct))
		fmt.Println(textContent(t.Reblog.Content))
		color.Set(color.Reset)
	} else {
		color.Set(color.FgHiRed)
		fmt.Printf(acct(t.Account.Acct))
		color.Set(color.Reset)
		color.Set(color.FgHiGreen)
		fmt.Printf(acct(t.Account.DisplayName))
		color.Set(color.Reset)
		fmt.Println(textContent(t.Content))
	}
}

func textContent(s string) string {
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		return s
	}
	var buf bytes.Buffer

	var extractText func(node *html.Node, w *bytes.Buffer)
	extractText = func(node *html.Node, w *bytes.Buffer) {
		if node.Type == html.TextNode {
			data := strings.Trim(node.Data, "\r\n")
			if data != "" {
				w.WriteString(data)
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			extractText(c, w)
		}
		if node.Type == html.ElementNode {
			name := strings.ToLower(node.Data)
			if name == "br" {
				w.WriteString("\n")
			}
		}
	}
	extractText(doc, &buf)
	return buf.String()
}
