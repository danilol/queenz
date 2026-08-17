package persona

import (
	"context"
	"fmt"
	"io"
	"strings"

	lineagedomain "queenx/internal/lineage/domain"
	"queenx/internal/persona/domain"
	"queenx/internal/platform/llm"
)

// LineageReader defines the subset of lineage context required by the Persona Service.
type LineageReader interface {
	FindQueensByClassifications(ctx context.Context, classifications []string, limit int) ([]*lineagedomain.Queen, error)
}

// GeminiClient defines the streaming interface for Gemini.
type GeminiClient interface {
	StreamGenerate(ctx context.Context, systemPrompt, prompt string, writer io.Writer) (string, error)
}

// FalClient defines the image generation interface for Fal.ai.
type FalClient interface {
	GenerateImage(ctx context.Context, prompt string) (string, error)
}

type PersonaService struct {
	lineage      LineageReader
	gemini       GeminiClient
	fal          FalClient
	promptSchema string
}

func NewPersonaService(lineage LineageReader, gemini GeminiClient, fal FalClient) *PersonaService {
	// Pre-generate the JSON instruction from the domain struct to use in prompts.
	schemaInstruction := llm.GenerateJSONInstruction(domain.PersonaResult{})
	return &PersonaService{
		lineage:      lineage,
		gemini:       gemini,
		fal:          fal,
		promptSchema: schemaInstruction,
	}
}

// GeneratePersona coordinates finding context, prompt construction, LLM streaming, parsing, and image generation.
func (s *PersonaService) GeneratePersona(ctx context.Context, req *domain.GeneratePersonaRequest, sseWriter io.Writer) (*domain.PersonaResult, string, error) {
	if len(req.Traits) == 0 {
		return nil, "", domain.ErrInvalidTraits
	}

	// 1. Query lineage context for similar queens matching the given traits/classifications.
	similarQueens, err := s.lineage.FindQueensByClassifications(ctx, req.Traits, 3)
	if err != nil {
		// Log error but proceed without context to keep the service resilient (graceful fallback)
		similarQueens = nil
	}

	// 2. Build system prompt & user prompt
	systemPrompt := s.buildSystemPrompt()
	userPrompt := s.buildUserPrompt(req.Traits, similarQueens)

	// 3. Stream text from Gemini
	fullText, err := s.gemini.StreamGenerate(ctx, systemPrompt, userPrompt, sseWriter)
	if err != nil {
		return nil, "", fmt.Errorf("gemini streaming failed: %w", err)
	}

	// 4. Parse & Validate accumulated text
	result, err := llm.UnmarshalAndValidate[domain.PersonaResult](fullText)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse structured LLM output: %w", err)
	}

	// 5. Generate high-resolution image asynchronously using Fal.ai
	imageURL, err := s.fal.GenerateImage(ctx, result.ImageGenerationPrompt)
	if err != nil {
		return &result, "", fmt.Errorf("fal.ai image generation failed: %w", err)
	}

	return &result, imageURL, nil
}

func (s *PersonaService) buildSystemPrompt() string {
	return "You are an expert drag queen coach, consultant, and creative director for the ultimate drag queen simulation platform.\n" +
		"Your mission is to craft a completely unique, charismatic, and legendary drag queen persona matching the user's selected traits.\n" +
		"Ensure you follow the strict output format required.\n\n" +
		s.promptSchema
}

func (s *PersonaService) buildUserPrompt(traits []string, contextQueens []*lineagedomain.Queen) string {
	var sb strings.Builder
	_, _ = fmt.Fprintf(&sb, "Create a unique drag queen with these traits: %s.\n\n", strings.Join(traits, ", "))

	if len(contextQueens) > 0 {
		_, _ = sb.WriteString("For inspiration, look at these real-world legendary queens with overlapping traits/classifications:\n")
		for _, q := range contextQueens {
			_, _ = fmt.Fprintf(&sb, "- Name: %s, Traits: %s\n", q.DragName, strings.Join(q.Classifications, ", "))
		}
		_, _ = sb.WriteString("\n")
	}

	sb.WriteString("Your output MUST include:\n" +
		"1. dragName: A fabulous, puns-galore, or legendary drag name.\n" +
		"2. bio: A funny, high-drama, and witty back-story/bio.\n" +
		"3. stats: Integer scores from 1 to 10 for: glamour, comedy, dance, and camp.\n" +
		"4. imageGenerationPrompt: A highly detailed, gorgeous Flux image prompt describing a full-glamour portrait of this drag queen in exquisite fashion, makeup, and hair, specifically matching their traits.\n" +
		"Ensure the response is ONLY valid JSON, with NO backticks or markdown formatting outside of the JSON structure itself.")

	return sb.String()
}
