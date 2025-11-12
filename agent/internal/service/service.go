package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hackathon-agent/internal/model"
	"hackathon-agent/pkg/kafka"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/text/encoding/charmap"
)

const (
	workerCount       = 5
	messagePipeBuffer = 1000
)

type Service struct {
	log          *zap.Logger
	consumer     kafka.ConsumerGroupRunner
	producer     kafka.Producer
	produceTopic string
}

func NewService(log *zap.Logger, consumer kafka.ConsumerGroupRunner, producer kafka.Producer, produceTopic string) *Service {
	return &Service{
		log:          log,
		consumer:     consumer,
		producer:     producer,
		produceTopic: produceTopic,
	}
}

func (s *Service) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		s.consumer.Run()
	}()

	messagePipe := make(chan *kafka.MessageWithMarkFunc, messagePipeBuffer)
	for i := 0; i < workerCount; i++ {
		go s.worker(ctx, i, messagePipe)
	}

	for {
		select {
		case <-ctx.Done():
			s.log.Info("context canceled, stopping Run")

			close(messagePipe)

			return nil
		case msg, ok := <-s.consumer.Messages():
			if !ok {
				s.log.Info("consumer messages channel closed")
				close(messagePipe)
				return nil
			}

			messagePipe <- msg
		}
	}
}

func (s *Service) Stop() error {
	if err := s.consumer.Shutdown(); err != nil {
		return fmt.Errorf("failed to close subscriber consumer: %w", err)
	}

	return nil
}

func (s *Service) worker(ctx context.Context, id int, messagePipe <-chan *kafka.MessageWithMarkFunc) {
	s.log.Info("Worker started", zap.Int("id", id))

	for {
		select {
		case <-ctx.Done():
			s.log.Info("Worker stopping", zap.Int("id", id))

			return

		case msg, ok := <-messagePipe:
			if !ok {
				s.log.Info("Message channel closed", zap.Int("id", id))

				return
			}

			messageID, err := uuid.FromBytes(msg.Message.Key)
			if err != nil {
				s.log.Error("Error parsing message id", zap.Int("workerID", id), zap.Error(err))

				continue
			}

			if err := s.process(msg); err != nil {
				s.log.Error("Processing failed",
					zap.Int("workerID", id),
					zap.String("messageID", messageID.String()),
					zap.Error(err),
				)
			}

			msg.Mark()
		}
	}
}

func (s *Service) process(message *kafka.MessageWithMarkFunc) error {
	task, err := ParseTask(message.Message.Value)
	if err != nil {
		s.log.Error("Error parsing task", zap.String("task", string(message.Message.Key)), zap.Error(err))
		return err
	}

	return s.RunCheck(context.Background(), task)
}

func ParseTask(data []byte) (model.TaskMessage, error) {
	var t model.TaskMessage
	if err := json.Unmarshal(data, &t); err != nil {
		return t, fmt.Errorf("bad task json: %w", err)
	}
	for i := range t.Checks {
		if err := normalizeCheckParams(&t.Checks[i]); err != nil {
			return t, fmt.Errorf("check %d (%s): %w", i, t.Checks[i].Type, err)
		}
	}
	return t, nil
}

