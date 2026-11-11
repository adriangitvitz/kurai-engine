package ranker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func tokens(s string) []string {
	t := []string{}
	for _, w := range splitOnSpaces(s) {
		if w != "" {
			t = append(t, w)
		}
	}
	return t
}

func splitOnSpaces(s string) []string {
	out := []string{""}
	for _, r := range s {
		if r == ' ' {
			out = append(out, "")
		} else {
			out[len(out)-1] += string(r)
		}
	}
	return out
}

func TestDetectIntent_WhatIsRoutesToAcademic(t *testing.T) {
	intent := DetectIntent(tokens("what is cognitive behavioral therapy"))
	assert.Equal(t, CategoryAcademic, intent.Category, "what is X → academic")
	assert.Greater(t, intent.Confidence, 0.6, "confidence should be > 0.6 for clear what-is patterns")
}

func TestDetectIntent_HowDoesItWorkRoutesToAcademic(t *testing.T) {
	intent := DetectIntent(tokens("how does insulin work in the body"))
	assert.Equal(t, CategoryAcademic, intent.Category)
	assert.Greater(t, intent.Confidence, 0.6)
}

func TestDetectIntent_WhyDoesRoutesToAcademic(t *testing.T) {
	intent := DetectIntent(tokens("why is the sky blue"))
	assert.Equal(t, CategoryAcademic, intent.Category)
}

func TestDetectIntent_QuestionDoesNotOverrideStrongerSignal(t *testing.T) {
	intent := DetectIntent(tokens("how to fix python error"))
	assert.Equal(t, CategoryTech, intent.Category, "tech signals must beat the academic question-form bias")
}

func TestDetectIntent_WhenIsRoutesToNews(t *testing.T) {
	intent := DetectIntent(tokens("when did the berlin wall fall"))
	assert.NotEqual(t, CategoryGeneral, intent.Category)
}

func TestDetectIntent_PlainQueryStillGeneral(t *testing.T) {
	intent := DetectIntent(tokens("petrichor"))
	assert.Equal(t, CategoryGeneral, intent.Category)
}
