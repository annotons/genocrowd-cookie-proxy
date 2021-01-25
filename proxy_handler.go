package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/patrickmn/go-cache"
	"golang.org/x/crypto/blowfish"
)

type ProxyHandler struct {
	Transport *http.Transport
	// Backend
	BackendAddress string
	BackendScheme  string
	GalaxyCipher   *blowfish.Cipher
	Cache          *cache.Cache
	EmailCache     *cache.Cache
	Header         string
	// Frontend
	AddForwarded bool
}

func Copy(dest *bufio.ReadWriter, src *bufio.ReadWriter) {
	buf := make([]byte, 40*1024)
	for {
		n, err := src.Read(buf)
		if err != nil && err != io.EOF {
			log.Printf("Read failed: %v", err)
			return
		}
		if n == 0 {
			return
		}
		dest.Write(buf[0:n])
		dest.Flush()
	}
}

func CopyBidir(conn1 io.ReadWriteCloser, rw1 *bufio.ReadWriter, conn2 io.ReadWriteCloser, rw2 *bufio.ReadWriter) {
	finished := make(chan bool)

	go func() {
		Copy(rw2, rw1)
		conn2.Close()
		finished <- true
	}()
	go func() {
		Copy(rw1, rw2)
		conn1.Close()
		finished <- true
	}()

	<-finished
	<-finished
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//log.Printf("incoming request: %#v", *r)
	r.RequestURI = ""
	r.URL.Scheme = h.BackendScheme

	if h.AddForwarded {
		remoteAddr := r.RemoteAddr
		idx := strings.LastIndex(remoteAddr, ":")
		if idx != -1 {
			remoteAddr = remoteAddr[0:idx]
			if remoteAddr[0] == '[' && remoteAddr[len(remoteAddr)-1] == ']' {
				remoteAddr = remoteAddr[1 : len(remoteAddr)-1]
			}
		}
		r.Header.Add("X-Forwarded-For", remoteAddr)
	}

	r.URL.Host = h.BackendAddress

	connHdr := ""
	connHdrs := r.Header["Connection"]

	if len(connHdrs) > 0 {
		log.WithFields(log.Fields{
			"headers": connHdrs,
		}).Debug("Connection headers")
		connHdr = connHdrs[0]
	}

	var email = ""
	gxCookie, err := r.Cookie("session")
	if err == nil {
		email, _ = timedLookupEmailByCookie(h, gxCookie.String())
		if email != "" {
			r.Header[h.Header] = []string{email}

			log.WithFields(log.Fields{
				"user": email,
				"path": r.URL.Path,
			}).Info("Authenticated request")

			metricIncr("requests.authenticated")
		} else {
			metricIncr("requests.unauthenticated")
		}
	} else {
		metricIncr("requests.nocookie")
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error: you don't have a genocrowd cookie. Please login to Genocrowd first.")
		return
	}

	upgradeWebsocket := false
	if strings.ToLower(connHdr) == "upgrade" {
		log.Debug("got Connection: Upgrade")

		upgradeHdrs := r.Header["Upgrade"]
		//log.Printf("Upgrade headers: %v", upgradeHdrs)
		if len(upgradeHdrs) > 0 {
			upgradeWebsocket = (strings.ToLower(upgradeHdrs[0]) == "websocket")
		}
	}

	if upgradeWebsocket {
		hj, ok := w.(http.Hijacker)

		if !ok {
			http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
			return
		}

		conn, bufrw, err := hj.Hijack()
		defer conn.Close()

		conn2, err := net.Dial("tcp", r.URL.Host)
		if err != nil {
			http.Error(w, "couldn't connect to backend server", http.StatusServiceUnavailable)
			return
		}
		defer conn2.Close()

		err = r.Write(conn2)
		if err != nil {
			log.Printf("writing WebSocket request to backend server failed: %v", err)
			return
		}

		CopyBidir(conn, bufrw, conn2, bufio.NewReadWriter(bufio.NewReader(conn2), bufio.NewWriter(conn2)))

	} else {

		resp, err := h.Transport.RoundTrip(r)
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, "Error: %v", err)
			return
		}

		for k, v := range resp.Header {
			for _, vv := range v {
				w.Header().Add(k, vv)
			}
		}

		w.WriteHeader(resp.StatusCode)

		io.Copy(w, resp.Body)
		resp.Body.Close()
	}
}
