package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/color/palette"
	"math/rand"
	"sync"
	"time"
)

import "github.com/go-macaron/captcha"

var (
	// Protects the map of captcha answers.
	captchasM sync.Mutex

	// Key is the captcha answer, like "208823751".
	// Val is unix time of captcha generation.
	captchas = map[string]int64{}
)

func CaptchaGenerateBase64ImageDataURL(width, height, maxCaptchas int) (
	string, error) {
	png, err := CaptchaGenerate(width, height, maxCaptchas)
	if err != nil {
		return "", nil
	}

	s := base64.StdEncoding.EncodeToString(png)

	return "data:image/png;base64," + s, nil
}

func CaptchaGenerate(width, height, maxCaptchas int) (
	[]byte, error) {
	answerI := rand.Int() % 1000000
	for answerI < 100000 {
		answerI += 100000
	}

	answer := fmt.Sprintf("%d", answerI)

	digits := []byte(answer)
	for i, digit := range digits {
		digits[i] = digit - byte('0')
	}

	var png bytes.Buffer

	n, err := captcha.NewImage(digits, width, height,
		palette.WebSafe[1:30]).WriteTo(&png)
	if err != nil || n <= 0 {
		return nil, fmt.Errorf("captcha.NewImage, n: %d, err: %v", n, err)
	}

	captchasM.Lock()
	defer captchasM.Unlock()

	captchas[answer] = time.Now().Unix()

	// Delete oldest answers when the captchas map gets too large.
	for len(captchas) > maxCaptchas {
		var oldest string
		var oldestTime int64

		for answer, answerTime := range captchas {
			if oldestTime <= 0 || answerTime < oldestTime {
				oldest = answer
				oldestTime = answerTime
			}
		}

		if oldest == "" || oldestTime <= 0 {
			break
		}

		delete(captchas, oldest)
	}

	return png.Bytes(), nil
}

func CaptchaCheck(guess string) bool {
	captchasM.Lock()
	defer captchasM.Unlock()

	_, ok := captchas[guess]
	if ok {
		delete(captchas, guess)
	}

	return ok
}