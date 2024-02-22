// promql provides an abstraction on the prometheus HTTP API
package promql

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/longxiucai/patrol-tools/pkg/common"
	"github.com/longxiucai/patrol-tools/pkg/result"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
) // Client is our prometheus v1 API interface
type Client interface {
	v1.API
}

// Cfg conatins the final configuration params parsed from a combo of flags, config file values, and env vars.
type PromQL struct {
	Host            string `mapstructure:"host" yaml:"host"`
	Step            string `mapstructure:"step" yaml:"step"`
	Output          string `mapstructure:"output" yaml:"output"`
	OutputPath      string `mapstructure:"output-path" yaml:"output-path"`
	TimeoutDuration time.Duration
	CfgFile         string    `mapstructure:"cfg-file" yaml:"cfg-file"`
	Time            time.Time `mapstructure:"time" yaml:"time"`
	Start           string    `mapstructure:"start" yaml:"start"`
	End             string    `mapstructure:"end" yaml:"end"`
	NoHeaders       bool      `mapstructure:"no-headers" yaml:"no-headers"`
	Auth            config.Authorization
	Client          v1.API
	TLSConfig       config.TLSConfig
	Rules           []common.Rule `mapstructure:"rules" yaml:"rules"`
	KubeConfigPath  string        `mapstructure:"kubeconfig" yaml:"kubeconfig"`
}

// CreateClientWithAuth creates a Client interface witht the provided hostname and auth config
func CreateClientWithAuth(host string, authCfg config.Authorization, tlsCfg config.TLSConfig) (v1.API, error) {
	cfg := api.Config{
		Address: host,
	}
	tc, err := config.NewTLSConfig(&tlsCfg)
	if err != nil {
		return nil, fmt.Errorf("error parsing TLS config, %s", err)
	}
	var rt http.RoundTripper = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     tc,
	}

	if authCfg != (config.Authorization{}) {
		switch {
		case authCfg.Type == "":
			return nil, fmt.Errorf("please specify an authentication type, run promql --help for more details")
		case authCfg.Credentials != "" && authCfg.CredentialsFile != "":
			return nil, fmt.Errorf("please specify either auth credentials or an auth credential file, not both")
		case authCfg.Credentials != "":
			rt = config.NewAuthorizationCredentialsRoundTripper(authCfg.Type, config.Secret(authCfg.Credentials), rt)
		default:
			rt = config.NewAuthorizationCredentialsFileRoundTripper(authCfg.Type, authCfg.CredentialsFile, rt)
		}
	}
	cfg.RoundTripper = rt
	a, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return v1.NewAPI(a), nil
}

// InstantQuery performs an instant query and returns the result
func (p *PromQL) Run() (result.ResultList, v1.Warnings, error) {
	var results result.ResultList
	if p.Start != "" {
		for _, rule := range p.Rules {
			matrixResult, warnings, err := p.rangeQuery(rule.Expr)
			if len(warnings) > 0 {
				return nil, warnings, nil
			}
			if err != nil {
				return nil, nil, err
			}
			result := result.Result{
				Rule:       rule,
				PromResult: matrixResult,
			}
			results = append(results, result)
		}
	} else {
		for _, rule := range p.Rules {
			vectorResult, warnings, err := p.instantQuery(rule.Expr)
			if len(warnings) > 0 {
				return nil, warnings, nil
			}
			if err != nil {
				return nil, nil, err
			}
			result := result.Result{
				Rule:       rule,
				PromResult: vectorResult,
			}
			results = append(results, result)
		}
	}
	return results, nil, nil
}

// InstantQuery performs an instant query and returns the result
func (p *PromQL) instantQuery(queryString string) (model.Vector, v1.Warnings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.TimeoutDuration)
	defer cancel()

	result, warnings, err := p.Client.Query(ctx, queryString, p.Time)
	if err != nil {
		return nil, warnings, fmt.Errorf("error querying prometheus: %v", err)
	}

	if result, ok := result.(model.Vector); ok {
		return result, warnings, nil
	} else {
		return nil, warnings, fmt.Errorf("did not receive an instant vector result")
	}
}

