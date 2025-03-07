// Copyright OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package signalfxreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/signalfxreceiver"

import (
	sfxpb "github.com/signalfx/com_signalfx_metrics_protobuf/model"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/splunk"
)

// signalFxV2ToMetricsData converts SignalFx event proto data points to
// plog.LogRecordSlice. Returning the converted data and the number of dropped log
// records.
func signalFxV2EventsToLogRecords(events []*sfxpb.Event, lrs plog.LogRecordSlice) {
	lrs.EnsureCapacity(len(events))

	for _, event := range events {
		lr := lrs.AppendEmpty()

		attrs := lr.Attributes()
		attrs.Clear()
		attrs.EnsureCapacity(2 + len(event.Dimensions) + len(event.Properties))

		// The EventType field is stored as an attribute.
		eventType := event.EventType
		if eventType == "" {
			eventType = "unknown"
		}
		attrs.InsertString(splunk.SFxEventType, eventType)

		// SignalFx timestamps are in millis so convert to nanos by multiplying
		// by 1 million.
		lr.SetTimestamp(pcommon.Timestamp(event.Timestamp * 1e6))

		if event.Category != nil {
			attrs.InsertInt(splunk.SFxEventCategoryKey, int64(*event.Category))
		} else {
			// This gives us an unambiguous way of determining that a log record
			// represents a SignalFx event, even if category is missing from the
			// event.
			attrs.Insert(splunk.SFxEventCategoryKey, pcommon.NewValueEmpty())
		}

		for _, dim := range event.Dimensions {
			attrs.InsertString(dim.Key, dim.Value)
		}

		if len(event.Properties) > 0 {
			propMapVal := pcommon.NewValueMap()
			propMap := propMapVal.MapVal()
			propMap.EnsureCapacity(len(event.Properties))

			for _, prop := range event.Properties {
				// No way to tell what value type is without testing each
				// individually.
				switch {
				case prop.Value.StrValue != nil:
					propMap.InsertString(prop.Key, prop.Value.GetStrValue())
				case prop.Value.IntValue != nil:
					propMap.InsertInt(prop.Key, prop.Value.GetIntValue())
				case prop.Value.DoubleValue != nil:
					propMap.InsertDouble(prop.Key, prop.Value.GetDoubleValue())
				case prop.Value.BoolValue != nil:
					propMap.InsertBool(prop.Key, prop.Value.GetBoolValue())
				default:
					// If there is no property value, just insert a null to
					// record that the key was present.
					propMap.Insert(prop.Key, pcommon.NewValueEmpty())
				}
			}

			attrs.Insert(splunk.SFxEventPropertiesKey, propMapVal)
		}
	}
}
