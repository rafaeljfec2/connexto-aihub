package gateway

import (
	"net/http"
	"time"

	"aihub/internal/records"
)

// TrafficBucketDTO represents a single bucket in the 24-hour time series.
type TrafficBucketDTO struct {
	Timestamp string `json:"timestamp"`
	Label     string `json:"label"`
	Count     int    `json:"count"`
}

// TrafficSummaryDTO represents the 24-hour request volume summary.
type TrafficSummaryDTO struct {
	Range                string             `json:"range"`
	TotalRequests        int                `json:"total_requests"`
	TrendPercentage      float64            `json:"trend_percentage"`
	Buckets              []TrafficBucketDTO `json:"buckets"`
	HasSufficientHistory bool               `json:"has_sufficient_history"`
}

const trafficSummaryRange = "24h"

// NewTrafficSummaryHandler computes traffic telemetry strictly from execution records.
func NewTrafficSummaryHandler(service *records.Service, nowFunc func() time.Time) http.Handler {
	if nowFunc == nil {
		nowFunc = time.Now
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")

		rangeParam := r.URL.Query().Get("range")
		if rangeParam == "" {
			rangeParam = trafficSummaryRange
		}
		if rangeParam != trafficSummaryRange {
			writeJSONError(w, http.StatusBadRequest, "invalid_request", "unsupported traffic summary range")
			return
		}

		if service == nil || service.Degraded() {
			writeJSON(w, http.StatusOK, TrafficSummaryDTO{
				Range:                trafficSummaryRange,
				TotalRequests:        0,
				TrendPercentage:      0,
				Buckets:              []TrafficBucketDTO{},
				HasSufficientHistory: false,
			})
			return
		}

		now := nowFunc().UTC()
		since := now.Add(-24 * time.Hour)

		allRecords, err := service.ListAll(records.Filter{
			StartedAfter: &since,
			Limit:        records.MaxPageLimit,
		})
		if err != nil {
			writeJSON(w, http.StatusOK, TrafficSummaryDTO{
				Range:                trafficSummaryRange,
				TotalRequests:        0,
				TrendPercentage:      0,
				Buckets:              []TrafficBucketDTO{},
				HasSufficientHistory: false,
			})
			return
		}

		total := len(allRecords)
		if total == 0 {
			writeJSON(w, http.StatusOK, TrafficSummaryDTO{
				Range:                trafficSummaryRange,
				TotalRequests:        0,
				TrendPercentage:      0,
				Buckets:              []TrafficBucketDTO{},
				HasSufficientHistory: false,
			})
			return
		}

		bucketLabels := []string{"00:00", "06:00", "12:00", "18:00", "Agora"}
		buckets := make([]TrafficBucketDTO, len(bucketLabels))
		bucketDuration := 24 * time.Hour / time.Duration(len(bucketLabels))

		for i, label := range bucketLabels {
			bucketStart := since.Add(time.Duration(i) * bucketDuration)
			buckets[i] = TrafficBucketDTO{
				Timestamp: bucketStart.Format(time.RFC3339),
				Label:     label,
				Count:     0,
			}
		}

		midpoint := since.Add(12 * time.Hour)
		firstHalf := 0
		secondHalf := 0

		for _, rec := range allRecords {
			diff := rec.StartedAt.Sub(since)
			if diff < 0 {
				continue
			}
			idx := int(diff / bucketDuration)
			if idx >= len(buckets) {
				idx = len(buckets) - 1
			}
			if idx >= 0 {
				buckets[idx].Count++
			}

			if rec.StartedAt.Before(midpoint) {
				firstHalf++
			} else {
				secondHalf++
			}
		}

		trend := 0.0
		if firstHalf > 0 {
			trend = (float64(secondHalf-firstHalf) / float64(firstHalf)) * 100
		}

		writeJSON(w, http.StatusOK, TrafficSummaryDTO{
			Range:                trafficSummaryRange,
			TotalRequests:        total,
			TrendPercentage:      trend,
			Buckets:              buckets,
			HasSufficientHistory: total >= 5,
		})
	})
}
