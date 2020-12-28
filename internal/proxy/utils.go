package proxy

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ContentRequest represents HTTP request that is received from the user
type ContentRequest struct {
	Title   string
	Suffix  string
	Channel *Channel
}

func getContentRequest(w http.ResponseWriter, r *http.Request, prefix string) (*ContentRequest, error) {
	reqPath := strings.Replace(r.URL.RequestURI(), prefix, "", 1)
	reqPathParts := strings.SplitN(reqPath, "/", 2)
	if len(reqPathParts) == 0 {
		return nil, errors.New("bad request")
	}

	// Unescape channel title
	var err error
	reqPathParts[0], err = url.PathUnescape(reqPathParts[0])
	if err != nil {
		return nil, err
	}

	// Find channel reference
	channel, ok := playlist[reqPathParts[0]]
	if !ok {
		return nil, errors.New("bad request")
	}
	log.Println(channel.LinkURL)

	if len(reqPathParts) == 1 {
		return &ContentRequest{reqPathParts[0], "", channel}, nil
	}
	return &ContentRequest{reqPathParts[0], reqPathParts[1], channel}, nil
}

func (cr *ContentRequest) validSession() bool {
	if time.Since(cr.Channel.sessionUpdated).Seconds() > 30 || cr.Channel.sessionUpdated.IsZero() {
		return false
	}
	return true
}

func (cr *ContentRequest) updateChannel() error {
	newLink, err := cr.Channel.StalkerChannel.NewLink()
	if err != nil {
		return err
	}
	log.Println("New Link")
	log.Println(newLink)

	cr.Channel.LinkURL = newLink
	cr.Channel.LinkType = 0
	if cr.Channel.LinkM3u8Ref != nil {
		cr.Channel.LinkM3u8Ref.link = ""
		cr.Channel.LinkM3u8Ref.linkRoot = ""
	}
	cr.Channel.sessionUpdated = time.Now()

	return nil
}

func downloadString(link string) (content string, contentType string, err error) {
	contentBytes, contentType, err := download(link)
	if err != nil {
		return "", "", err
	}
	return string(contentBytes), contentType, nil
}

func download(link string) (content []byte, contentType string, err error) {
	resp, err := response(link)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	content, err = ioutil.ReadAll(resp.Body)
	return content, resp.Header.Get("Content-Type"), err
}

// HTTP client that does not follow redirects
// It automatically adds "Referrerr" header which causes
// 404 errors on some backends.
var httpClient = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

func response(link string) (*http.Response, error) {
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "VLC/3.0.9 LibVLC/3.0.9")

	u, err := url.Parse(link)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Host", u.Host)
	// req.Header.Set("Accept", "*/*")
	// req.Header.Set("Accept-Language", "en_US")
	// req.Header.Set("Range", "bytes=0-")

	resp, err := httpClient.Do(req)
	log.Println(req)
	log.Println(resp)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp, nil
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		linkURL, err := url.Parse(link)
		if err != nil {
			return nil, errors.New("Unknown error occurred")
		}
		redirectURL, err := url.Parse(resp.Header.Get("Location"))
		if err != nil {
			return nil, errors.New("Unknown error occurred")
		}
		newLink := linkURL.ResolveReference(redirectURL)
		return response(newLink.String())
	}

	return nil, errors.New(link + " returned HTTP code " + strconv.Itoa(resp.StatusCode))
}

func addHeaders(from, to http.Header, contentLength bool) {
	for k, v := range from {
		switch k {
		case "Connection":
			to.Set("Connection", strings.Join(v, "; "))
		case "Content-Type":
			to.Set("Content-Type", strings.Join(v, "; "))
		case "Transfer-Encoding":
			to.Set("Transfer-Encoding", strings.Join(v, "; "))
		case "Cache-Control":
			to.Set("Cache-Control", strings.Join(v, "; "))
		case "Date":
			to.Set("Date", strings.Join(v, "; "))
		case "Content-Length":
			if contentLength {
				to.Set("Content-Length", strings.Join(v, "; "))
			}
		}
	}
}
