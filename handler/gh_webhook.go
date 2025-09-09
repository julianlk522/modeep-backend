package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/go-chi/render"
	e "github.com/julianlk522/modeep/error"
)

func HandleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	signkey_secret := os.Getenv("MODEEP_WEBHOOK_SECRET")
	if signkey_secret == "" {
		render.Render(w, r, e.Err500(e.ErrNoWebhookSecret))
		return
	}

	signature_header := r.Header.Get("X-Hub-Signature-256")
	if signature_header == "" || !strings.HasPrefix(signature_header, "sha256=") {
		render.Render(w, r, e.ErrForbidden(e.ErrNoWebhookSignature))
		return
	}
	log.Printf("Signature header: %s", signature_header)

	// get signature, skipping "sha256="
	gh_hash := signature_header[7:]

	// get payload
	defer r.Body.Close()
	payload_bytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Print("Cannot read GH webhook request payload")
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	}

	// generate hmac using secret
	server_hash := hmac.New(
		sha256.New,
		[]byte(signkey_secret),
	)
	// update hash object with payload
	if _, err := server_hash.Write(payload_bytes); err != nil {
		log.Print("Cannot compute HMAC for GH webhook request body")
		render.Render(w, r, e.Err500(err))
		return
	}

	// generate expected signature
	server_signature := "sha256=" + hex.EncodeToString(server_hash.Sum(nil))
	if !hmac.Equal([]byte(gh_hash), []byte(server_signature)) {
		log.Printf("Signature mismatch: expected %s, got %s", server_signature, gh_hash)
		render.Render(w, r, e.ErrForbidden(e.ErrInvalidWebhookSignature))
		return
	}

	log.Println("Authenticated webhook: updating Modeep backend")

	modeep_root_path := os.Getenv("MODEEP_BACKEND_ROOT")
	cmd := exec.Command("./update_and_restart_backend.sh")
	cmd.Dir = modeep_root_path
	err = cmd.Run()
	if err != nil {
		log.Println(err)
		render.Render(w, r, e.Err500(err))
		return
	}

	render.Status(r, http.StatusOK)
}
