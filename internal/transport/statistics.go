package transport

import (
	"net/http"
)

func (s *server) GetUsersStatisticsHandler(w http.ResponseWriter, r *http.Request) {
	userStatsResp, errDetails := s.service.GetUsersStatistics(r.Context())
	if errDetails != nil {
		s.respondWithError(w, s.mapServiceErrors(errDetails.Code), *errDetails)
		return
	}

	s.respondWithJSON(w, http.StatusOK, userStatsResp)
}

func (s *server) GetPullRequestStatisticsHandler(w http.ResponseWriter, r *http.Request) {
	prStatsResp, errDetails := s.service.GetPullRequestStatistics(r.Context())
	if errDetails != nil {
		s.respondWithError(w, s.mapServiceErrors(errDetails.Code), *errDetails)
		return
	}

	s.respondWithJSON(w, http.StatusOK, prStatsResp)
}

func (s *server) GetReviewersStatisticHandler(w http.ResponseWriter, r *http.Request) {
	reviewersStatsResp, errDetails := s.service.GetReviewersStatistics(r.Context())
	if errDetails != nil {
		s.respondWithError(w, s.mapServiceErrors(errDetails.Code), *errDetails)
		return
	}

	s.respondWithJSON(w, http.StatusOK, reviewersStatsResp)
}
