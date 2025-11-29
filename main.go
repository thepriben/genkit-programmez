package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

// QuestionInput is the typed input for the QA flow.
type QuestionInput struct {
	Question string `json:"question"`
}

// AnswerOutput is the typed output for the QA flow.
type AnswerOutput struct {
	Answer string `json:"answer"`
}

// CyclingRAGInput carries a free-form question about cycling transfers.
type CyclingRAGInput struct {
	Question string `json:"question"`
}

// CyclingRAGOutput returns the answer and the list of sources used.
type CyclingRAGOutput struct {
	Answer  string   `json:"answer"`
	Sources []string `json:"sources"`
}

const (
	maxItemsPerFeed     = 5
	defaultCyclingQuery = "Quelles sont les dernières mutations et transferts en cyclisme ?"
)

var cyclingFeeds = []struct {
	name string
	urls []string
}{
	{
		name: "L'Équipe (Cyclisme)",
		urls: []string{
			"https://dwh.lequipe.fr/api/edito/rss?path=/Cyclisme/",
		},
	},
	{
		name: "DirectVelo",
		urls: []string{
			"https://feeds.feedburner.com/ActualitsDirectvelo",
		},
	},
}

var transferKeywords = []string{
	"transfert", "transfer", "mutation", "mercato", "signe", "signature",
	"recrut", "rejoint", "quitte", "engage", "arrive", "contrat", "renforce",
}

type rssItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	PubDate string `xml:"pubDate"`
}

type rssFeed struct {
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

func main() {
	ctx := context.Background()

	// Initialize Genkit with the Google AI plugin (expects GOOGLE_API_KEY in the environment).
	g, err := genkit.Init(ctx,
		genkit.WithPlugins(&googlegenai.GoogleAI{}),
	)
	if err != nil {
		log.Fatal(err)
	}

	qaFlow := genkit.DefineFlow(g, "qaFlow",
		func(ctx context.Context, in QuestionInput) (AnswerOutput, error) {
			resp, err := genkit.Generate(ctx, g,
				ai.WithModelName("googleai/gemini-2.0-flash"),
				ai.WithPrompt(in.Question),
			)
			if err != nil {
				return AnswerOutput{}, err
			}
			return AnswerOutput{Answer: resp.Text()}, nil
		},
	)

	out, err := qaFlow.Run(ctx, QuestionInput{
		Question: "Le magazine Programmez!, donne-moi les informations principales en trois phrases.",
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Question : %s", "Le magazine Programmez!, donne-moi les informations principales en trois phrases.")
	log.Printf("Réponse : %s", out.Answer)
	log.Println("")
	log.Println("---- Début RAG cyclisme ----")

	// Example RAG run focused on cycling transfer news.
	ragFlow := genkit.DefineFlow(g, "cyclingRAG",
		func(ctx context.Context, in CyclingRAGInput) (CyclingRAGOutput, error) {
			question := strings.TrimSpace(in.Question)
			if question == "" {
				question = defaultCyclingQuery
			}

			snippets, sources, err := fetchCyclingContext(ctx)
			if err != nil {
				return CyclingRAGOutput{}, err
			}

			contextBlock := strings.Join(snippets, "\n")
			prompt := fmt.Sprintf(
				"Tu es un assistant cyclisme.\n"+
					"Contexte issu de flux d'actualités (mutations/transferts) :\n%s\n\n"+
					"Question : %s\n"+
					"Réponds en français par une liste concise de mutations : Nom — équipe actuelle -> équipe annoncée (ou rumeur). Si l'équipe n'est pas précisée, indique 'vers équipe inconnue'.",
				contextBlock, question,
			)

			resp, err := genkit.Generate(ctx, g,
				ai.WithModelName("googleai/gemini-2.0-flash"),
				ai.WithPrompt(prompt),
			)
			if err != nil {
				return CyclingRAGOutput{}, err
			}

			return CyclingRAGOutput{
				Answer:  resp.Text(),
				Sources: sources,
			}, nil
		},
	)

	ragOut, err := ragFlow.Run(ctx, CyclingRAGInput{
		Question: "Quelles sont les dernières mutations dans le cyclisme pro ?",
	})
	if err != nil {
		log.Printf("RAG cycling error: %v", err)
	} else {
		logRAGSummaries(ragOut.Answer)
	}
	log.Println("---- Fin RAG cyclisme ----")
}

func fetchCyclingContext(ctx context.Context) ([]string, []string, error) {
	var snippets []string
	var sources []string

	for _, feed := range cyclingFeeds {
		items, srcURL, err := fetchFirstWorkingFeed(ctx, feed.urls, maxItemsPerFeed)
		if err != nil {
			log.Printf("skip feed %s: %v", feed.name, err)
			continue
		}
		for _, it := range filterTransferItems(items) {
			date := it.PubDate
			if date == "" {
				date = "date inconnue"
			}
			snippets = append(snippets, fmt.Sprintf("- %s (%s)", it.Title, date))
			if it.Link != "" {
				sources = append(sources, it.Link)
			}
		}
		if srcURL != "" {
			sources = append(sources, srcURL)
		}
	}

	if len(snippets) == 0 {
		log.Printf("warning: aucun flux cyclisme accessible, usage d'un contexte de secours.")
		snippets = append(snippets, "- Aucun flux cyclisme accessible pour le moment. Réponds de façon générale et prudente sur les transferts récents.")
	}

	return snippets, sources, nil
}

func logRAGSummaries(answer string) {
	log.Println("Mutations détectées :")
	lines := strings.Split(answer, "\n")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		trimmed = strings.TrimPrefix(trimmed, "*")
		trimmed = strings.TrimPrefix(trimmed, "-")
		trimmed = strings.TrimSpace(trimmed)
		if trimmed != "" {
			log.Printf("- %s", trimmed)
		}
	}
}

func filterTransferItems(items []rssItem) []rssItem {
	var filtered []rssItem
	for _, it := range items {
		titleLower := strings.ToLower(it.Title)
		for _, kw := range transferKeywords {
			if strings.Contains(titleLower, kw) {
				filtered = append(filtered, it)
				break
			}
		}
	}
	// If nothing matched, fall back to the original list to avoid empty context per feed.
	if len(filtered) == 0 {
		return items
	}
	return filtered
}

func fetchFirstWorkingFeed(ctx context.Context, urls []string, limit int) ([]rssItem, string, error) {
	for _, feedURL := range urls {
		items, err := fetchRSSItems(ctx, feedURL, limit)
		if err == nil && len(items) > 0 {
			return items, feedURL, nil
		}
		if err != nil {
			log.Printf("feed attempt failed (%s): %v", feedURL, err)
		}
	}
	return nil, "", fmt.Errorf("no working URL among %v", urls)
}

func fetchRSSItems(ctx context.Context, feedURL string, limit int) ([]rssItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "genkit-cycling-rag/1.0 (+https://github.com/thepriben/genkit-programmez)")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var feed rssFeed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, err
	}

	items := feed.Channel.Items
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}
