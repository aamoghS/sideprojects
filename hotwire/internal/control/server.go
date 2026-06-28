package control

import (
	"context"
	"io"
	"sync"
	"time"

	hotwirev1 "github.com/aamoghS/sideprojects/hotwire/proto/hotwire/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	hotwirev1.UnimplementedControlPlaneServer

	registry          *Registry
	rebalanceInterval time.Duration

	mu            sync.RWMutex
	subscribers   map[int]chan *hotwirev1.WeightUpdate
	metricStreams map[string]chan *hotwirev1.WeightAssignment
	nextSubID     int

	lastUpdate fingerprint
}

type fingerprint struct {
	reason  string
	weights map[string]float64
}

func NewServer(rebalanceInterval time.Duration) *Server {
	if rebalanceInterval <= 0 {
		rebalanceInterval = 500 * time.Millisecond
	}
	s := &Server{
		registry:          NewRegistry(DefaultStaleThreshold),
		rebalanceInterval: rebalanceInterval,
		subscribers:       make(map[int]chan *hotwirev1.WeightUpdate),
		metricStreams:     make(map[string]chan *hotwirev1.WeightAssignment),
	}
	return s
}

func (s *Server) Registry() *Registry {
	return s.registry
}

func (s *Server) RunRebalanceLoop(ctx context.Context) {
	ticker := time.NewTicker(s.rebalanceInterval)
	defer ticker.Stop()
	s.rebalance(time.Now())
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.rebalance(now)
		}
	}
}

func (s *Server) ReportMetrics(stream hotwirev1.ControlPlane_ReportMetricsServer) error {
	ctx := stream.Context()
	backendID := ""
	pushCh := make(chan *hotwirev1.WeightAssignment, 8)

	defer func() {
		s.mu.Lock()
		if backendID != "" {
			delete(s.metricStreams, backendID)
			s.registry.Remove(backendID)
		}
		s.mu.Unlock()
	}()

	for {
		report, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if report.GetBackendId() == "" {
			return status.Error(codes.InvalidArgument, "backend_id required")
		}

		if backendID == "" {
			backendID = report.GetBackendId()
			s.mu.Lock()
			s.metricStreams[backendID] = pushCh
			s.mu.Unlock()
		} else if report.GetBackendId() != backendID {
			return status.Error(codes.InvalidArgument, "backend_id mismatch on stream")
		}

		s.registry.Update(AgentMetrics{
			BackendID:  report.GetBackendId(),
			P50Ms:      report.GetP50Ms(),
			P99Ms:      report.GetP99Ms(),
			ErrorRate:  report.GetErrorRate(),
			Inflight:   report.GetInflight(),
			RPS:        report.GetRps(),
			LastReport: time.Now(),
		})

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		now := time.Now()
		result := Score(s.registry, now)
		for _, b := range result.Backends {
			if b.BackendID != backendID {
				continue
			}
			if err := stream.Send(&hotwirev1.WeightAssignment{
				BackendId:       b.BackendID,
				Weight:          b.Weight,
				TimestampUnixMs: now.UnixMilli(),
			}); err != nil {
				return err
			}
			break
		}

		select {
		case assignment := <-pushCh:
			if err := stream.Send(assignment); err != nil {
				return err
			}
		default:
		}
	}
}

func (s *Server) SubscribeWeights(req *hotwirev1.SubscribeRequest, stream hotwirev1.ControlPlane_SubscribeWeightsServer) error {
	ch := make(chan *hotwirev1.WeightUpdate, 4)
	id := s.addSubscriber(ch)
	defer s.removeSubscriber(id)

	if update := s.currentWeightUpdate(time.Now()); update != nil {
		if err := stream.Send(update); err != nil {
			return err
		}
	}

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case update := <-ch:
			if err := stream.Send(update); err != nil {
				return err
			}
		}
	}
}

