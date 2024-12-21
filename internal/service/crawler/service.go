package crawler

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"os"
	"path"
	"smp-youtube/internal/client/dropbox"
	"smp-youtube/internal/model"
	"smp-youtube/internal/repository"
	"smp-youtube/internal/service"
	"smp-youtube/internal/utils"
	"strings"
	"sync"
	"time"
)

const (
	everyRedirectSleep     = 3 * time.Second
	maxConcurrentDownloads = 5
)

var (
	selectorByType = map[string]string{
		"videos": "//a[@id='video-title-link']",
		"shorts": "//a[contains(@href,'/shorts') and @title]",
	}
)

type Service struct {
	cookiesPath       string
	outputPath        string
	dropboxClient     *dropbox.Client
	crawlerRepository repository.Crawler
}

func NewService(
	cookiesPath string,
	outputPath string,
	dropboxClient *dropbox.Client,
	crawlerRepo repository.Crawler,
) service.Crawler {
	return &Service{
		cookiesPath:       cookiesPath,
		outputPath:        outputPath,
		dropboxClient:     dropboxClient,
		crawlerRepository: crawlerRepo,
	}
}

func (srv *Service) Crawl(ctx context.Context, resources []model.Resource) error {
	for _, resource := range resources {
		if err := srv.get(ctx, fmt.Sprintf("%s/%s", resource.URL, resource.Type)); err != nil {
			return err
		}

		channel := strings.Replace(resource.URL, "https://www.youtube.com/@", "", 1)

		nodes, err := srv.scrollAndFind(ctx, selectorByType[resource.Type], resource.Count)
		if err != nil {
			return err
		}

		var wg sync.WaitGroup
		sem := semaphore.NewWeighted(maxConcurrentDownloads)

		for _, node := range nodes {
			wg.Add(1)

			err := sem.Acquire(ctx, 1)
			if err != nil {
				return err
			}

			go func() {
				defer wg.Done()
				defer sem.Release(1)

				err := srv.SaveNetscapeCookies(ctx, srv.cookiesPath)
				if err != nil {
					log.Error(err)
					return
				}

				href, _ := node.Attribute("href")
				title, _ := node.Attribute("title")
				remotePath := path.Join("youtube", resource.PathToStorage, channel, resource.Type)
				title = utils.SanitizeFilename(title)
				videoHash := fmt.Sprintf("%x", md5.Sum([]byte(href)))

				has, err := srv.crawlerRepository.HasURLByHash(ctx, videoHash)
				if err != nil {
					log.Error(err)
					return
				}

				if has {
					return
				}

				err = srv.DownloadProcess(fmt.Sprintf("https://www.youtube.com%s", href), srv.outputPath, title, remotePath)
				if err != nil {
					log.Error(err)
					return
				}

				err = srv.crawlerRepository.SaveURLByHash(ctx, videoHash, href)
				if err != nil {
					log.Error(err)
					return
				}
			}()
		}

		wg.Wait()
	}

	return nil
}

func (srv *Service) CrawlOnce(ctx context.Context, resources []model.Resource) (string, error) {
	if err := srv.get(ctx, "https://www.youtube.com"); err != nil {
		return "", err
	}

	err := srv.SaveNetscapeCookies(ctx, srv.cookiesPath)
	if err != nil {
		return "", err
	}

	pathHash := fmt.Sprintf("%x", md5.Sum([]byte(time.Now().String())))
	remotePath := path.Join("youtube", "downloads", pathHash)

	var wg sync.WaitGroup
	sem := semaphore.NewWeighted(maxConcurrentDownloads)

	for _, resource := range resources {
		wg.Add(1)

		err := sem.Acquire(ctx, 1)
		if err != nil {
			return "", err
		}

		go func() {
			defer wg.Done()
			defer sem.Release(1)

			videoHash := fmt.Sprintf("%x", md5.Sum([]byte(resource.URL)))

			err = srv.DownloadProcess(resource.URL, srv.outputPath, videoHash, remotePath)
			if err != nil {
				log.Error(err)
				return
			}
		}()

	}

	wg.Wait()

	link, err := srv.dropboxClient.GetSharingLink(remotePath)
	if err != nil {
		return "", nil
	}

	return link, nil
}

