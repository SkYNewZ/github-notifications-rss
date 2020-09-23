# github-notifications-rss

Go HTTP handler to recevice your Github notifications as [JSONFeed](https://jsonfeed.org/version/1.1).
It can be use to view your Github notfications feed is your favorite RSS reader !

Designed to be deploy on a [Google Cloud Function](https://cloud.google.com/functions/docs/writing/http)

## Usage

1. Create a Github Personnal token [here](https://github.com/settings/tokens)
2. If you want your private organization notifcations is this feed, clic on "Enable SSO"
3. Run this project wherever you want
4. Get your feed at `http://URL/URI?token=GITHUB_TOKEN`

## Example output:

```json
{
  "version": "https://jsonfeed.org/version/1.1",
  "title": "Github Notifications",
  "home_page_url": "https://github.com/notifications",
  "feed_url": "https://europe-west1-skynewz-dev-dt3s2a.cloudfunctions.net/github_notifications_feed",
  "description": "Your Github notifications",
  "icon": "https://www.iconfinder.com/data/icons/octicons/1024/mark-github-512.png",
  "favicon": "https://github.com/favicon.ico",
  "authors": [
    {
      "name": "Quentin Lemaire",
      "url": "https://lemairepro.fr",
      "avatar": "https://gravatar.com/avatar/ae3ee0665731b1010ed57bd608ac213b?s=400&d=robohash&r=x"
    },
    {
      "name": "Github",
      "url": "https://github.com",
      "avatar": "https://www.iconfinder.com/data/icons/octicons/1024/mark-github-512.png"
    }
  ],
  "language": "en-US",
  "items": [
    {
      "id": "1173005664",
      "url": "https://github.com/hashicorp/terraform/releases/tag/v0.13.3",
      "title": "[Release] hashicorp/terraform - v0.13.3",
      "content_text": "[Release] hashicorp/terraform - v0.13.3",
      "date_published": "2020-09-16T19:56:44Z"
    },
    {
      "id": "1188927339",
      "url": "https://github.com/traefik/traefik/releases/tag/v2.3.0",
      "title": "[Release] traefik/traefik - v2.3.0",
      "content_text": "[Release] traefik/traefik - v2.3.0",
      "date_published": "2020-09-23T11:41:01Z"
    },
    {
      "id": "1172627840",
      "url": "https://github.com/cli/cli/releases/tag/v1.0.0",
      "title": "[Release] cli/cli - v1.0.0",
      "content_text": "[Release] cli/cli - v1.0.0",
      "date_published": "2020-09-16T17:19:41Z"
    },
    {
      "id": "1180918216",
      "url": "https://github.com/restic/restic/releases/tag/v0.10.0",
      "title": "[Release] restic/restic - restic 0.10.0",
      "content_text": "[Release] restic/restic - restic 0.10.0",
      "date_published": "2020-09-19T16:26:43Z"
    },
    {
      "id": "817939610",
      "url": "https://github.com/restic/restic/issues/2688",
      "title": "[Issue] restic/restic - List snapshots with multiple filters tag and host filter not working ?",
      "content_text": "[Issue] restic/restic - List snapshots with multiple filters tag and host filter not working ?",
      "date_published": "2020-09-20T18:25:15Z"
    }
  ]
}
```

## Run in local

Simply build/run the `/main.go` file.

## Run in a Google Cloud Function

```sh
#!/bin/bash

set -e
(cd function && gcloud functions deploy github_notifications_feed \
--runtime=go111 \
--trigger-http \
--allow-unauthenticated \
--project=PROJECT \
--region=europe-west1 \
--entry-point=GetGithubNotificationsJSONFeed)
```
