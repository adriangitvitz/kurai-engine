package server

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adriangitvitz/kurai/internal/apierror"
)

func TestService_Search_MissingQueryReturnsTypedError(t *testing.T) {
	srv := setupTestServer()

	resp, err := srv.ServiceSearch(context.Background(), ServiceSearchReq{
		UserID: "alice",
		Query:  "",
	})

	require.Error(t, err)
	require.Nil(t, resp)

	se, ok := err.(*ServiceError)
	require.True(t, ok, "expected *ServiceError, got %T", err)
	assert.Equal(t, http.StatusBadRequest, se.Status)
	assert.Equal(t, apierror.CodeMissingQuery, se.Code)
}

func TestService_Search_ReturnsEmptyResultsWhenSearXNGAbsent(t *testing.T) {
	srv := setupTestServer()
	resp, err := srv.ServiceSearch(context.Background(), ServiceSearchReq{
		UserID: "alice",
		Query:  "anything",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "anything", resp.Query)
	assert.Empty(t, resp.Results)
}

func TestService_Search_HonorsDebugCategory(t *testing.T) {
	srv := setupTestServer()
	resp, err := srv.ServiceSearch(context.Background(), ServiceSearchReq{
		UserID:   "alice",
		Query:    "what is bm25",
		Debug:    true,
		Category: "academic",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Intent, "intent should be populated when Debug=true")
	assert.Equal(t, "academic", resp.Intent.Category)
}

func TestService_News_MissingQueryReturnsTypedError(t *testing.T) {
	srv := setupTestServer()
	resp, err := srv.ServiceNews(context.Background(), ServiceNewsReq{
		UserID: "alice",
	})
	require.Error(t, err)
	require.Nil(t, resp)

	se, ok := err.(*ServiceError)
	require.True(t, ok)
	assert.Equal(t, apierror.CodeMissingQuery, se.Code)
}

func TestService_Images_MissingQueryReturnsTypedError(t *testing.T) {
	srv := setupTestServer()
	resp, err := srv.ServiceImages(context.Background(), ServiceImagesReq{
		UserID: "alice",
	})
	require.Error(t, err)
	require.Nil(t, resp)

	se, ok := err.(*ServiceError)
	require.True(t, ok)
	assert.Equal(t, apierror.CodeMissingQuery, se.Code)
}
