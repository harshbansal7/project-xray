package clickhouse

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/xray-sdk/xray-api/internal/models"
)

// TraceModel maps to xray_traces table
type TraceModel struct {
	TraceID    string     `gorm:"primaryKey;type:String"`
	PipelineID string     `gorm:"type:LowCardinality(String)"`
	StartedAt  time.Time  `gorm:"type:DateTime64(6)"`
	EndedAt    *time.Time `gorm:"type:Nullable(DateTime64(6))"`
	Status     string     `gorm:"type:LowCardinality(String)"`
	Metadata   JSONMap    `gorm:"type:String"`
	InputData  JSONAny    `gorm:"type:String"`
	Tags       []string   `gorm:"type:Array(String)"`
	CreatedAt  time.Time  `gorm:"type:DateTime64(6);default:now64(6)"`
}

func (TraceModel) TableName() string {
	return "xray_traces"
}

// EventModel maps to xray_events table
type EventModel struct {
	EventID        string     `gorm:"primaryKey;type:String"`
	TraceID        string     `gorm:"type:String;index:idx_events_trace_id"`
	ParentEventID  *string    `gorm:"type:Nullable(String)"`
	StepName       string     `gorm:"type:LowCardinality(String)"`
	StepType       string     `gorm:"type:LowCardinality(String)"`
	CaptureMode    string     `gorm:"type:LowCardinality(String);default:'metrics'"`
	InputCount     *int       `gorm:"type:Nullable(Int32)"`
	InputSample    JSONArray  `gorm:"type:String;default:'[]'"`
	OutputCount    *int       `gorm:"type:Nullable(Int32)"`
	OutputSample   JSONArray  `gorm:"type:String;default:'[]'"`
	Metrics        JSONMap    `gorm:"type:String;default:'{}'"`
	Annotations    JSONMap    `gorm:"type:String;default:'{}'"`
	PipelineID     string     `gorm:"type:LowCardinality(String)"`
	StartedAt      time.Time  `gorm:"type:DateTime64(6)"`
	EndedAt        *time.Time `gorm:"type:Nullable(DateTime64(6))"`
	ReductionRatio *float64   `gorm:"type:Nullable(Float32)"`
	CreatedAt      time.Time  `gorm:"type:DateTime64(6);default:now64(6)"`
}

func (EventModel) TableName() string {
	return "xray_events"
}

// DecisionModel maps to xray_decisions table
type DecisionModel struct {
	DecisionID   string     `gorm:"primaryKey;type:String"`
	EventID      string     `gorm:"type:String"`
	TraceID      string     `gorm:"type:String;index:idx_decisions_trace_id"`
	ItemID       string     `gorm:"type:String;index:idx_decisions_item_id"`
	Outcome      string     `gorm:"type:LowCardinality(String)"`
	ReasonCode   *string    `gorm:"type:Nullable(String)"`
	ReasonDetail *string    `gorm:"type:Nullable(String)"`
	Scores       JSONScores `gorm:"type:String;default:'{}'"`
	ItemSnapshot JSONMap    `gorm:"type:String;default:'{}'"`
	Timestamp    time.Time  `gorm:"type:DateTime64(6)"`
	CreatedAt    time.Time  `gorm:"type:DateTime64(6);default:now64(6)"`
}

func (DecisionModel) TableName() string {
	return "xray_decisions"
}

// JSONMap is a helper type for handling map[string]interface{} in GORM
type JSONMap map[string]interface{}

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return "{}", nil
	}
	bytes, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return string(bytes), nil
}

func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONMap)
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return errors.New("failed to unmarshal JSONB value")
	}

	if len(bytes) == 0 {
		*j = make(JSONMap)
		return nil
	}

	return json.Unmarshal(bytes, j)
}

// JSONArray is a helper type for handling []interface{} in GORM
type JSONArray []interface{}

func (j JSONArray) Value() (driver.Value, error) {
	if j == nil {
		return "[]", nil
	}
	bytes, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return string(bytes), nil
}

func (j *JSONArray) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONArray, 0)
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return errors.New("failed to unmarshal JSONArray value")
	}

	if len(bytes) == 0 {
		*j = make(JSONArray, 0)
		return nil
	}

	return json.Unmarshal(bytes, j)
}

// JSONAny is a helper type for handling interface{} in GORM
type JSONAny struct {
	Data interface{}
}

func (j JSONAny) Value() (driver.Value, error) {
	if j.Data == nil {
		return "{}", nil // Default to empty object if nil, or "null"
	}
	bytes, err := json.Marshal(j.Data)
	if err != nil {
		return nil, err
	}
	return string(bytes), nil
}

func (j *JSONAny) Scan(value interface{}) error {
	if value == nil {
		j.Data = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return errors.New("failed to unmarshal JSONAny value")
	}

	if len(bytes) == 0 {
		j.Data = nil
		return nil
	}

	return json.Unmarshal(bytes, &j.Data)
}

// JSONScores is a helper type for handling map[string]float64 in GORM
type JSONScores map[string]float64

func (j JSONScores) Value() (driver.Value, error) {
	if j == nil {
		return "{}", nil
	}
	bytes, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return string(bytes), nil
}

func (j *JSONScores) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONScores)
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return errors.New("failed to unmarshal JSONScores value")
	}

	if len(bytes) == 0 {
		*j = make(JSONScores)
		return nil
	}

	return json.Unmarshal(bytes, j)
}

// Helper converters
func timeToPtr(t time.Time) *time.Time {
	return &t
}

func (m *TraceModel) ToDomain() *models.Trace {
	return &models.Trace{
		TraceID:    m.TraceID,
		PipelineID: m.PipelineID,
		StartedAt:  m.StartedAt,
		EndedAt:    m.EndedAt,
		Status:     m.Status,
		Metadata:   map[string]interface{}(m.Metadata),
		InputData:  m.InputData.Data,
		Tags:       m.Tags,
	}
}

func (m *EventModel) ToDomain() *models.Event {
	return &models.Event{
		EventID:       m.EventID,
		TraceID:       m.TraceID,
		ParentEventID: m.ParentEventID,
		StepName:      m.StepName,
		StepType:      models.StepType(m.StepType),
		CaptureMode:   models.CaptureMode(m.CaptureMode),
		InputCount:    m.InputCount,
		InputSample:   []interface{}(m.InputSample),
		OutputCount:   m.OutputCount,
		OutputSample:  []interface{}(m.OutputSample),
		Metrics:       map[string]interface{}(m.Metrics),
		Annotations:   map[string]interface{}(m.Annotations),
		// PipelineID is omitted as it's not on the domain model
		StartedAt: m.StartedAt,
		EndedAt:   m.EndedAt,
	}
}

func (m *DecisionModel) ToDomain() *models.Decision {
	return &models.Decision{
		DecisionID:   m.DecisionID,
		EventID:      m.EventID,
		TraceID:      m.TraceID,
		ItemID:       m.ItemID,
		Outcome:      m.Outcome,
		ReasonCode:   m.ReasonCode,
		ReasonDetail: m.ReasonDetail,
		Scores:       map[string]float64(m.Scores),
		ItemSnapshot: map[string]interface{}(m.ItemSnapshot),
		Timestamp:    m.Timestamp,
	}
}
