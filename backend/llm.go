package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/alberrttt/langgraphgo/graph"
	"github.com/tmc/langchaingo/llms"
)

type PracticeProblem struct {
	Answer   string `json:"answer"`
	Question string `json:"question"`
}
type Intent struct {
	Continue bool `json:"continue"`
}
type TutorGraphState struct {
	Messages         []llms.MessageContent
	InternalThought  []llms.MessageContent
	PracticeProblems []PracticeProblem
}

func NewTutorGraphState() TutorGraphState {
	return TutorGraphState{
		Messages:         []llms.MessageContent{},
		InternalThought:  []llms.MessageContent{},
		PracticeProblems: []PracticeProblem{},
	}
}

func (s *TutorGraphState) AddMessage(message llms.MessageContent) {
	s.Messages = append(s.Messages, message)
}

func (s *TutorGraphState) LastMessage() llms.MessageContent {
	return s.Messages[len(s.Messages)-1]
}

func (s *TutorGraphState) LastInternalThought() llms.MessageContent {
	return s.InternalThought[len(s.InternalThought)-1]
}
func (s *TutorGraphState) AddPracticeProblem(problem PracticeProblem) {
	s.PracticeProblems = append(s.PracticeProblems, problem)
}
func (s *TutorGraphState) AddInternalThought(message string) {
	s.InternalThought = append(s.InternalThought, llms.TextParts(llms.ChatMessageTypeAI, message))
}
func (s *TutorGraphState) PopInternalThought() llms.MessageContent {
	if len(s.InternalThought) == 0 {
		return llms.MessageContent{}
	}
	thought := s.InternalThought[len(s.InternalThought)-1]
	s.InternalThought = s.InternalThought[:len(s.InternalThought)-1]
	return thought
}
func (s *TutorGraphState) NthMessageOf(n int, t func(llms.MessageContent) bool) llms.MessageContent {
	for i := len(s.Messages) - 1; i >= 0; i-- {
		if t(s.Messages[i]) {
			if n == 0 {
				return s.Messages[i]
			}
			n--
		}
	}
	return llms.MessageContent{}
}
func parseAnswerAndQuestion(response string) (PracticeProblem, error) {
	startAnswerTag := "<answer>"
	endAnswerTag := "</answer>"
	startQuestionTag := "<question>"
	endQuestionTag := "</question>"

	var detail PracticeProblem

	// Helper function to extract content between start and end tags
	extractContent := func(content, startTag, endTag string) (string, error) {
		start := strings.Index(content, startTag)
		if start == -1 {
			return content[start:], errors.New("missing start tag: " + startTag)
		}
		start += len(startTag)
		end := strings.Index(content[start:], endTag)
		if end == -1 {
			return content[start:], errors.New("missing end tag: " + endTag)
		}
		end += start
		return strings.TrimSpace(content[start:end]), nil
	}

	// Extract Answer
	answer, err := extractContent(response, startAnswerTag, endAnswerTag)
	if err != nil {
		return detail, err
	}
	detail.Answer = answer

	// Extract Question
	question, err := extractContent(response, startQuestionTag, endQuestionTag)
	if err != nil {
		return detail, err
	}
	detail.Question = question

	return detail, nil
}
func parsePracticeProblems(response string) ([]string, error) {
	startTag := "<practice>"
	endTag := "</practice>"

	var problems []string
	currentIndex := 0

	for {
		// Find the start of the next <practice> tag
		start := strings.Index(response[currentIndex:], startTag)
		if start == -1 {
			break // No more <practice> tags found
		}
		start += currentIndex + len(startTag)

		// Find the end of the </practice> tag
		end := strings.Index(response[start:], endTag)
		if end == -1 {
			return nil, errors.New("malformed response: missing </practice> tag")
		}
		end += start

		// Extract the content between the tags and trim whitespace
		problem := strings.TrimSpace(response[start:end])
		problems = append(problems, problem)

		// Move the current index past the end of the current </practice> tag
		currentIndex = end + len(endTag)
	}

	if len(problems) == 0 {
		return nil, errors.New("no practice problems found in the response")
	}

	return problems, nil
}

