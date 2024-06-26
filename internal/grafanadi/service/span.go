package service

import (
	"time"

	"github.com/alipay/container-observability-service/internal/grafanadi/model"
	storagemodel "github.com/alipay/container-observability-service/pkg/dal/storage-client/model"
	"github.com/google/uuid"
)

func ConverSpan2Frame(spans []*storagemodel.Span) model.DataFrame {
	rootBegin := time.Now()
	rootEnd := time.Unix(0, 0)
	if len(spans) == 0 {
		return model.DataFrame{}

	}

	minBegin := time.Now()
	maxEnd := time.Unix(0, 0)
	for _, e := range spans {
		if e.Begin.Before(rootBegin) && e.Begin.After(time.Unix(0, 0)) {
			rootBegin = e.Begin
		}

		if e.End.After(rootEnd) {
			rootEnd = e.End
		}
		if e.ActionType == "PodCreate" {
			if e.Begin.Before(minBegin) && e.Begin.After(time.Unix(0, 0)) {
				minBegin = e.Begin
			}
			if e.End.After(maxEnd) {
				maxEnd = e.End
			}
		}
	}
	rootTraceID := uuid.New().String()
	rootSpanID := uuid.New().String()

	traceAry := []string{rootTraceID}
	spanAry := []string{rootSpanID}
	parentSpanAry := []interface{}{nil}
	opsAry := []string{"PodDelivery"}
	serviceAry := []string{"PodDelivery"}
	serviceTagAry := []interface{}{nil}
	startTimeAry := []int64{rootBegin.UnixNano() / 1e6}
	var durationAry []int64
	if rootEnd == time.Unix(0, 0) {
		durationAry = []int64{0}
	} else {
		durationAry = []int64{int64(rootEnd.Sub(rootBegin) / 1000 / 1000)}
	}

	var opsType string
	deduplicateMap := make(map[string]string, 0)
	for _, sp := range spans {
		if sp.Begin.Before(rootBegin) {
			continue
		}
		traceAry = append(traceAry, rootTraceID)
		spanId := uuid.New().String()
		spanAry = append(spanAry, spanId)

		opsType = sp.Type
		if sp.Type != sp.Name {
			opsType = sp.Type + ":" + sp.Name
		}
		opsAry = append(opsAry, opsType)
		serviceAry = append(serviceAry, sp.ActionType)
		serviceTagAry = append(serviceTagAry, nil)
		startTimeAry = append(startTimeAry, sp.Begin.UnixNano()/1e6)
		elapsedDur := sp.Elapsed
		durationAry = append(durationAry, elapsedDur)

		value, ok := deduplicateMap[sp.ActionType]
		if !ok {
			deduplicateMap[sp.ActionType] = spanId
			parentSpanAry = append(parentSpanAry, rootSpanID)
			if opsType != "pod_ready_span" && sp.ActionType == "PodCreate" {
				opsAry[1] = ""
				durationAry[1] = int64(maxEnd.Sub(minBegin) / 1000 / 1000)
				startTimeAry[1] = minBegin.UnixNano() / 1e6
				traceAry = append(traceAry, rootTraceID)
				spanAry = append(spanAry, spanId)

				parentSpanAry = append(parentSpanAry, spanId)
				opsAry = append(opsAry, opsType)
				serviceAry = append(serviceAry, sp.ActionType)
				serviceTagAry = append(serviceTagAry, nil)
				startTimeAry = append(startTimeAry, sp.Begin.UnixNano()/1e6)
				durationAry = append(durationAry, elapsedDur)
			}
		} else {
			parentSpanAry = append(parentSpanAry, value)
		}
	}

	return model.DataFrame{
		Schema: model.SchemaType{
			Fields: []model.FieldType{
				{Name: "traceID", Type: "string"},
				{Name: "spanID", Type: "string"},
				{Name: "parentSpanID", Type: "string"},
				{Name: "operationName", Type: "string"},
				{Name: "serviceName", Type: "string"},
				{Name: "serviceTags", Type: "string"},
				{Name: "startTime", Type: "time"},
				{Name: "duration", Type: "number"},
			},
		},
		Data: model.DataType{
			Values: []interface{}{
				traceAry, spanAry, parentSpanAry, opsAry, serviceAry, serviceTagAry, startTimeAry, durationAry,
			},
		},
	}
}
