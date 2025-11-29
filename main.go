package main

import (
	"context"
	"log"

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
}