func setupGraph(g *graph.StateGraph[TutorGraphState], model llms.Model, llama405b llms.Model) {
	g.SetEntryPoint("assistant")
	g.AddNode("assistant", func(ctx context.Context, state *TutorGraphState) error {
		system := "You are a helpful assistant. "
		response, err := model.GenerateContent(ctx, []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, system),
			llms.TextParts(llms.ChatMessageTypeHuman, state.LastMessage().Parts[0].(llms.TextContent).Text),
		}, llms.WithTemperature(0))
		if err != nil {
			return err
		}
		state.AddMessage(llms.TextParts(llms.ChatMessageTypeAI, response.Choices[0].Content))
		return nil
	})
	g.AddConditionalEdges("assistant", func(ctx context.Context, state *TutorGraphState) ([]string, error) {
		sys_prompt := "In the following conversation, is the user EXPLICITLY seeking help or practice? If so, respond with `{ \"continue\": true }`. If the user is not seeking help or practice, respond with `{ \"continue\": false }`. Only respond in JSON format with the single field `continue`, which is a boolean value."
		response, err := model.GenerateContent(ctx, append(state.Messages, llms.TextParts(llms.ChatMessageTypeSystem, sys_prompt)), llms.WithTemperature(0))
		if err != nil {
			return nil, err
		}
		var intent Intent

		err = json.Unmarshal([]byte(response.Choices[0].Content), &intent)
		if err != nil {
			return nil, err
		}
		log.Println(intent)
		log.Printf("%v", response.Choices[0].Content)
		if intent.Continue {
			return []string{"tutor"}, nil
		}
		return []string{graph.END}, nil
	})
	g.AddNode("tutor", func(ctx context.Context, state *TutorGraphState) error {
		prompt := "You are a tutor. Given the user's prompt, in first person, think about what to do to address it, be detailed, and have a plan. You do not need to actually address the prompt now, just think about plans. Ask yourself what questions could help you understand the user's intent better. Respond in this format: `\"address_prompt\": \"<your thought>\", \"questions_for_user\": [\"<question 1>\", \"<question 2>\"]`"
		response, err := model.GenerateContent(ctx, []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, prompt),
			state.NthMessageOf(0, func(m llms.MessageContent) bool {
				return m.Role == llms.ChatMessageTypeHuman
			}),
		}, llms.WithTemperature(0.5))
		if err != nil {
			return err
		}
		log.Println(response.Choices[0].Content)
		state.AddInternalThought(response.Choices[0].Content)
		return nil
	})
	g.AddNode("draft_practice_problems", func(ctx context.Context, state *TutorGraphState) error {
		prompt := "You are a tutor. Draft up practice problems. Format your answer by using <practice> </practice> tags. Each practice problem MUST be inside its own <practice> </practice> tag."
		response, err := model.GenerateContent(ctx, []llms.MessageContent{
			state.LastInternalThought(),
			llms.TextParts(llms.ChatMessageTypeSystem, prompt),
		}, llms.WithTemperature(0.5))
		log.Println(response.Choices[0].Content)
		if err != nil {
			return err
		}
		state.AddInternalThought(response.Choices[0].Content)

		prompt = fmt.Sprintf("Summarize what's going on in these practice problems. Don't answer them, just give a summary of what the user would learn. Write it as if you are talking to the user. Also address the user's initial prompt which is: \n %s", state.NthMessageOf(0, func(mc llms.MessageContent) bool {
			return mc.Role == llms.ChatMessageTypeHuman
		}).Parts[0].(llms.TextContent).Text)
		response, err = model.GenerateContent(ctx, []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, prompt),
			llms.TextParts(llms.ChatMessageTypeHuman, response.Choices[0].Content),
		}, llms.WithTemperature(0.5))
		if err != nil {
			return err
		}
		state.AddMessage(llms.TextParts(llms.ChatMessageTypeAI, response.Choices[0].Content))
		return nil
	})
	g.AddEdge("tutor", "draft_practice_problems")
	g.AddNode("finalize_practice_problems", func(ctx context.Context, state *TutorGraphState) error {
		draft := state.PopInternalThought()
		problems, err := parsePracticeProblems(draft.Parts[0].(llms.TextContent).Text)
		if err != nil {
			return err
		}
		for _, problem := range problems {

			prompt := fmt.Sprintf("You are a tutor. You have access to KaTeX and mhchem, expressions are delimited by $$. Please use LaTeX for math expressions and chemical formulas. Given the following practice problem, create an answer, then the question for that answer.The answer MUST be inside its own <answer> </answer> tag, and the question MUST be inside its own <question> </question> tag. All tags MUST be closed. The practice problem is:\n\n %s", problem)
			response, err := llama405b.GenerateContent(ctx, []llms.MessageContent{
				llms.TextParts(llms.ChatMessageTypeSystem, prompt),
			}, llms.WithTemperature(0.25))
			if err != nil {
				return err
			}
			practice_problem, err := parseAnswerAndQuestion(response.Choices[0].Content)
			if err != nil {
				log.Println(response.Choices[0].Content)
				log.Println(err)
			}
			state.AddPracticeProblem(practice_problem)

		}
		return nil
	})
	g.AddEdge("draft_practice_problems", "finalize_practice_problems")
	g.AddEdge("finalize_practice_problems", graph.END)
}