func normalizeCheckParams(c *model.CheckRequest) error {
	switch strings.ToLower(c.Type) {
	case "http":
		// expectedStatusRange: строка "[200,299]" или массив
		if v, ok := c.Params["expectedStatusRange"]; ok {
			switch vv := v.(type) {
			case string:
				r, err := parseRange2Int(vv)
				if err != nil {
					return fmt.Errorf("expectedStatusRange: %w", err)
				}
				c.Params["expectedStatusRange"] = [2]int{r[0], r[1]}
			case []interface{}:
				if len(vv) != 2 {
					return errors.New("expectedStatusRange must have 2 elements")
				}
				c.Params["expectedStatusRange"] = [2]int{toInt(vv[0]), toInt(vv[1])}
			}
		} else {
			c.Params["expectedStatusRange"] = [2]int{200, 299}
		}
		// headers: "" -> убрать; строка с JSON -> разобрать
		if v, ok := c.Params["headers"]; ok {
			if s, ok := v.(string); ok {
				str := strings.TrimSpace(s)
				if str == "" {
					delete(c.Params, "headers")
				} else if strings.HasPrefix(str, "{") {
					var m map[string]string
					if json.Unmarshal([]byte(str), &m) == nil {
						c.Params["headers"] = m
					} else {
						delete(c.Params, "headers")
					}
				} else {
					delete(c.Params, "headers")
				}
			}
		}
		var hp model.HTTPParams
		if err := decodeLoose(c.Params, &hp); err != nil {
			return err
		}
		if hp.Scheme == "" {
			hp.Scheme = "https"
		}
		if hp.Path == "" {
			hp.Path = "/"
		}
		if hp.ExpectedStatusRange == ([2]int{}) {
			hp.ExpectedStatusRange = [2]int{200, 299}
		}
		c.Params = mustToMap(hp)

	case "ping":
		var pp model.PingParams
		if err := decodeLoose(c.Params, &pp); err != nil {
			return err
		}
		if pp.Count <= 0 {
			pp.Count = 4
		}
		if pp.IntervalMs <= 0 {
			pp.IntervalMs = 1000
		}
		c.Params = mustToMap(pp)

	case "tcp":
		var tp model.TCPParams
		if err := decodeLoose(c.Params, &tp); err != nil {
			return err
		}
		if tp.ConnectTimeoutMs <= 0 {
			tp.ConnectTimeoutMs = 3000
		}
		c.Params = mustToMap(tp)

	case "traceroute":
		var tp model.TracerouteParams
		if err := decodeLoose(c.Params, &tp); err != nil {
			return err
		}
		if tp.MaxHops <= 0 {
			tp.MaxHops = 30
		}
		if tp.Mode == "" {
			tp.Mode = "udp"
		}
		c.Params = mustToMap(tp)

	case "dns":
		if v, ok := c.Params["records"]; ok {
			if s, ok := v.(string); ok {
				var arr []string
				if json.Unmarshal([]byte(s), &arr) == nil {
					c.Params["records"] = arr
				}
			}
		}
		var dp model.DNSParams
		if err := decodeLoose(c.Params, &dp); err != nil {
			return err
		}
		if len(dp.Records) == 0 {
			dp.Records = []string{"A"}
		}
		c.Params = mustToMap(dp)

	default:
		return fmt.Errorf("unsupported check type %q", c.Type)
	}
	return nil
}

func decodeLoose(m map[string]interface{}, out any) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

func mustToMap(v any) map[string]interface{} {
	b, _ := json.Marshal(v)
	var m map[string]interface{}
	_ = json.Unmarshal(b, &m)
	return m
}

func parseRange2Int(s string) ([2]int, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return [2]int{}, errors.New("range must be like [a,b]")
	}
	a, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return [2]int{}, err
	}
	b, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return [2]int{}, err
	}
	return [2]int{a, b}, nil
}

func toInt(v interface{}) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case json.Number:
		i, _ := t.Int64()
		return int(i)
	default:
		return 0
	}
}

