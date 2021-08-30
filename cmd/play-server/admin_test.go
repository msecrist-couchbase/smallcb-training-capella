package main

import(
  "testing"
  "net/http"
  "net/http/httptest"
)

func TestHttpHandleHealth(t *testing.T) {
  req := httptest.NewRequest("GET", "/admin/health", nil)
  rr := httptest.NewRecorder()

  handler := http.HandlerFunc(HttpHandleHealth)
  handler.ServeHTTP(rr, req)

  if rr.Code != http.StatusOK {
    t.Errorf("Healthcheck reported wrong initial status");
  }

  rr = httptest.NewRecorder()
  statsNums["http.Run"] = 1
  handler.ServeHTTP(rr, req)
  if rr.Code != http.StatusOK {
    t.Errorf("Service is healthy but the healthcheck is failing")
  }

  rr = httptest.NewRecorder()
  statsNums["http.Run.err"] = 1
  handler.ServeHTTP(rr, req)
  if rr.Code != http.StatusInternalServerError {
    t.Errorf("Service is not healthy but the healthcheck is ok")
  }
}
