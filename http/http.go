package http

import (
	"fmt"
	logging "github.com/ipfs/go-log/v2"
	"github.com/zscboy/titan-workers-sdk/proxy"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

var log = logging.Logger("http")

const (
	defaultHTTPPort = 80
	httpScheme      = "http://"
	httpsScheme     = "https://"
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
		if r.Method == http.MethodConnect {
			handleHTTPSRequest(w, r, tm)
			return
		}
		handleHTTPRequest(w, r, tm)
	}
}

func handleHTTPRequest(w http.ResponseWriter, r *http.Request, tm *proxy.TunMgr) {
	info, err := parseRequestInfo(r)
	if err != nil {
		http.Error(w, "Invalid request URL", http.StatusBadRequest)
		return
	}

	log.Infof("accept http, srcAddr: %s, srcPort: %s, dstAddr: %s, dstPort: %d",
		info.srcAddr, info.srcPort, info.dstAddr, info.dstPort)

	strHead := buildRequestHeader(r)

	conn := newConn()
	defer conn.Close()

	tm.OnAcceptHTTPRequest(conn, &proxy.DestAddr{Addr: info.dstAddr, Port: info.dstPort}, []byte(strHead))

	_, err = io.Copy(w, conn)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func handleHTTPSRequest(w http.ResponseWriter, r *http.Request, tm *proxy.TunMgr) {
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

	tm.OnAcceptHTTPsRequest(conn, &proxy.DestAddr{Addr: info.dstAddr, Port: info.dstPort})
}

func parseRequestInfo(r *http.Request) (*struct {
	srcAddr, srcPort, dstAddr string
	dstPort                   int
}, error) {
	if !strings.HasPrefix(r.RequestURI, "http") {
		r.RequestURI = httpScheme + r.RequestURI
	}

	srvUrl, err := url.Parse(r.RequestURI)
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

func buildRequestHeader(r *http.Request) string {
	headerString := fmt.Sprintf("%s %s %s\r\n", r.Method, r.URL.String(), r.Proto)
	headerString += fmt.Sprintf("Host: %s\r\n", r.RemoteAddr)
	for name, values := range r.Header {
		for _, value := range values {
			headerString += fmt.Sprintf("%s: %s\r\n", name, value)
		}
	}
	headerString += "\r\n"
	return headerString
}