func parseRangeStart(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	} else if d, err := time.ParseDuration(s); err == nil {
		return time.Now().Add(-d), nil
	} else {
		err = fmt.Errorf("unable to parse range start time, %v", err)
		return time.Time{}, err
	}
}

func parseRangeEnd(e string) (time.Time, error) {
	if e == "now" {
		return time.Now(), nil
	}
	t, err := time.Parse(time.RFC3339, e)
	if err != nil {
		return time.Time{}, fmt.Errorf("error parsing range end time, %v", err)
	}
	return t, nil
}

// getRange creates a prometheus range from the provided start, end, and step options
func (p *PromQL) getRange() (r v1.Range, err error) {
	// At minimum we need a start time so we attempt to parse that first
	r.Start, err = parseRangeStart(p.Start)
	if err != nil {
		return r, err
	}
	// Set up defaults for the step value
	r.Step = time.Minute
	// If the user provided a step value, parse it as a time.Duration and override the default
	if p.Step != "" {
		r.Step, err = time.ParseDuration(p.Step)
		if err != nil {
			err = fmt.Errorf("unable to parse step duration, %v", err)
			return r, err
		}
	}

	// If the user provided an end value, parse it to a time struct and override the default
	r.End, err = parseRangeEnd(p.End)
	if err != nil {
		return r, err
	}

	return r, err
}

// rangeQuery performs a range query and writes the results to stdout
func (p *PromQL) rangeQuery(queryString string) (model.Matrix, v1.Warnings, error) {
	// create context with a timeout,
	ctx, cancel := context.WithTimeout(context.Background(), p.TimeoutDuration)
	defer cancel()

	r, err := p.getRange()
	if err != nil {
		return nil, nil, err
	}
	// execute query
	result, warnings, err := p.Client.QueryRange(ctx, queryString, r)
	if err != nil {
		return nil, warnings, err
	}

	if result, ok := result.(model.Matrix); ok {
		return result, warnings, err
	} else {
		return nil, warnings, fmt.Errorf("did not receive a range result")
	}
}

// LabelsQuery runs a labels query and returns the result
func (p *PromQL) LabelsQuery(query string) (model.Vector, v1.Warnings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.TimeoutDuration)
	defer cancel()

	result, warnings, err := p.Client.Query(ctx, query, time.Now())
	if err != nil {
		return nil, warnings, err
	}

	// if result is the expected type, Write it out in the
	// desired output format
	if result, ok := result.(model.Vector); ok {
		return result, warnings, err
	} else {
		return nil, warnings, fmt.Errorf("did not recieve an instant vector")
	}
}

// MetaQuery returns prometheus metrics metadata. Used for our metrics and meta commands
func (p *PromQL) MetaQuery(query string) (map[string][]v1.Metadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.TimeoutDuration)
	defer cancel()

	result, err := p.Client.Metadata(ctx, query, "")
	if err != nil {
		return map[string][]v1.Metadata{}, fmt.Errorf("error querying metadata endpoint: %v", err)
	}
	return result, nil
}

// SeriesQuery returns prometheus series data
func (p *PromQL) SeriesQuery(query string) ([]model.LabelSet, v1.Warnings, error) {
	// Set defaults based on time flag
	s := p.Time.Add(-15 * time.Second)
	e := p.Time
	// Parse range start and end if provided and override the defaults
	if p.Start != "" {
		var err error
		s, err = parseRangeStart(p.Start)
		if err != nil {
			return []model.LabelSet{}, v1.Warnings{}, err
		}
	}
	if p.End != "" {
		var err error
		e, err = parseRangeEnd(p.End)
		if err != nil {
			return []model.LabelSet{}, v1.Warnings{}, err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), p.TimeoutDuration)
	defer cancel()
	result, warnings, err := p.Client.Series(ctx, []string{query}, s, e)
	if err != nil {
		return []model.LabelSet{}, warnings, fmt.Errorf("error querying series endpoint: %v", err)
	}
	return result, warnings, err
}
