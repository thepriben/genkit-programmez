# Genkit Go : article *Programmez!*

Petit exemple de flows Genkit Go avec le plugin Google AI (Gemini) :
- `qaFlow` : question → réponse ;
- `cyclingRAG` : synthèse des dernières mutations/transferts en cyclisme en s’appuyant sur deux flux RSS : [*L’Équipe* > Cyclisme](https://dwh.lequipe.fr/api/edito/rss?path=/Cyclisme/) et [directvelo.com](https://feeds.feedburner.com/ActualitsDirectvelo).

## Prérequis
- Go 1.22+
- Une clé Google AI dans `GOOGLE_API_KEY`

## Lancer les flows (CLI)
```
GOOGLE_API_KEY="XXXX" go run .
```