func (s *Service) RunCheck(ctx context.Context, task model.TaskMessage) error {
	// общий дедлайн на весь таск
	if task.TimeoutSeconds <= 0 {
		task.TimeoutSeconds = 20
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(task.TimeoutSeconds)*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(len(task.Checks))

	for i, chk := range task.Checks {
		i, chk := i, chk
		go func() {
			defer wg.Done()

			// индивидуальный таймаут, не длиннее общего
			per := perCheckTimeout(chk.Type)
			perCtx, perCancel := context.WithTimeout(ctx, per)
			defer perCancel()

			// если общий уже умер — сразу выходим
			select {
			case <-ctx.Done():
				res := makeResTemplate(task.ID, i, chk.Type, task.Target, time.Now(), false, ctx.Err(), nil)
				s.publish(perCtx, res)

				return

			default:
			}

			res := s.runOne(perCtx, task, i, chk)
			s.publish(perCtx, res)
		}()
	}

	wg.Wait()
	return nil
}

func (s *Service) publish(ctx context.Context, res model.CheckResult) {
	b, err := json.Marshal(res)
	if err != nil {
		s.log.Error("Failed to marshal message", zap.Error(err), zap.String("taskID", res.TaskID.String()))
	}

	taskID, err := res.TaskID.MarshalBinary()
	if err != nil {
		s.log.Error("Failed to marshal taskID", zap.Error(err), zap.String("taskID", res.TaskID.String()))
	}

	partition, offset, err := s.producer.PushMessage(ctx, taskID, b, s.produceTopic)
	if err != nil {
		s.log.Error("Failed to push message", zap.Error(err), zap.String("taskID", res.TaskID.String()))
	}

	s.log.Info("Message sent",
		zap.String("taskID", res.TaskID.String()),
		zap.Int32("partition", partition),
		zap.Int64("offset", offset),
	)
}

func perCheckTimeout(typ string) time.Duration {
	switch strings.ToLower(typ) {
	case "http":
		return 5 * time.Second
	case "tcp":
		return 3 * time.Second
	case "ping":
		return 6 * time.Second
	case "dns":
		return 4 * time.Second
	case "traceroute":
		return 16 * time.Second
	default:
		return 5 * time.Second
	}
}

func (s *Service) runOne(ctx context.Context, task model.TaskMessage, idx int, chk model.CheckRequest) model.CheckResult {
	start := time.Now()

	makeRes := func(ok bool, err error, payload any) model.CheckResult {
		return makeResTemplate(task.ID, idx, chk.Type, task.Target, start, ok, err, payload)
	}

	switch strings.ToLower(chk.Type) {
	case "http":
		var p model.HTTPParams
		_ = decodeLoose(chk.Params, &p)
		return runHTTP(ctx, task.Target, p, start, makeRes)
	case "ping":
		var p model.PingParams
		_ = decodeLoose(chk.Params, &p)
		return runPing(ctx, task.Target, p, start, makeRes)
	case "tcp":
		var p model.TCPParams
		_ = decodeLoose(chk.Params, &p)
		return runTCP(ctx, task.Target, p, start, makeRes)
	case "traceroute":
		var p model.TracerouteParams
		_ = decodeLoose(chk.Params, &p)
		return runTraceroute(ctx, task.Target, p, start, makeRes)
	case "dns":
		var p model.DNSParams
		_ = decodeLoose(chk.Params, &p)
		return runDNS(ctx, task.Target, p, start, makeRes)
	default:
		return makeRes(false, fmt.Errorf("unsupported check type %q", chk.Type), nil)
	}
}

func makeResTemplate(taskID uuid.UUID, idx int, typ, target string, start time.Time, ok bool, err error, payload any) model.CheckResult {
	var raw json.RawMessage
	if payload != nil {
		if b, e := json.Marshal(payload); e == nil {
			raw = b
		}
	}
	res := model.CheckResult{
		TaskID:     taskID,
		CheckIndex: idx,
		Type:       strings.ToLower(typ),
		Target:     target,
		StartedAt:  start.UTC(),
		DurationMs: time.Since(start).Milliseconds(),
		OK:         ok,
		Payload:    raw,
	}
	if err != nil {
		res.Error = err.Error()
	}
	return res
}

func runHTTP(ctx context.Context, target string, p model.HTTPParams, start time.Time,
	makeRes func(bool, error, any) model.CheckResult,
) model.CheckResult {
	url := fmt.Sprintf("%s://%s%s", nonEmpty(p.Scheme, "https"), target, nonEmpty(p.Path, "/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return makeRes(false, err, nil)
	}
	for k, v := range p.Headers {
		req.Header.Set(k, v)
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 3 * time.Second}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		IdleConnTimeout:       10 * time.Second,
	}
	client := &http.Client{Transport: transport}
	if !p.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	t0 := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(t0)
	if err != nil {
		return makeRes(false, err, map[string]any{"url": url})
	}
	defer resp.Body.Close()

	ok := resp.StatusCode >= p.ExpectedStatusRange[0] && resp.StatusCode <= p.ExpectedStatusRange[1]
	payload := map[string]any{
		"url":        url,
		"status":     resp.StatusCode,
		"latencyMs":  latency.Milliseconds(),
		"finalURL":   resp.Request.URL.String(),
		"limitBytes": p.MaxBodyBytes,
	}
	return makeRes(ok, nil, payload)
}

func runTCP(ctx context.Context, target string, p model.TCPParams, _ time.Time,
	makeRes func(bool, error, any) model.CheckResult,
) model.CheckResult {
	addr := net.JoinHostPort(target, strconv.Itoa(p.Port))
	d := net.Dialer{Timeout: time.Duration(p.ConnectTimeoutMs) * time.Millisecond}
	t0 := time.Now()
	conn, err := d.DialContext(ctx, "tcp", addr)
	lat := time.Since(t0)
	if err != nil {
		return makeRes(false, err, map[string]any{"addr": addr})
	}
	_ = conn.Close()
	return makeRes(true, nil, map[string]any{
		"addr":      addr,
		"handshake": lat.Milliseconds(),
	})
}

func runDNS(ctx context.Context, target string, p model.DNSParams, _ time.Time,
	makeRes func(bool, error, any) model.CheckResult,
) model.CheckResult {
	r := newResolver(p.Resolver, 2*time.Second)
	results := map[string]any{}
	var haveError bool

	for _, rr := range p.Records {
		switch strings.ToUpper(rr) {
		case "A":
			ips, err := r.LookupHost(ctx, target)
			if err != nil {
				results["A_error"] = err.Error()
				haveError = true
			} else {
				var a []string
				for _, ip := range ips {
					if parsed := net.ParseIP(ip); parsed != nil && parsed.To4() != nil {
						a = append(a, ip)
					}
				}
				results["A"] = a
			}
		case "AAAA":
			ips, err := r.LookupHost(ctx, target)
			if err != nil {
				results["AAAA_error"] = err.Error()
				haveError = true
			} else {
				var aaaa []string
				for _, ip := range ips {
					if parsed := net.ParseIP(ip); parsed != nil && parsed.To4() == nil {
						aaaa = append(aaaa, ip)
					}
				}
				results["AAAA"] = aaaa
			}
		case "MX":
			mx, err := r.LookupMX(ctx, target)
			if err != nil {
				results["MX_error"] = err.Error()
				haveError = true
			} else {
				type m struct {
					Host string `json:"host"`
					Pref uint16 `json:"pref"`
				}
				out := make([]m, 0, len(mx))
				for _, rec := range mx {
					out = append(out, m{Host: rec.Host, Pref: rec.Pref})
				}
				results["MX"] = out
			}
		default:
			results[strings.ToUpper(rr)+"_error"] = "unsupported record type"
			haveError = true
		}
	}

	return makeRes(!haveError, nil, results)
}

func newResolver(addr string, timeout time.Duration) *net.Resolver {
	if strings.TrimSpace(addr) == "" {
		return &net.Resolver{}
	}
	a := net.JoinHostPort(addr, "53")
	d := &net.Dialer{Timeout: timeout}
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return d.DialContext(ctx, "udp", a)
		},
	}
}

