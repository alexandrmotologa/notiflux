package api

import (
	"encoding/json"
	"net/http"

	"github.com/alexandrmotologa/notiflux/internal/engine"
	"github.com/alexandrmotologa/notiflux/internal/store"
	"github.com/go-chi/chi/v5"
)

// WorkflowHandler coordinates workflow definition and template management endpoints.
type WorkflowHandler struct {
	store    *store.Store
	compiler *engine.Compiler
}

func NewWorkflowHandler(s *store.Store) *WorkflowHandler {
	return &WorkflowHandler{
		store:    s,
		compiler: engine.NewCompiler(),
	}
}

func (h *WorkflowHandler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.ListWorkflows()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if list == nil {
		list = []*store.Workflow{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *WorkflowHandler) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var wf store.Workflow
	if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json: " + err.Error()})
		return
	}

	if wf.ID == "" || wf.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id and name are required"})
		return
	}
	if len(wf.Channels) == 0 {
		wf.Channels = []string{store.ChannelInApp}
	}

	if err := h.store.CreateWorkflow(&wf); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, wf)
}

func (h *WorkflowHandler) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	wf, err := h.store.GetWorkflow(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if wf == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "workflow not found"})
		return
	}

	templates, _ := h.store.GetTemplatesByWorkflow(id)
	if templates == nil {
		templates = []*store.Template{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"workflow":  wf,
		"templates": templates,
	})
}

func (h *WorkflowHandler) UpdateWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var wf store.Workflow
	if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json: " + err.Error()})
		return
	}
	wf.ID = id

	if err := h.store.UpdateWorkflow(&wf); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, wf)
}

func (h *WorkflowHandler) DeleteWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.DeleteWorkflow(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "deleted_id": id})
}

func (h *WorkflowHandler) GetTemplates(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	templates, err := h.store.GetTemplatesByWorkflow(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if templates == nil {
		templates = []*store.Template{}
	}
	writeJSON(w, http.StatusOK, templates)
}

func (h *WorkflowHandler) UpdateTemplates(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var body struct {
		Templates []struct {
			Channel string `json:"channel"`
			Subject string `json:"subject"`
			Body    string `json:"body"`
		} `json:"templates"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json: " + err.Error()})
		return
	}

	for _, item := range body.Templates {
		tpl := &store.Template{
			WorkflowID: id,
			Channel:    item.Channel,
			Subject:    item.Subject,
			Body:       item.Body,
		}
		if err := h.store.UpsertTemplate(tpl); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}

	templates, _ := h.store.GetTemplatesByWorkflow(id)
	writeJSON(w, http.StatusOK, map[string]any{
		"success":   true,
		"templates": templates,
	})
}

func (h *WorkflowHandler) PreviewTemplate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Channel string         `json:"channel"`
		Subject string         `json:"subject"`
		Body    string         `json:"body"`
		Data    map[string]any `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json: " + err.Error()})
		return
	}

	escapeHTML := (body.Channel == store.ChannelEmail)
	renderedSubject, _ := h.compiler.Render(body.Subject, body.Data, false)
	renderedBody, _ := h.compiler.Render(body.Body, body.Data, escapeHTML)

	htmlOutput := ""
	if body.Channel == store.ChannelEmail {
		rawBody, _ := h.compiler.Render(body.Body, body.Data, false)
		converted := engine.MarkdownToHTML(rawBody)
		htmlOutput = engine.WrapEmailHTML(converted, renderedSubject)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"subject":  renderedSubject,
		"body":     renderedBody,
		"html":     htmlOutput,
		"channel":  body.Channel,
	})
}
