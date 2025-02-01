package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mmcdole/gofeed"
)

var feeds string

func main() {
	feeds = os.Getenv("RSS_FEEDS_URL")

	fp := gofeed.NewParser()

	model, err := initialModel(fp)
	if err != nil {
		slog.Error("model creation exception", slog.Any("error", err))
		os.Exit(1)
	}
	model.list.Title = "RSS feeds"
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		slog.Error("program running error", slog.Any("error", err))
		os.Exit(1)
	}
}

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type model struct {
	list  list.Model
	items []RssFeedItem
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:

		switch msg.String() {

		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			exec.Command("open", m.items[m.list.Index()].Link).Run()
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return docStyle.Render(m.list.View())
}

type RssFeedItem struct {
	FeedTitle string
	ItemTitle string
	Link      string
	Date      *time.Time
}

func (i RssFeedItem) Title() string { return i.ItemTitle }
func (i RssFeedItem) Description() string {
	return fmt.Sprintf("[%v] - %v", i.Date.Format(time.ANSIC), i.FeedTitle)
}
func (i RssFeedItem) FilterValue() string { return i.ItemTitle }

func initialModel(fp *gofeed.Parser) (model, error) {
	items, err := extractFromFeeds(fp)
	if err != nil {
		return model{}, fmt.Errorf("feed parsing error %w", err)
	}
	listItems := []list.Item{}
	for _, item := range items {
		listItems = append(listItems, item)
	}
	return model{
		items: items,
		list:  list.New(listItems, list.NewDefaultDelegate(), 0, 0),
	}, nil
}

func extractFromFeeds(fp *gofeed.Parser) ([]RssFeedItem, error) {
	items := []RssFeedItem{}
	for _, feedUrl := range strings.Split(feeds, ",") {
		feed, err := fp.ParseURL(feedUrl)
		if err != nil {
			return []RssFeedItem{}, fmt.Errorf("feed %v cannot be parsed %w", feedUrl, err)
		}
		for _, item := range feed.Items {
			items = append(items, RssFeedItem{
				FeedTitle: feed.Title,
				ItemTitle: item.Title,
				Link:      item.Link,
				Date:      extractDate(item),
			})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Date.After(*items[j].Date)
	})

	return items, nil
}

func extractDate(item *gofeed.Item) *time.Time {
	if item.PublishedParsed != nil {
		return item.PublishedParsed
	}
	if item.UpdatedParsed != nil {
		return item.UpdatedParsed
	}
	return nil
}
