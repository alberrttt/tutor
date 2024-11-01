package backend

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/alberrttt/langgraphgo/graph"
	"github.com/joho/godotenv"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

type ChatResponse struct {
	Type             string            `json:"type"`
	Content          string            `json:"content"`
	ProblemSolutions []PracticeProblem `json:"problem_solutions"`
	ElapsedTimeMs    int64             `json:"elapsed_time_ms"`
}

func Server() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	api_key := os.Getenv("SAMBANOVA_CLOUD_API_KEY")
	if api_key == "" {
		log.Fatal("SAMBANOVA_CLOUD_API_KEY is not set")
	}
	fs := http.FileServer(http.Dir("./build")) // Change "./build" to your actual build directory

	llama3b, err := openai.New(
		openai.WithBaseURL("https://api.sambanova.ai/v1/"),
		openai.WithToken(api_key),
		openai.WithModel("Meta-Llama-3.2-3B-Instruct"),
	)
	llama405b, err := openai.New(
		openai.WithBaseURL("https://api.sambanova.ai/v1/"),
		openai.WithToken(api_key),
		openai.WithModel("Meta-Llama-3.1-405B-Instruct"),
	)
	if err != nil {
		log.Fatal(err)
	}

	g := graph.NewStateGraph[TutorGraphState]()
	setupGraph(g, llama3b, llama405b)
	runnable, err := g.Compile()
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/new", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusOK)

	})
	tutor_state := NewTutorGraphState()

	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Set headers for streaming
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")

		message := r.URL.Query().Get("message")
		if message == "" {
			http.Error(w, "Missing message parameter", http.StatusBadRequest)
			return
		}
		tutor_state.AddMessage(llms.TextParts(llms.ChatMessageTypeHuman, message))
		now := time.Now()
		err := runnable.Invoke(context.Background(), &tutor_state)
		elapsed := time.Since(now)

		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		response := ChatResponse{
			Type:             "assistant",
			Content:          tutor_state.LastMessage().Parts[0].(llms.TextContent).Text,
			ProblemSolutions: tutor_state.PracticeProblems,
			ElapsedTimeMs:    elapsed.Milliseconds(),
		}
		tutor_state.PracticeProblems = []PracticeProblem{}
		json.NewEncoder(w).Encode(response)
	})

	http.Handle("/", fs)

	// Start the server on port 8080
	log.Println("Serving on :8080")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Println("Server error:", err)
	}
}
