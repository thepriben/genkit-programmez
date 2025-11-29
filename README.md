# Genkit Go : article *Programmez!*

Petit exemple de flows Genkit Go avec le plugin Google AI (Gemini) :
- `qaFlow` : question → réponse direct.
- `cyclingRAG` : synthèse des dernières mutations/transferts en cyclisme en s’appuyant sur deux flux : L’Équipe (`https://dwh.lequipe.fr/api/edito/rss?path=/Cyclisme/`) et Directvelo (`https://feeds.feedburner.com/ActualitsDirectvelo`). En cas d’échec, les flux sont ignorés et un contexte de secours est utilisé (logs explicites).

## Prérequis
- Go 1.22+
- Une clé Google AI dans `GOOGLE_API_KEY`

## Lancer les flows (CLI)
```
GOOGLE_API_KEY="XXXX" go run .
```

Le programme exécute d’abord `qaFlow`, puis `cyclingRAG`. Ajuste les questions ou le modèle dans `main.go` si besoin.

Le flow `qaFlow` appelle le modèle `googleai/gemini-2.0-flash` et journalise question/réponse. Ajuste la question ou le modèle dans `main.go` si besoin.