func runPing(ctx context.Context, target string, p model.PingParams, _ time.Time,
	makeRes func(bool, error, any) model.CheckResult,
) model.CheckResult {
	cmdName := "ping"
	args := []string{}
	if runtime.GOOS == "windows" {
		args = []string{"-n", strconv.Itoa(p.Count), target}
	} else {
		iv := fmt.Sprintf("%.3f", float64(p.IntervalMs)/1000.0)
		args = []string{"-c", strconv.Itoa(p.Count), "-i", iv, target}
	}
	cmd := exec.CommandContext(ctx, cmdName, args...)
	out, err := cmd.CombinedOutput()
	output := decodeConsole(out)

	if err != nil {
		return makeRes(false, err, map[string]any{
			"command":  cmd.String(),
			"output":   tail(output, 4096),
			"exitCode": exitCode(err),
		})
	}
	return makeRes(true, nil, map[string]any{
		"command":  cmd.String(),
		"output":   tail(output, 4096),
		"exitCode": 0,
	})
}

type Hop struct {
	IP  string   `json:"ip"`
	Lat *float64 `json:"lat,omitempty"`
	Lon *float64 `json:"lon,omitempty"`
}

type GeoIPResolver interface {
	Resolve(ctx context.Context, ip string) (float64, float64, error)
}

type httpGeoIP struct {
	client *http.Client
	// эндпоинт должен возвращать {"status":"success","lat":..,"lon":..}
	// по умолчанию используем ip-api.com (без ключа, не злоупотребляй)
}

func NewHTTPGeoIP(timeout time.Duration) GeoIPResolver {
	return &httpGeoIP{
		client: &http.Client{Timeout: timeout},
	}
}

