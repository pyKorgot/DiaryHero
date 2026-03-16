package narrator

import (
	"context"
	"fmt"
	"strings"

	"diaryhero/internal/domain"
	"diaryhero/internal/openrouter"
)

type Service struct {
	client *openrouter.Client
}

func New(client *openrouter.Client) *Service {
	return &Service{client: client}
}

func (s *Service) GenerateEntry(ctx context.Context, input domain.NarrativeInput) (domain.NarrativeOutput, error) {
	if s.client == nil || !s.client.Enabled() {
		return stubEntry(input), nil
	}

	response, err := s.client.ChatCompletion(ctx, openrouter.ChatCompletionRequest{
		Messages: []openrouter.Message{
			{Role: "system", Content: buildSystemPrompt()},
			{Role: "user", Content: buildUserPrompt(input)},
		},
	})
	if err != nil {
		fallback := stubEntry(input)
		fallback.Source = "stub-fallback"
		fallback.Model = response.Model
		return fallback, nil
	}

	if len(response.Choices) == 0 {
		fallback := stubEntry(input)
		fallback.Source = "stub-empty"
		fallback.Model = response.Model
		return fallback, nil
	}

	text := strings.TrimSpace(response.Choices[0].Message.Content)
	if text == "" {
		fallback := stubEntry(input)
		fallback.Source = "stub-empty"
		fallback.Model = response.Model
		return fallback, nil
	}

	return domain.NarrativeOutput{
		Text:   text,
		Source: "openrouter",
		Model:  response.Model,
	}, nil
}

func buildSystemPrompt() string {
	return strings.Join([]string{
		"You write short diary entries for a persistent fantasy hero.",
		"Rules:",
		"- Write in Russian.",
		"- First person only.",
		"- Keep continuity with the provided state and event.",
		"- Make it feel like a personal diary note, not a game log.",
		"- Length: 350-700 characters.",
		"- Mention concrete details, but do not overexplain.",
		"- Do not invent major world changes not supported by the event.",
	}, "\n")
}

func buildUserPrompt(input domain.NarrativeInput) string {
	return fmt.Sprintf(`Hero:
- name: %s
- archetype: %s
- voice: %s

Current state:
- location: %s
- time: %s
- health: %d
- energy: %d
- stress: %d
- gold: %d

Current event:
- code: %s
- title: %s
- payload: %s
- outcome: %s

Task:
Write the next diary entry this hero would publish right now.`,
		input.Hero.Name,
		input.Hero.Archetype,
		input.Hero.VoiceStyle,
		input.HeroState.LocationCode,
		input.HeroState.CurrentTime,
		input.HeroState.Health,
		input.HeroState.Energy,
		input.HeroState.Stress,
		input.HeroState.Gold,
		input.EventType.Code,
		input.EventType.Title,
		input.WorldEvent.PayloadJSON,
		input.WorldEvent.OutcomeJSON,
	)
}

func stubEntry(input domain.NarrativeInput) domain.NarrativeOutput {
	text := fmt.Sprintf(
		"Сегодня у меня вышло что-то вроде '%s'. Я все еще в %s, на дворе уже %s. Чувствую себя на %d из 100, сил осталось %d, напряжение держится на %d. Денег теперь %d. День вроде бы обычный, но я записываю такие мелочи именно затем, чтобы потом не делать вид, будто все случилось само собой.",
		strings.ToLower(input.EventType.Title),
		input.HeroState.LocationCode,
		input.HeroState.CurrentTime,
		input.HeroState.Health,
		input.HeroState.Energy,
		input.HeroState.Stress,
		input.HeroState.Gold,
	)

	return domain.NarrativeOutput{
		Text:   text,
		Source: "stub",
		Model:  "local-template",
	}
}