func (srv *Service) DownloadProcess(
	href string,
	downloadPath string,
	title string,
	remoteOutputPath string,
) error {
	logFields := log.Fields{
		"href":             href,
		"downloadPath":     downloadPath,
		"title":            title,
		"remoteOutputPath": remoteOutputPath,
	}

	log.WithFields(logFields).Println("downloading start ...")

	err := utils.DownloadVideoWithCookies(
		href,
		srv.cookiesPath,
		downloadPath,
		title,
	)
	if err != nil {
		return err
	}

	log.WithFields(logFields).Println("downloading end ...")

	filepath, err := utils.FindFileByName(downloadPath, title)
	if err != nil {
		return err
	}

	splitFilePath := strings.Split(filepath, "\\")
	fileName := splitFilePath[len(splitFilePath)-1]

	remoteOutput := path.Join(remoteOutputPath, fileName)

	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			log.Error(err)
		}
	}(filepath)

	log.WithFields(logFields).Println("Uploading start ...")

	err = srv.dropboxClient.Upload(filepath, remoteOutput)
	if err != nil {
		return err
	}

	log.WithFields(logFields).Println("Uploading end ...")

	return nil
}

func (srv *Service) get(ctx context.Context, url string) error {
	err := chromedp.Run(
		ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(everyRedirectSleep),
	)
	if err != nil {
		return err
	}

	return nil
}

func (srv *Service) scrollAndFind(ctx context.Context, selector string, count int) ([]*cdp.Node, error) {
	var nodes []*cdp.Node

	tmp := 0

	for tmp <= count {
		crawlnodes, err := srv.getNodesBySelector(ctx, selector)
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, crawlnodes...)

		if len(nodes) == tmp {
			break
		}

		tmp = len(nodes)

		if tmp <= count {
			if err := srv.scrollBy(ctx, 0, 1400, everyRedirectSleep); err != nil {
				return nil, err
			}
		}
	}

	if (count - tmp) >= 0 {
		return nodes, nil
	}

	return nodes[:count], nil
}

func (srv *Service) getNodesBySelector(ctx context.Context, selector string) ([]*cdp.Node, error) {
	var nodes []*cdp.Node

	if err := chromedp.Run(ctx, chromedp.Nodes(selector, &nodes)); err != nil {
		return nil, err
	}

	return nodes, nil
}

func (srv *Service) scrollBy(
	ctx context.Context,
	from uint16,
	to uint16,
	sleepDuration time.Duration,
) error {
	return chromedp.Run(ctx,
		chromedp.Evaluate(fmt.Sprintf("window.scrollBy(%d,%d);", from, to), nil),
		chromedp.Sleep(sleepDuration),
	)
}

func (srv *Service) SaveNetscapeCookies(ctx context.Context, path string) error {
	err := chromedp.Run(
		ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			var buffer bytes.Buffer
			var cookiesStr []byte

			cookies, err := network.GetCookies().Do(ctx)
			if err != nil {
				return err
			}

			buffer.WriteString("# Netscape HTTP Cookie File\n")
			buffer.WriteString("# http://curl.haxx.se/rfc/cookie_spec.html\n")
			buffer.WriteString("# This is a generated file! Do not edit.\n")

			boolToString := func(value bool) string {
				if value {
					return "TRUE"
				} else {
					return "FALSE"
				}
			}

			for _, c := range cookies {
				buffer.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
					c.Domain,
					boolToString(c.Domain[0] == '.'),
					c.Path,
					boolToString(c.Secure),
					c.Name,
					c.Value,
				))
			}

			cookiesStr = buffer.Bytes()

			err = os.WriteFile(path, cookiesStr, 0644)
			if err != nil {
				return err
			}

			return nil
		}))
	if err != nil {
		return err
	}

	return nil
}