func (r *httpGeoIP) Resolve(ctx context.Context, ip string) (float64, float64, error) {
	// ip-api.com/json/{ip}?fields=status,lat,lon
	url := "http://ip-api.com/json/" + ip + "?fields=status,lat,lon"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, 0, err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()
	var x struct {
		Status string  `json:"status"`
		Lat    float64 `json:"lat"`
		Lon    float64 `json:"lon"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&x); err != nil {
		return 0, 0, err
	}
	if x.Status != "success" {
		return 0, 0, errors.New("geoip: not found")
	}
	return x.Lat, x.Lon, nil
}

type MemoryGeoCache struct {
	inner GeoIPResolver
	ttl   time.Duration

	mu   sync.RWMutex
	mapt map[string]geoEntry
}
type geoEntry struct {
	lat float64
	lon float64
	exp time.Time
}

func NewMemoryGeoCache(inner GeoIPResolver, ttl time.Duration) *MemoryGeoCache {
	return &MemoryGeoCache{
		inner: inner,
		ttl:   ttl,
		mapt:  make(map[string]geoEntry, 1024),
	}
}

var ipRe = regexp.MustCompile(`\b(\d{1,3}(?:\.\d{1,3}){3})\b`)

func (c *MemoryGeoCache) Resolve(ctx context.Context, ip string) (float64, float64, error) {
	now := time.Now()
	c.mu.RLock()
	if e, ok := c.mapt[ip]; ok && now.Before(e.exp) {
		c.mu.RUnlock()
		return e.lat, e.lon, nil
	}
	c.mu.RUnlock()

	lat, lon, err := c.inner.Resolve(ctx, ip)
	if err != nil {
		return 0, 0, err
	}
	c.mu.Lock()
	c.mapt[ip] = geoEntry{lat: lat, lon: lon, exp: now.Add(c.ttl)}
	c.mu.Unlock()
	return lat, lon, nil
}

func runTraceroute(
	ctx context.Context,
	target string,
	p model.TracerouteParams,
	_ time.Time,
	makeRes func(bool, error, any) model.CheckResult,
) model.CheckResult {
	start := time.Now()

	cmdName, args := buildTracerouteArgs(target, p)
	cmd := exec.CommandContext(ctx, cmdName, args...)
	out, err := cmd.CombinedOutput()
	output := decodeConsole(out)

	ips := parseTraceIPs(output)

	geo := NewMemoryGeoCache(NewHTTPGeoIP(2*time.Second), 1*time.Hour)
	hops := make([]Hop, 0, len(ips))
	for _, ip := range ips {
		if isPrivateOrReserved(ip) {
			// оставим без координат
			hops = append(hops, Hop{IP: ip})
			continue
		}
		lat, lon, gerr := geo.Resolve(ctx, ip)
		if gerr != nil {
			hops = append(hops, Hop{IP: ip})
			continue
		}
		hops = append(hops, Hop{IP: ip, Lat: &lat, Lon: &lon})
	}

	payload := map[string]any{
		"command":  cmd.String(),
		"output":   tail(output, 8192),
		"exitCode": exitCode(err),
		"hops":     hops,
	}
	ok := err == nil
	res := makeRes(ok, err, payload)
	res.DurationMs = time.Since(start).Milliseconds()
	return res
}

func buildTracerouteArgs(target string, p model.TracerouteParams) (string, []string) {
	if p.MaxHops <= 0 {
		p.MaxHops = 30
	}
	if runtime.GOOS == "windows" {
		// tracert / d, все флаги ДО цели, а не как у тебя
		args := []string{"-d", "-h", strconv.Itoa(p.MaxHops), "-w", "1000", target}
		return "tracert", args
	}
	args := []string{"-n", "-m", strconv.Itoa(p.MaxHops)}
	switch strings.ToLower(p.Mode) {
	case "tcp":
		args = append(args, "-T")
		if p.Port > 0 {
			args = append(args, "-p", strconv.Itoa(p.Port))
		}
	case "icmp":
		args = append(args, "-I")
	}
	args = append(args, target)
	return "traceroute", args
}

func nonEmpty(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

func parseTraceIPs(output string) []string {
	lines := strings.Split(output, "\n")
	seen := make(map[string]struct{}, 64)
	var ips []string
	for _, ln := range lines {
		// пропускаем строки где только звездочки
		if strings.Count(ln, "*") >= 3 && !ipRe.MatchString(ln) {
			continue
		}
		m := ipRe.FindAllString(ln, -1)
		for _, ip := range m {
			if !validIPv4(ip) {
				continue
			}
			if _, ok := seen[ip]; ok {
				continue
			}
			seen[ip] = struct{}{}
			ips = append(ips, ip)
		}
	}
	return ips
}

func validIPv4(ip string) bool {
	parsed := net.ParseIP(ip)
	return parsed != nil && parsed.To4() != nil
}

func isPrivateOrReserved(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return true
	}
	ip4 := parsed.To4()
	if ip4 == nil {
		return true
	}
	// 10.0.0.0/8
	if ip4[0] == 10 {
		return true
	}
	// 172.16.0.0/12
	if ip4[0] == 172 && ip4[1]&0xf0 == 16 {
		return true
	}
	// 192.168.0.0/16
	if ip4[0] == 192 && ip4[1] == 168 {
		return true
	}
	// Carrier-grade NAT 100.64.0.0/10
	if ip4[0] == 100 && (ip4[1]&0xc0) == 64 {
		return true
	}
	// loopback 127.0.0.0/8
	if ip4[0] == 127 {
		return true
	}
	// link-local 169.254.0.0/16
	if ip4[0] == 169 && ip4[1] == 254 {
		return true
	}
	return false
}

func decodeConsole(out []byte) string {
	if runtime.GOOS == "windows" {
		if s, err := charmap.CodePage866.NewDecoder().String(string(out)); err == nil {
			return s
		}
		if s, err := charmap.Windows1251.NewDecoder().String(string(out)); err == nil {
			return s
		}
	}
	return string(out)
}

func tail(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[len(s)-max:]
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode()
	}
	return -1
}
