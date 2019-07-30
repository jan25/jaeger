package zipkin

import (
	zmodel "github.com/jaegertracing/jaeger/proto-gen/zipkin"
	"github.com/jaegertracing/jaeger/thrift-gen/zipkincore"
	"github.com/openzipkin/zipkin-go/model"
)

// Converts Zipkin Protobuf spans to Thrift model
func protoSpansV2ToThrift(listOfSpans *zmodel.ListOfSpans) ([]*zipkincore.Span, error) {
	tSpans := make([]*zipkincore.Span, 0, len(listOfSpans.Spans))
	for _, span := range listOfSpans.Spans {
		tSpan, err := protoSpanV2ToThrift(span)
		if err != nil {
			return nil, err
		}
		tSpans = append(tSpans, tSpan)
	}
	return tSpans, nil
}

func protoSpanV2ToThrift(s *zmodel.Span) (*zipkincore.Span, error) {
	id, err := model.SpanIDFromString(cutLongID(*s.ID))
	if err != nil {
		return nil, err
	}
	traceID, err := model.TraceIDFromString(*s.TraceID)
	if err != nil {
		return nil, err
	}
	tSpan := &zipkincore.Span{
		ID:        int64(id),
		TraceID:   int64(traceID.Low),
		Name:      s.Name,
		Debug:     s.Debug,
		Timestamp: &s.Timestamp,
		Duration:  &s.Duration,
	}
	if traceID.High != 0 {
		help := int64(traceID.High)
		tSpan.TraceIDHigh = &help
	}

	if len(s.ParentID) > 0 {
		parentID, err := model.SpanIDFromString(cutLongID(s.ParentID))
		if err != nil {
			return nil, err
		}
		signed := int64(parentID)
		tSpan.ParentID = &signed
	}

	var localE *zipkincore.Endpoint
	if s.LocalEndpoint != nil {
		localE, err = endpointV2ToThrift(s.LocalEndpoint)
		if err != nil {
			return nil, err
		}
	}

	for _, a := range s.Annotations {
		tA := annoV2ToThrift(a, localE)
		tSpan.Annotations = append(tSpan.Annotations, tA)
	}

	tSpan.BinaryAnnotations = append(tSpan.BinaryAnnotations, tagsToThrift(s.Tags, localE)...)
	tSpan.Annotations = append(tSpan.Annotations, kindToThrift(s.Timestamp, s.Duration, s.Kind, localE)...)

	if s.RemoteEndpoint != nil {
		rAddrAnno, err := remoteEndpToThrift(s.RemoteEndpoint, s.Kind)
		if err != nil {
			return nil, err
		}
		tSpan.BinaryAnnotations = append(tSpan.BinaryAnnotations, rAddrAnno)
	}

	// add local component to represent service name
	// to_domain looks for a service name in all [bin]annotations
	if localE != nil && len(tSpan.BinaryAnnotations) == 0 && len(tSpan.Annotations) == 0 {
		tSpan.BinaryAnnotations = append(tSpan.BinaryAnnotations, &zipkincore.BinaryAnnotation{
			Key:            zipkincore.LOCAL_COMPONENT,
			Host:           localE,
			AnnotationType: zipkincore.AnnotationType_STRING,
		})
	}
	return tSpan, nil
}
