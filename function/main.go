// Package function generate a JSON feed from authenticated user notifications
// Check https://jsonfeed.org/version/1.1 for more details
// https://docs.github.com/en/rest/reference/activity#list-notifications-for-the-authenticated-user
package function

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v32/github"
	"github.com/patrickmn/go-cache"
	"golang.org/x/oauth2"
	"golang.org/x/text/language"

	joonix "github.com/joonix/log"
	log "github.com/sirupsen/logrus"
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
	// Set up logger for Stackdriver
	log.SetLevel(log.DebugLevel)
	if os.Getenv("GCP_PROJECT") != "" || os.Getenv("FUNCTION_NAME") != "" {
		log.SetFormatter(joonix.NewFormatter())
	}

	// Set up cache for rate-limit request
	// Create a cache with a default expiration time of 5 minutes, and which
	// purges expired items every 10 minutes
	c = cache.New(5*time.Minute, 10*time.Minute)
}

// Execute a custom HTTP request with Github client
func customGithubRequest(ctx context.Context, url string) (string, *github.Response, error) {
	log.Debugf("URL being requested: %s\n", url)
	req, err := githubClient.NewRequest("GET", url, nil)
	if err != nil {
		return "", nil, err
	}

	var result map[string]interface{}
	resp, err := githubClient.Do(ctx, req, &result)
	if err != nil {
		return "", resp, err
	}

	// html_url is always a string
	return result["html_url"].(string), nil, nil
}

func getFeedURLFromGoogleRuntime() string {
	return fmt.Sprintf("https://%s-%s.cloudfunctions.net/%s", os.Getenv("FUNCTION_REGION"), os.Getenv("GCP_PROJECT"), os.Getenv("FUNCTION_NAME"))
}

func sendResponse(w http.ResponseWriter, feed *jsonFeed) {
	data, _ := json.Marshal(feed)
	w.Header().Set("Content-Type", "application/feed+json")
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%s", 5*time.Minute))
	w.Write(data)
}

// GetGithubNotificationsJSONFeed Google CLoud Function entrypoint
func GetGithubNotificationsJSONFeed(w http.ResponseWriter, r *http.Request) {
	log.Infoln("Read Github user token from request")
	var githubToken string = r.URL.Query().Get("token")
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
	log.Infof("Found %d notifications\n", len(notifications))
	var items []item = make([]item, 0)
	var wg sync.WaitGroup

	for _, notification := range notifications {
		wg.Add(1)
		go func(n *github.Notification) {
			defer wg.Done()

			// Create default title
			t := fmt.Sprintf("[%s] %s - %s", n.GetSubject().GetType(), n.Repository.GetFullName(), n.Subject.GetTitle())

			// Default Notification HTML URL to repo URL for private repo
			// https://docs.github.com/en/free-pro-team@latest/rest/reference/pulls#get-a-pull-request
			var htmlURL string = strings.Replace(n.Subject.GetURL(), "https://api.github.com/repos", "https://github.com", 1)
			htmlURL = strings.Replace(htmlURL, "pulls", "pull", 1)

			// Try to get  the real URL in case of public repo
			u, _, err := customGithubRequest(ctx, n.Subject.GetURL())
			if err == nil {
				// If not error, use this URL instead of the default one
				htmlURL = u
			} else {
				// Use Github Error if provided
				var m []string = []string{err.Error()}
				if v, ok := err.(*github.ErrorResponse); ok {
					for _, e := range v.Errors {
						m = append(m, e.Message)
					}
				}
				log.Errorf("Error on notifications list: %s", strings.Join(m, ", "))
			}

			items = append(items, item{
				ID:            n.GetID(),
				URL:           htmlURL,
				Title:         t,
				ContentText:   t,
				DatePublished: n.GetUpdatedAt().Format(time.RFC3339),
			})
		}(notification)
	}

	log.Debugln("Waiting for process")
	wg.Wait()

	// Sort items by date
	sort.Slice(items, func(i, j int) bool {
		return items[i].DatePublished > items[j].DatePublished
	})

	feed = &jsonFeed{
		Version:     "https://jsonfeed.org/version/1.1",
		Title:       "Github Notifications",
		HomePageURL: "https://github.com/notifications",
		FeedURL:     getFeedURLFromGoogleRuntime(),
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

	log.Infoln("Store in cache")
	c.Set(githubToken, feed, cache.DefaultExpiration)

	// Send final response
	sendResponse(w, feed)
}
