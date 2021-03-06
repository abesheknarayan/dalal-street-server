package utils

import (
	"errors"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	sendgrid "github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// RandString generates a random string which is n characters long
func RandString(n int) string {
	b := make([]byte, n)
	// A rand.Int63() generates 63 random bits, enough for letterIdxMax letters!
	for i, cache, remain := n-1, rand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

func MinInt32(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

func MinInt64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

func MinTripleInt64(a, b, c int64) int64 {
	if a < b {
		if c < a {
			return c
		} else {
			return a
		}
	} else {
		if c < b {
			return c
		} else {
			return b
		}
	}
}

func AbsInt64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

func IsGrpcRequest(req *http.Request) bool {
	return strings.Contains(req.Header.Get("Content-Type"), "application/grpc")
}

func GetCurrentTimeISO8601() string {
	return time.Now().Format(time.RFC3339)
}

func GetImageBasePath() string {
	return os.Getenv("GOPATH") + "/src/github.com/delta/dalal-street-server/frontend/public/src/images/news/"
}

func IsProdEnv() bool {
	return strings.Contains(strings.ToLower(config.Stage), "prod")
}

func SendEmail(fromAddr, subject, toAddr, plainTextContent, htmlContent string) error {
	from := mail.NewEmail("DalalStreet", fromAddr)
	to := mail.NewEmail("Example User", toAddr)
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	response, err := client.Send(message)
	if err != nil {
		return err
	} else if response.StatusCode >= 300 {
		return errors.New(response.Body)
	}
	return nil
}
