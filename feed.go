// Package function generate a JSON feed from authenticated user notifications
// Check https://jsonfeed.org/version/1.1 for more details
// https://docs.github.com/en/rest/reference/activity#list-notifications-for-the-authenticated-user
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v32/github"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/text/language"
)

var githubClient *github.Client
var c *cache.Cache

// Check https://jsonfeed.org/version/1.1 for more details
type jsonFeed struct {
	Version     string       `json:"version"`
	Title       string       `json:"title"`
	HomePageURL string       `json:"home_page_url"`
	FeedURL     string       `json:"feed_url"`
	Description string       `json:"description,omitempty"`
	UserComment string       `json:"user_comment,omitempty"`
	NextURL     string       `json:"next_url,omitempty"`
	Icon        string       `json:"icon,omitempty"`
	Favicon     string       `json:"favicon,omitempty"`
	Authors     []author     `json:"authors,omitempty"`
	Language    language.Tag `json:"language,omitempty"`
	Expired     bool         `json:"expired,omitempty"`
	Items       []item       `json:"items"`
}

type author struct {
	Name   string `json:"name,omitempty"`
	URL    string `json:"url,omitempty"`
	Avatar string `json:"avatar,omitempty"`
}

type item struct {
	ID            string `json:"id"`
	URL           string `json:"url"`
	Title         string `json:"title"`
	ContentHTML   string `json:"content_html,omitempty"`
	ContentText   string `json:"content_text,omitempty"`
	DatePublished string `json:"date_published,omitempty"`
	DateModified  string `json:"date_modified,omitempty"`
}

func init() {
	// Set up cache for rate-limit request
	// Create a cache with a default expiration time of 5 minutes, and which
	// purges expired items every 10 minutes
	c = cache.New(5*time.Minute, 10*time.Minute)
}

// Execute a custom HTTP request with Github client
func customGithubRequest(ctx context.Context, url string) (string, *github.Response, error) {
	log.Debugf("URL being requested: %s", url)
	req, err := githubClient.NewRequest("GET", url, nil)
	if err != nil {
		return "", nil, err
	}

	var result map[string]interface{}
	resp, err := githubClient.Do(ctx, req, &result)
	if err != nil {
		return "", resp, err
	}
	defer resp.Body.Close()

	// html_url is always a string
	return result["html_url"].(string), nil, nil
}

func sendResponse(w http.ResponseWriter, feed *jsonFeed) {
	data, _ := json.Marshal(feed)
	w.Header().Set("Content-Type", "application/feed+json")
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%s", 5*time.Minute))
	_, _ = w.Write(data)
}

func getGithubNotificationsJSONFeed(w http.ResponseWriter, r *http.Request) {
	log.Infoln("Read Github user token from request")
	var githubToken = r.URL.Query().Get("token")
	if githubToken == "" {
		log.Warningln("Token not found, aborting")
		http.Error(w, "Forbidden", 403)
		return
	}

	log.Debugln("Check if not already in cache")
	var feed *jsonFeed
	if x, found := c.Get(githubToken); found {
		log.Infoln("Found in cache, send it!")
		feed = x.(*jsonFeed)

		// Send immediately
		sendResponse(w, feed)
		return
	}

	ctx := context.Background()
	// Create Github client only if not exist yet
	if githubClient == nil {
		log.Debugln("Create Github client")
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: githubToken},
		)
		tc := oauth2.NewClient(ctx, ts)

		// Store github client in memory for future calls
		githubClient = github.NewClient(tc)
	}

	log.Debugln("List unread notifications")
	notifications, resp, err := githubClient.Activity.ListNotifications(ctx, &github.NotificationListOptions{
		// Get all notifications
		All: true,

		// Only 20 is sufficient
		ListOptions: github.ListOptions{
			PerPage: 20,
		},
	})
	if err != nil {
		log.Errorf("Error on notifications list: %s", err)
		http.Error(w, resp.Status, resp.StatusCode)
		return
	}

	// For each notifications, create a feed item
	log.Infof("Found %d notifications", len(notifications))
	var items = make([]item, len(notifications))
	var wg sync.WaitGroup
	wg.Add(len(notifications))

	for i, notification := range notifications {
		go func(j int, n *github.Notification) {
			defer wg.Done()

			// Create default title
			t := fmt.Sprintf("[%s] %s - %s", n.GetSubject().GetType(), n.Repository.GetFullName(), n.Subject.GetTitle())

			// Default Notification HTML URL to repo URL for private repo
			// https://docs.github.com/en/free-pro-team@latest/rest/reference/pulls#get-a-pull-request
			var htmlURL = strings.Replace(n.Subject.GetURL(), "https://api.github.com/repos", "https://github.com", 1)
			htmlURL = strings.Replace(htmlURL, "pulls", "pull", 1)

			// Create the default response object
			items[j] = item{
				ID:            n.GetID(),
				URL:           htmlURL,
				Title:         t,
				ContentText:   t,
				DatePublished: n.GetUpdatedAt().Format(time.RFC3339),
			}

			// If invalid notifications, use the repo URL instead and continue
			if n.Subject.GetURL() == "" {
				items[j].URL = n.Repository.GetHTMLURL()
				log.Warningf("[%s] %q: missing URL", n.Repository.GetFullName(), n.Subject.GetTitle())
				return
			}

			// Try to get the real URL in case of public repo
			u, _, err := customGithubRequest(ctx, n.Subject.GetURL())
			if err == nil {
				// if not error, replace the URL with the real subject and continue
				items[j].URL = u
				return
			}

			// Else, use the GitHub error response
			var m = []string{err.Error()}
			if v, ok := err.(*github.ErrorResponse); ok {
				for _, e := range v.Errors {
					m = append(m, e.Message)
				}
			}

			log.Errorf("Error on notifications list: %s", strings.Join(m, ", "))
		}(i, notification)
	}

	wg.Wait()

	// Sort items by date
	sort.Slice(items, func(i, j int) bool {
		return items[i].DatePublished > items[j].DatePublished
	})

	feed = &jsonFeed{
		Version:     "https://jsonfeed.org/version/1.1",
		Title:       "Github Notifications",
		HomePageURL: "https://github.com/notifications",
		FeedURL:     feedURL,
		Description: "Your Github notifications",
		Icon:        "https://www.iconfinder.com/data/icons/octicons/1024/mark-github-512.png",
		Favicon:     "https://github.com/favicon.ico",
		Authors: []author{
			{
				Name:   "Quentin Lemaire",
				Avatar: "https://gravatar.com/avatar/ae3ee0665731b1010ed57bd608ac213b?s=400&d=robohash&r=x",
				URL:    "https://lemairepro.fr",
			},
			{
				Name:   "Github",
				Avatar: "https://www.iconfinder.com/data/icons/octicons/1024/mark-github-512.png",
				URL:    "https://github.com",
			},
		},
		Language: language.AmericanEnglish,
		Expired:  false,
		Items:    items,
	}

	if useCache {
		log.Infoln("Store in cache")
		c.Set(githubToken, feed, cache.DefaultExpiration)
	}

	// Send final response
	sendResponse(w, feed)
}
