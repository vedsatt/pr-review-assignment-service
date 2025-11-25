package transport

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/vedsatt/pr-review-assignment-service/internal/models"
	"go.uber.org/zap"
)

func (s *server) respondWithError(w http.ResponseWriter, code int, err models.ErrDetails) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(code)
	resp := models.ErrorResponse{
		Error: err,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		zap.L().Error("failed to encode JSON for response", zap.Error(err))
	}
}

func (s *server) respondWithJSON(w http.ResponseWriter, code int, resp interface{}) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		zap.L().Error("failed to encode JSON for response", zap.Error(err))
	}
}

func (s *server) AddTeamHandler(w http.ResponseWriter, r *http.Request) {
	body := r.Body
	defer body.Close()

	var addTeamRequest models.AddTeamRequest
	err := json.NewDecoder(body).Decode(&addTeamRequest)
	if err != nil {
		err := models.ErrDetails{
			Code:    models.InvalidJSONErr,
			Message: fmt.Sprintf("failed to decode json: %v", err),
		}
		s.respondWithError(w, http.StatusBadRequest, err)
		return
	}

	team, serviceErr := s.service.AddTeam(r.Context(), addTeamRequest)
	if serviceErr != nil {
		s.respondWithError(w, s.mapServiceErrors(serviceErr.Code), *serviceErr)
		return
	}

	resp := models.AddTeamResponse{
		Team: *team,
	}
	s.respondWithJSON(w, http.StatusCreated, resp)
}

func (s *server) GetTeamHandler(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		err := models.ErrDetails{
			Code:    models.NotFoundErr,
			Message: "resourse not found",
		}
		s.respondWithError(w, http.StatusBadRequest, err)
		return
	}

	teamResp, serviceErr := s.service.GetTeam(r.Context(), teamName)
	if serviceErr != nil {
		s.respondWithError(w, s.mapServiceErrors(serviceErr.Code), *serviceErr)
		return
	}

	s.respondWithJSON(w, http.StatusOK, *teamResp)
}

func (s *server) SetUserStatusHandler(w http.ResponseWriter, r *http.Request) {
	body := r.Body
	defer body.Close()

	var request models.SetUserStatusRequest
	if err := json.NewDecoder(body).Decode(&request); err != nil {
		err := models.ErrDetails{
			Code:    models.InvalidJSONErr,
			Message: fmt.Sprintf("failed to decode json: %v", err),
		}
		s.respondWithError(w, http.StatusBadRequest, err)
		return
	}

	user, serviceErr := s.service.SetUserStatus(r.Context(), request)
	if serviceErr != nil {
		s.respondWithError(w, s.mapServiceErrors(serviceErr.Code), *serviceErr)
		return
	}

	resp := models.SetUserStatusResponse{
		User: user,
	}
	s.respondWithJSON(w, http.StatusOK, resp)
}

func (s *server) GetUserReviewsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		err := models.ErrDetails{
			Code:    models.NotFoundErr,
			Message: "resourse not found",
		}
		s.respondWithError(w, http.StatusBadRequest, err)
		return
	}

	pullRequests, serviceErr := s.service.GetUserReviews(r.Context(), userID)
	if serviceErr != nil {
		s.respondWithError(w, s.mapServiceErrors(serviceErr.Code), *serviceErr)
		return
	}

	resp := models.GetUserReviewsResponse{
		UserID:       userID,
		PullRequests: pullRequests,
	}
	s.respondWithJSON(w, http.StatusOK, resp)
}

func (s *server) CreatePullRequestHandler(w http.ResponseWriter, r *http.Request) {
	body := r.Body
	defer body.Close()

	var createPRRequest models.CreatePRRequest
	if err := json.NewDecoder(body).Decode(&createPRRequest); err != nil {
		err := models.ErrDetails{
			Code:    models.InvalidJSONErr,
			Message: fmt.Sprintf("failed to decode json: %v", err),
		}
		s.respondWithError(w, http.StatusBadRequest, err)
		return
	}

	pullRequest, serviceErr := s.service.CreatePullRequest(r.Context(), createPRRequest)
	if serviceErr != nil {
		s.respondWithError(w, s.mapServiceErrors(serviceErr.Code), *serviceErr)
		return
	}

	resp := models.CreatePRResponse{
		PullRequest: *pullRequest,
	}
	s.respondWithJSON(w, http.StatusCreated, resp)
}

func (s *server) MergePullRequestHandler(w http.ResponseWriter, r *http.Request) {
	body := r.Body
	defer body.Close()

	var request models.MergePRRequest
	if err := json.NewDecoder(body).Decode(&request); err != nil {
		err := models.ErrDetails{
			Code:    models.InvalidJSONErr,
			Message: fmt.Sprintf("failed to decode json: %v", err),
		}
		s.respondWithError(w, http.StatusBadRequest, err)
		return
	}

	pullRequest, serviceErr := s.service.MergePullRequest(r.Context(), request.ID)
	if serviceErr != nil {
		s.respondWithError(w, s.mapServiceErrors(serviceErr.Code), *serviceErr)
		return
	}

	resp := models.MergePullRequestResponse{
		PullRequest: pullRequest,
	}
	s.respondWithJSON(w, http.StatusOK, resp)
}

func (s *server) ReassignPullRequestReviewerHandler(w http.ResponseWriter, r *http.Request) {
	body := r.Body
	defer body.Close()

	var request models.ReassignPRReviewerRequest
	if err := json.NewDecoder(body).Decode(&request); err != nil {
		err := models.ErrDetails{
			Code:    models.InvalidJSONErr,
			Message: fmt.Sprintf("failed to decode json: %v", err),
		}
		s.respondWithError(w, http.StatusBadRequest, err)
		return
	}

	pullRequest, replacedBy, serviceErr := s.service.ReassignPullRequestReviewer(r.Context(), request)
	if serviceErr != nil {
		s.respondWithError(w, s.mapServiceErrors(serviceErr.Code), *serviceErr)
		return
	}

	resp := models.ReassignPullRequestReviewerResponse{
		PullRequest: pullRequest,
		ReplacedBy:  replacedBy,
	}
	s.respondWithJSON(w, http.StatusOK, resp)
}