func (s *Server) ListBackends(context.Context, *hotwirev1.ListBackendsRequest) (*hotwirev1.ListBackendsResponse, error) {
	return s.listBackends(time.Now()), nil
}

func (s *Server) listBackends(now time.Time) *hotwirev1.ListBackendsResponse {
	result := Score(s.registry, now)
	backends := make([]*hotwirev1.BackendState, 0, len(result.Backends))
	for _, b := range result.Backends {
		backends = append(backends, &hotwirev1.BackendState{
			BackendId:         b.BackendID,
			Weight:            b.Weight,
			Score:             b.Score,
			P99Ms:             b.P99Ms,
			P50Ms:             b.P50Ms,
			ErrorRate:         b.ErrorRate,
			Inflight:          b.Inflight,
			Rps:               b.RPS,
			LastReportUnixMs:  b.LastReport.UnixMilli(),
			Stale:             b.Stale,
		})
	}
	return &hotwirev1.ListBackendsResponse{Backends: backends}
}

func (s *Server) rebalance(now time.Time) {
	result := Score(s.registry, now)
	fp := fingerprintFrom(result)
	s.mu.RLock()
	changed := !fp.equal(s.lastUpdate)
	s.mu.RUnlock()
	if !changed {
		return
	}

	update := scoreResultToUpdate(result, now)
	assignments := scoreResultToAssignments(result, now)

	s.mu.Lock()
	s.lastUpdate = fp
	for id, ch := range s.subscribers {
		select {
		case ch <- update:
		default:
			delete(s.subscribers, id)
		}
	}
	for backendID, ch := range s.metricStreams {
		for _, a := range assignments {
			if a.BackendId == backendID {
				select {
				case ch <- a:
				default:
				}
				break
			}
		}
	}
	s.mu.Unlock()
}

func (s *Server) currentWeightUpdate(now time.Time) *hotwirev1.WeightUpdate {
	result := Score(s.registry, now)
	if len(result.Backends) == 0 {
		return nil
	}
	return scoreResultToUpdate(result, now)
}

func (s *Server) addSubscriber(ch chan *hotwirev1.WeightUpdate) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextSubID++
	id := s.nextSubID
	s.subscribers[id] = ch
	return id
}

func (s *Server) removeSubscriber(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.subscribers, id)
}

func scoreResultToUpdate(result ScoreResult, now time.Time) *hotwirev1.WeightUpdate {
	weights := make([]*hotwirev1.BackendWeight, 0, len(result.Backends))
	for _, b := range result.Backends {
		weights = append(weights, &hotwirev1.BackendWeight{
			BackendId: b.BackendID,
			Weight:    b.Weight,
			Score:     b.Score,
			P99Ms:     b.P99Ms,
			ErrorRate: b.ErrorRate,
		})
	}
	return &hotwirev1.WeightUpdate{
		Weights:         weights,
		Reason:          result.Reason,
		TimestampUnixMs: now.UnixMilli(),
	}
}

func scoreResultToAssignments(result ScoreResult, now time.Time) []*hotwirev1.WeightAssignment {
	out := make([]*hotwirev1.WeightAssignment, 0, len(result.Backends))
	ts := now.UnixMilli()
	for _, b := range result.Backends {
		out = append(out, &hotwirev1.WeightAssignment{
			BackendId:       b.BackendID,
			Weight:          b.Weight,
			TimestampUnixMs: ts,
		})
	}
	return out
}

func fingerprintFrom(result ScoreResult) fingerprint {
	weights := make(map[string]float64, len(result.Backends))
	for _, b := range result.Backends {
		weights[b.BackendID] = b.Weight
	}
	return fingerprint{reason: result.Reason, weights: weights}
}

func (f fingerprint) equal(o fingerprint) bool {
	if f.reason != o.reason || len(f.weights) != len(o.weights) {
		return false
	}
	for k, v := range f.weights {
		if o.weights[k] != v {
			return false
		}
	}
	return true
}
