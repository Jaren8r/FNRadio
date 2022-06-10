package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"syscall"

	"github.com/elazarl/goproxy"
)

type FNRadioClient struct {
	Proxy       *goproxy.ProxyHttpServer
	Certificate *tls.Certificate
	APIClient   APIClient
	Users       map[string]*APIUser
	Party       Party
	BoundUser   string
	LogFile     io.Writer
	Logger      *log.Logger

	alreadyProxying      bool
	previousProxyEnabled uint64
	previousProxyServer  string
}

var client *FNRadioClient

func setupCloseHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		client.Destroy()
		os.Exit(0)
	}()
}

func (client *FNRadioClient) handleAkamaizedConnect(host string, _ *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
	return &goproxy.ConnectAction{
		Action:    goproxy.ConnectMitm,
		TLSConfig: goproxy.TLSConfigFromCA(client.Certificate),
	}, host
}

func (client *FNRadioClient) handleAkamaizedRequest(r *http.Request, _ *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	regex := regexp.MustCompile(`^/([^/,\s]+)/master\.blurl$`)

	matchString := regex.FindStringSubmatch(r.URL.Path)

	if len(matchString) > 0 {
		for _, binding := range client.Users[client.BoundUser].Bindings {
			if binding.ID == matchString[1] {
				_ = client.Logger.Output(2, "Rewriting request "+r.URL.String()+" to station "+binding.StationUser+":"+binding.StationID)

				r.URL, _ = url.Parse(APIRoot + "/users/" + binding.StationUser + "/stations/" + binding.StationID)

				r.Header.Set("Authorization", client.APIClient.generateAuthHeader())

				r.Header.Set("X-API-Root", APIRoot)

				r.Host = r.URL.Host

				break
			}
		}
	}

	return r, nil
}

func (client *FNRadioClient) Destroy() {
	client.revertSystemProxy()
}

func (client *FNRadioClient) FetchSelf() {
	_ = client.Logger.Output(2, "Fetching self")

	user, err := client.APIClient.GetUser("@me")
	if err != nil {
		client.Destroy()

		panic(err)
	}

	client.BoundUser = client.APIClient.ID
	client.Users[client.APIClient.ID] = user
}

func main() {
	fmt.Println("FNRadio by Jaren (@The1Jaren) [" + Version + "]")
	fmt.Println("")
	fmt.Println("Join our discord: https://discord.gg/bgRM3XdhnA")
	fmt.Println("SAC Code: Jaren")

	client = &FNRadioClient{
		Proxy:       goproxy.NewProxyHttpServer(),
		Certificate: setupSSL(),
		APIClient:   APIClient{},
		Users:       map[string]*APIUser{},
		LogFile:     ioutil.Discard,
	}

	logFile, err := os.OpenFile("FNRadio.log", os.O_CREATE|os.O_TRUNC, 0666)
	if err == nil {
		client.LogFile = logFile
	}

	client.Logger = log.New(client.LogFile, "[FNRadio] ", log.LstdFlags)

	client.APIClient.Setup()

	client.FetchSelf()

	client.Proxy.Verbose = true

	client.Proxy.Logger = log.New(client.LogFile, "[GoProxy] ", log.LstdFlags)

	client.Proxy.OnRequest(goproxy.ReqHostIs("fortnite-vod.akamaized.net:443")).HandleConnect(goproxy.FuncHttpsHandler(client.handleAkamaizedConnect))

	client.Proxy.OnRequest(goproxy.ReqHostIs("fortnite-vod.akamaized.net:443")).Do(goproxy.FuncReqHandler(client.handleAkamaizedRequest))

	client.Proxy.OnRequest(goproxy.ReqHostIs("cdn-0001.qstv.on.epicgames.com:443")).HandleConnect(goproxy.AlwaysReject)

	client.setupSystemProxy()

	setupCloseHandler()

	go setupCLI()

	go client.readGameLog()

	err = http.ListenAndServe("127.0.0.1:18149", client.Proxy)
	if err != nil {
		fmt.Println(err)
		client.Destroy()
	}
}
