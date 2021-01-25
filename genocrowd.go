package main

import (
	"encoding/hex"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	_ "github.com/lib/pq"
	cache "github.com/patrickmn/go-cache"
	"golang.org/x/crypto/blowfish"
)

func timedLookupEmailByCookie(b *ProxyHandler, cookie string) (string, bool) {
	start := time.Now()
	email, found := lookupEmailByCookie(b, cookie)
	t := time.Now()
	elapsed := t.Sub(start)
	metricTime("query_timing", elapsed)

	return email, found
}

func cookieToSessionKey(b *ProxyHandler, cookie string) (sessionKey string) {
	data, err := hex.DecodeString(cookie[14:])
	// If we can decode, exit early.
	if err != nil {
		return "will-never-match"
	}

	// Decrypt the session key
	pt := make([]byte, 40)
	for i := 0; i < len(data); i += blowfish.BlockSize {
		j := i + blowfish.BlockSize
		b.GalaxyCipher.Decrypt(pt[i:j], data[i:j])
	}

	// And strip all the exclamations from it.
	sessionKey := strings.Replace(string(pt), "!", "", -1)
	safeSessionKey := hexReg.ReplaceAllString(sessionKey, "")

	// Debugging
	log.WithFields(log.Fields{
		"sk": safeSessionKey,
	}).Debug("Session Key Decoded")
	return safeSessionKey
}

func lookupEmailByCookie(b *ProxyHandler, cookie string) (email string, found bool) {
	cachedEmail, found := b.Cache.Get(cookie[14:])
	log.WithFields(log.Fields{
		"hit": found,
	}).Debug("Cache hit")
	if found {
		metricIncr("cache.hit")
		return cachedEmail.(string), found
	}
	metricIncr("cache.miss")

	safeSessionKey := cookieToSessionKey(b, cookie)

	b.Cache.Set(cookie[14:], safeSessionKey, cache.DefaultExpiration)
	return email, false
}
