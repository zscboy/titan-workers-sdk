package http

import (
	"fmt"
	logging "github.com/ipfs/go-log/v2"
	"github.com/zscboy/titan-workers-sdk/proxy"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

var log = logging.Logger("http")

const (
	defaultHTTPPort    = 80
	defaultHTTPsScheme = "https://"
)

type ProxyServer struct {
	server *http.Server
	tm     *proxy.TunMgr
}

func NewProxyServer(address string, tm *proxy.TunMgr) *ProxyServer {
	s := &http.Server{
		Addr:    address,
		Handler: proxyHandler(tm),
	}

	return &ProxyServer{server: s, tm: tm}
}

func (p *ProxyServer) Start() {
	log.Infof("HTTP Server listening on %s\n", p.server.Addr)
	if err := p.server.ListenAndServe(); err != nil {
		log.Errorf("HTTP Server failed to start: %v", err)
	}
}

func proxyHandler(tm *proxy.TunMgr) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect {
			http.NotFound(w, r)
			return
		}
		handleRequest(w, r, tm)
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request, tm *proxy.TunMgr) {
	info, err := parseRequestInfo(r)
	if err != nil {
		http.Error(w, "Invalid request URL", http.StatusBadRequest)
		return
	}

	log.Infof("accept https, srcAddr: %s, srcPort: %s, dstAddr: %s, dstPort: %d",
		info.srcAddr, info.srcPort, info.dstAddr, info.dstPort)

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}

	conn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	tm.OnAcceptHTTPsRequest(conn, &proxy.DestAddr{Addr: info.dstAddr, Port: info.dstPort}, nil)

	strHead := buildRequestHeader(r, info.dstAddr)

	tm.OnAcceptHTTPRequest(conn, &proxy.DestAddr{Addr: info.dstAddr, Port: info.dstPort}, []byte(strHead))
}

func parseRequestInfo(r *http.Request) (*struct {
	srcAddr, srcPort, dstAddr string
	dstPort                   int
}, error) {
	srvUrl, err := url.Parse(defaultHTTPsScheme + r.RequestURI)
	if err != nil {
		return nil, err
	}
	dstPort := defaultHTTPPort
	if port := srvUrl.Port(); port != "" {
		dstPort, _ = strconv.Atoi(port)
	}
	return &struct {
		srcAddr, srcPort, dstAddr string
		dstPort                   int
	}{
		srcAddr: r.RemoteAddr,
		srcPort: strings.Split(r.RemoteAddr, ":")[1],
		dstAddr: srvUrl.Hostname(),
		dstPort: dstPort,
	}, nil
}

func buildRequestHeader(r *http.Request, dstAddr string) string {
	strHead := fmt.Sprintf("%s %s HTTP/%s\r\n", r.Method, dstAddr, r.Proto)
	for k, v := range r.Header {
		strHead += fmt.Sprintf("%s: %s\r\n", k, strings.Join(v, ", "))
	}
	strHead += "\r\n"
	return strHead
}
