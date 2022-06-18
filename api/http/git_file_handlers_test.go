package http

import (
	"bytes"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/api/auth"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
)

func TestGitFileHandlers(t *testing.T) {
	repo := repository.CreateGoGitTestRepo(t, false)

	mrc := cache.NewMultiRepoCache()
	repoCache, err := mrc.RegisterDefaultRepository(repo)
	require.NoError(t, err)

	author, err := repoCache.NewIdentity("test identity", "test@test.org")
	require.NoError(t, err)

	err = repoCache.SetUserIdentity(author)
	require.NoError(t, err)

	// UPLOAD

	uploadHandler := NewGitUploadFileHandler(mrc)

	img := image.NewNRGBA(image.Rect(0, 0, 50, 50))
	data := &bytes.Buffer{}
	err = png.Encode(data, img)
	require.NoError(t, err)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("uploadfile", "noname")
	assert.NoError(t, err)

	_, err = part.Write(data.Bytes())
	assert.NoError(t, err)

	err = writer.Close()
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/", body)
	r.Header.Add("Content-Type", writer.FormDataContentType())

	// Simulate auth
	r = r.WithContext(auth.CtxWithUser(r.Context(), author.Id()))

	// Handler's params
	r = mux.SetURLVars(r, map[string]string{
		"repo": "",
	})

	uploadHandler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `{"hash":"3426a1488292d8f3f3c59ca679681336542b986f"}`, w.Body.String())
	// DOWNLOAD

	downloadHandler := NewGitFileHandler(mrc)

	w = httptest.NewRecorder()
	r, _ = http.NewRequest("GET", "/", nil)

	// Simulate auth
	r = r.WithContext(auth.CtxWithUser(r.Context(), author.Id()))

	// Handler's params
	r = mux.SetURLVars(r, map[string]string{
		"repo": "",
		"hash": "3426a1488292d8f3f3c59ca679681336542b986f",
	})

	downloadHandler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	assert.Equal(t, data.Bytes(), w.Body.Bytes())
}
