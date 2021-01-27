package main

import (
	"bytes"
	"compress/zlib"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io/ioutil"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	_ "github.com/lib/pq"
	cache "github.com/patrickmn/go-cache"
)

func timedLookupEmailByCookie(b *ProxyHandler, cookie string) (string, bool) {
	start := time.Now()
	email, found := lookupEmailByCookie(b, cookie)
	t := time.Now()
	elapsed := t.Sub(start)
	metricTime("query_timing", elapsed)

	return email, found
}

func lookupEmailByCookie(b *ProxyHandler, cookie string) (email string, found bool) {
	// 8 is the size of "session="
	cachedEmail, found := b.Cache.Get(cookie[8:])
	log.WithFields(log.Fields{
		"hit": found,
	}).Debug("Cache hit")
	if found {
		metricIncr("cache.hit")
		return cachedEmail.(string), found
	}
	metricIncr("cache.miss")

	email, success := decryptCookie(cookie[8:], b.GenocrowdSecret, b.MaxAge)

	if success {
		b.Cache.Set(cookie[8:], email, cache.DefaultExpiration)
	}

	return email, success
}

func decryptCookie(token string, secret string, maxAge int) (email string, found bool) {
	// Split cookie content
	// cookie content is ".<base64+gzipped payload>.<vase64 timestamp>.<signature>"
	tokenElmts := strings.Split(token[1:], ".")
	if len(tokenElmts) != 3 {
		log.Error("Unexpected cookie format")
		return "", false
	}

	// Check timestamp
	timestamp, err := base64.RawURLEncoding.DecodeString(tokenElmts[1])
	if err != nil {
		log.Error("Failed to decode cookie timestamp")
		return "", false
	}
	timestampInt := binary.BigEndian.Uint32(timestamp)
	log.WithFields(log.Fields{
		"timestamp": timestampInt,
	}).Debug("Cookie timestamp")

	now := time.Now()
	timestampNow := now.Unix()
	if timestampNow-int64(timestampInt) > int64(maxAge) {
		log.Error("Cookie too old")
		return "", false
	}

	// Check cookie signature
	salt := "cookie-session" // This is Flask's default
	hashedSecret := ComputeHmacSha1(salt, []byte(secret))

	/*log.Printf("secret to hash: %#v", secret)
	log.Printf("salt to hash: %#v", salt)
	log.Printf("=> hashed secret: %#v", hashedSecret)*/

	if !verifySignature(token, hashedSecret) {
		log.Error("Invalid cookie signature")
		return "", false
	}

	// Inspect the payload
	payloadEncoded, err := base64.RawURLEncoding.DecodeString(tokenElmts[0])
	if err != nil {
		log.Error("Failed to decode cookie payload")
		return "", false
	}

	payloadStr, err := readSegment(payloadEncoded)
	if err != nil {
		log.Error("Failed to decompress cookie payload")
		return "", false
	}

	/*log.WithFields(log.Fields{
		"payload": string(payloadStr),
	}).Debug("Received payload")*/

	var payloadJSON map[string]interface{}
	errj := json.Unmarshal([]byte(payloadStr), &payloadJSON)
	if errj != nil {
		log.Error("Failed to decode cookie payload as json")
		return "", false
	}

	if payloadJSON["user"] != nil {
		user := payloadJSON["user"].(map[string]interface{})
		if user["email"] != nil {
			log.WithFields(log.Fields{
				"email": user["email"].(string),
			}).Debug("Found user email in cookie payload")

			return user["email"].(string), true
		}
	}

	log.Error("Failed to find email in cookie payload")
	return "", false
}

func ComputeHmacSha1(message string, key []byte) string {
	h := hmac.New(sha1.New, key)
	h.Write([]byte(message))
	a := base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	k := strings.Trim(a, "=")
	return k
}

func readSegment(data []byte) ([]byte, error) {
	b := bytes.NewReader(data)
	z, err := zlib.NewReader(b)
	if err != nil {
		return nil, err
	}
	defer z.Close()
	p, err := ioutil.ReadAll(z)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func verifySignature(token string, secret string) bool {
	tokenElmts := strings.Split(token, ".")
	add12 := "." + tokenElmts[1] + "." + tokenElmts[2]

	/*log.Printf("hashed value: %#v", add12)
	log.Printf("secret: %#v", secret)*/

	decodedSecret, err := base64.RawURLEncoding.DecodeString(secret)
	if err != nil {
		log.Error("Failed to decode cookie signature")
		return false
	}
	k := ComputeHmacSha1(add12, decodedSecret)

	/*log.Printf("computed signature: %#v", k)
	log.Printf("expected signature: %#v", tokenElmts[3])*/

	return k == tokenElmts[3]
}
