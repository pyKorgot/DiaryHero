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

	text = sanitizeGeneratedText(text)
	if text == "" {
		fallback := stubEntry(input)
		fallback.Source = "stub-sanitized"
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
		"- The hero is a woman. Use feminine forms consistently.",
		"- Keep continuity with the provided state and event.",
		"- Make it feel like a personal diary note, not a game log.",
		"- Length: 350-700 characters.",
		"- Do not start with the current date, diary heading, or day count.",
		"- Do not mention exact amounts of money or numeric stats.",
		"- Do not talk in terms of health, energy, stress, points, or deltas.",
		"- Prefer concrete sensory details, movement, and mood.",
		"- Mention concrete details, but do not overexplain.",
		"- Do not invent major world changes not supported by the event.",
	}, "\n")
}

func buildUserPrompt(input domain.NarrativeInput) string {
	return fmt.Sprintf(`Hero:
- name: %s
- archetype: %s
- gender: %s
- voice: %s

Current state:
- location: %s
- time: %s
- condition hint: %s

Current event:
- code: %s
- title: %s
- payload: %s
- outcome: %s

Task:
Write the next diary entry this hero would publish right now.`,
		input.Hero.Name,
		input.Hero.Archetype,
		input.Hero.Gender,
		input.Hero.VoiceStyle,
		input.HeroState.LocationTitle,
		input.HeroState.CurrentTime,
		describeCondition(input.HeroState),
		input.EventType.Code,
		input.EventType.Title,
		input.WorldEvent.PayloadJSON,
		input.WorldEvent.OutcomeJSON,
	)
}

func stubEntry(input domain.NarrativeInput) domain.NarrativeOutput {
	text := fmt.Sprintf(
		"В %s к %s случилось %s. %s Я все чаще замечаю, что важнее всего не громкие повороты, а такие вот странные мелочи: чьи-то голоса, запах сырого дерева, быстрый взгляд через плечо. Потом именно они и держат историю вместе.",
		input.HeroState.LocationTitle,
		input.HeroState.CurrentTime,
		strings.ToLower(input.EventType.Title),
		stubMoodLine(input),
	)

	return domain.NarrativeOutput{
		Text:   text,
		Source: "stub",
		Model:  "local-template",
	}
}

func sanitizeGeneratedText(text string) string {
	text = strings.TrimSpace(text)
	text = strings.Trim(text, "`")
	lines := strings.Split(text, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if trimmed == "" {
			if len(filtered) > 0 && filtered[len(filtered)-1] != "" {
				filtered = append(filtered, "")
			}
			continue
		}
		if strings.Contains(lower, " число") || strings.HasPrefix(lower, "день ") || strings.HasPrefix(lower, "запись ") {
			continue
		}
		filtered = append(filtered, trimmed)
	}
	return strings.TrimSpace(strings.Join(filtered, "\n"))
}

func describeCondition(state domain.HeroState) string {
	parts := make([]string, 0, 3)
	if state.Health < 50 {
		parts = append(parts, "a little battered")
	}
	if state.Energy < 35 {
		parts = append(parts, "very tired")
	} else if state.Energy < 60 {
		parts = append(parts, "tired")
	}
	if state.Stress > 65 {
		parts = append(parts, "on edge")
	} else if state.Stress > 35 {
		parts = append(parts, "uneasy")
	}
	if len(parts) == 0 {
		return "steady, alert, and trying not to dwell too much"
	}

	return strings.Join(parts, ", ")
}

func stubMoodLine(input domain.NarrativeInput) string {
	switch {
	case input.HeroState.Energy < 35:
		return "Меня клонит в усталость, и даже обычные звуки вокруг кажутся тяжелее, чем надо."
	case input.HeroState.Stress > 50:
		return "После такого я еще долго прислушивалась к каждому шороху, будто он имел ко мне личное отношение."
	case input.WorldEvent.EventCode == "road_to_new_place" || input.WorldEvent.EventCode == "ferry_crossing":
		return "Смена места немного встряхнула меня, и мысли наконец перестали ходить по кругу."
	default:
		return "Ничего великого не случилось, но именно такие эпизоды потом почему-то и вспоминаются лучше всего."
	}
}
