// Copyright 2020 OpenTelemetry Authors
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

package metricstransformprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/metricstransformprocessor"

import (
	"context"
	"fmt"
	"strconv"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// extractAndRemoveMatchedMetrics extracts matched metrics from ms metric slice and returns a new slice.
// Extracted metrics can have reduced number of data point if not all of them match the filter.
// All matched metrics, including metrics with only a subset of matched data points,
// are removed from the original ms metric slice.
func extractAndRemoveMatchedMetrics(f internalFilter, ms pmetric.MetricSlice) pmetric.MetricSlice {
	extractedMetrics := pmetric.NewMetricSlice()
	ms.RemoveIf(func(metric pmetric.Metric) bool {
		if extractedMetric := f.extractMatchedMetric(metric); extractedMetric != (pmetric.Metric{}) {
			extractedMetric.MoveTo(extractedMetrics.AppendEmpty())
			return true
		}
		return false
	})
	return extractedMetrics
}

// matchMetrics returns a slice of metrics matching the filter f. Original metrics slice is not affected.
func matchMetrics(f internalFilter, metrics pmetric.MetricSlice) []pmetric.Metric {
	mm := make([]pmetric.Metric, 0, metrics.Len())
	for i := 0; i < metrics.Len(); i++ {
		if f.matchMetric(metrics.At(i)) {
			mm = append(mm, metrics.At(i))
		}
	}
	return mm
}

// extractMatchedMetric returns a metric matching the filter.
// If provided metric matches the filter with all its data points, the original metric returned as is.
// If only part of data points match the filter, a new metric is returned with data points matching the filter.
// Otherwise, an invalid metric is returned.
func (f internalFilterStrict) extractMatchedMetric(metric pmetric.Metric) pmetric.Metric {
	if metric.Name() == f.include {
		return extractMetricWithMatchingAttrs(metric, f)
	}
	return pmetric.Metric{}
}

func (f internalFilterStrict) matchMetric(metric pmetric.Metric) bool {
	if metric.Name() == f.include {
		return matchAnyDps(metric, f)
	}
	return false
}

func (f internalFilterStrict) submatches(_ pmetric.Metric) []int {
	return nil
}

func (f internalFilterStrict) expand(_, _ string) string {
	return ""
}

func (f internalFilterStrict) matchAttrs(attrs pcommon.Map) bool {
	return matchAttrs(f.attrMatchers, attrs)
}

// extractMatchedMetric returns a metric matching the filter.
// If provided metric matches the filter with all its data points, the original metric returned as is.
// If only part of data points match the filter, a new metric is returned with data points matching the filter.
// Otherwise, an invalid metric is returned.
func (f internalFilterRegexp) extractMatchedMetric(metric pmetric.Metric) pmetric.Metric {
	if submatches := f.include.FindStringSubmatchIndex(metric.Name()); submatches != nil {
		return extractMetricWithMatchingAttrs(metric, f)
	}
	return pmetric.Metric{}
}

func (f internalFilterRegexp) matchMetric(metric pmetric.Metric) bool {
	if submatches := f.include.FindStringSubmatchIndex(metric.Name()); submatches != nil {
		return matchAnyDps(metric, f)
	}
	return false
}

func (f internalFilterRegexp) submatches(metric pmetric.Metric) []int {
	return f.include.FindStringSubmatchIndex(metric.Name())
}

func (f internalFilterRegexp) expand(metricTempate, metricName string) string {
	if submatches := f.include.FindStringSubmatchIndex(metricName); submatches != nil {
		return string(f.include.ExpandString([]byte{}, metricTempate, metricName, submatches))
	}
	return ""
}

func (f internalFilterRegexp) matchAttrs(attrs pcommon.Map) bool {
	return matchAttrs(f.attrMatchers, attrs)
}

// matchAnyDps checks whether any metric data points match the filter, returns true if metric has no data points.
func matchAnyDps(metric pmetric.Metric, f internalFilter) bool {
	match := true
	rangeDataPointAttributes(metric, func(attrs pcommon.Map) bool {
		if f.matchAttrs(attrs) {
			match = true
			return false
		}
		match = false
		return true
	})
	return match
}

// matchAllDps checks whether all metric data points match the filter, returns true if metric has no data points.
func matchAllDps(metric pmetric.Metric, f internalFilter) bool {
	match := true
	rangeDataPointAttributes(metric, func(attrs pcommon.Map) bool {
		if !f.matchAttrs(attrs) {
			match = false
			return false
		}
		return true
	})
	return match
}

// matchDps returns a slice of bool values representing data points matches following by the total number of matched
// data points.
// For example, for a metric with 3 data points where only first and third match the filter, the output will be:
// ([]bool{true, false, true}, 2).
func matchDps(metric pmetric.Metric, f internalFilter) (matchedDps []bool, matchedDpsCount int) {
	matchedDps = []bool{}
	rangeDataPointAttributes(metric, func(attrs pcommon.Map) bool {
		match := f.matchAttrs(attrs)
		if match {
			matchedDpsCount++
		}
		matchedDps = append(matchedDps, match)
		return true
	})
	return
}

// extractMetricWithMatchingAttrs returns a metric with data points matching attrMatchers.
// New metric is returned if part of data points match the filter,
// original metric returned if all data points match the filter,
// and invalid metric returned if no data points match the filter.
func extractMetricWithMatchingAttrs(metric pmetric.Metric, f internalFilter) pmetric.Metric {
	dpsMatches, matchedDpsCount := matchDps(metric, f)
	if matchedDpsCount == len(dpsMatches) {
		return metric
	}
	if matchedDpsCount == 0 {
		return pmetric.Metric{}
	}

	newMetric := pmetric.NewMetric()
	newMetric.SetDataType(metric.DataType())
	newMetric.SetName(metric.Name())
	newMetric.SetDescription(metric.Description())
	newMetric.SetUnit(metric.Unit())

	switch metric.DataType() {
	case pmetric.MetricDataTypeGauge:
		newMetric.Gauge().DataPoints().EnsureCapacity(matchedDpsCount)
		for i := 0; i < metric.Gauge().DataPoints().Len(); i++ {
			if dpsMatches[i] {
				metric.Gauge().DataPoints().At(i).CopyTo(newMetric.Gauge().DataPoints().AppendEmpty())
			}
		}
	case pmetric.MetricDataTypeSum:
		newMetric.Sum().DataPoints().EnsureCapacity(matchedDpsCount)
		for i := 0; i < metric.Sum().DataPoints().Len(); i++ {
			if dpsMatches[i] {
				metric.Sum().DataPoints().At(i).CopyTo(newMetric.Sum().DataPoints().AppendEmpty())
			}
		}
		newMetric.Sum().SetAggregationTemporality(metric.Sum().AggregationTemporality())
		newMetric.Sum().SetIsMonotonic(metric.Sum().IsMonotonic())
	case pmetric.MetricDataTypeHistogram:
		newMetric.Histogram().DataPoints().EnsureCapacity(matchedDpsCount)
		for i := 0; i < metric.Histogram().DataPoints().Len(); i++ {
			if dpsMatches[i] {
				metric.Histogram().DataPoints().At(i).CopyTo(newMetric.Histogram().DataPoints().AppendEmpty())
			}
		}
		newMetric.Histogram().SetAggregationTemporality(metric.Histogram().AggregationTemporality())
	case pmetric.MetricDataTypeExponentialHistogram:
		newMetric.ExponentialHistogram().DataPoints().EnsureCapacity(matchedDpsCount)
		for i := 0; i < metric.ExponentialHistogram().DataPoints().Len(); i++ {
			if dpsMatches[i] {
				metric.ExponentialHistogram().DataPoints().At(i).CopyTo(newMetric.ExponentialHistogram().DataPoints().AppendEmpty())
			}
		}
		newMetric.ExponentialHistogram().SetAggregationTemporality(metric.ExponentialHistogram().AggregationTemporality())
	case pmetric.MetricDataTypeSummary:
		newMetric.Summary().DataPoints().EnsureCapacity(matchedDpsCount)
		for i := 0; i < metric.Summary().DataPoints().Len(); i++ {
			if dpsMatches[i] {
				metric.Summary().DataPoints().At(i).CopyTo(newMetric.Summary().DataPoints().AppendEmpty())
			}
		}
	}

	return newMetric
}

func matchAttrs(attrMatchers map[string]StringMatcher, attrs pcommon.Map) bool {
	for k, v := range attrMatchers {
		attrVal, ok := attrs.Get(k)
		// attribute values doesn't match, drop datapoint
		if ok && !v.MatchString(attrVal.StringVal()) {
			return false
		}

		// if a label-key is not found then return nil only if the given label-value is non-empty. If a given label-value is empty
		// and the key is not found then move forward. In this approach we can make sure certain key is not present which is a valid use case.
		if !ok && !v.MatchString("") {
			return false
		}
	}
	return true
}

func (mtp *metricsTransformProcessor) processMetrics(_ context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	rms := md.ResourceMetrics()
	groupedRMs := pmetric.NewResourceMetricsSlice()

	rms.RemoveIf(func(rm pmetric.ResourceMetrics) bool {
		rm.ScopeMetrics().RemoveIf(func(sm pmetric.ScopeMetrics) bool {
			metrics := sm.Metrics()

			for _, transform := range mtp.transforms {
				switch transform.Action {
				case Group:
					extractedMetrics := extractAndRemoveMatchedMetrics(transform.MetricIncludeFilter, metrics)
					groupMatchedMetrics(rm.Resource(), sm.Scope(), extractedMetrics,
						transform).CopyTo(groupedRMs.AppendEmpty())
				case Combine:
					matchedMetrics := matchMetrics(transform.MetricIncludeFilter, metrics)
					if len(matchedMetrics) == 0 {
						continue
					}

					if err := canBeCombined(matchedMetrics); err != nil {
						// TODO: report via trace / metric instead
						mtp.logger.Warn(err.Error())
						continue
					}

					extractedMetrics := extractAndRemoveMatchedMetrics(transform.MetricIncludeFilter, metrics)
					combinedMetric := combine(transform, extractedMetrics)
					if transformMetric(combinedMetric, transform) {
						combinedMetric.MoveTo(metrics.AppendEmpty())
					}
				case Insert:
					newMetrics := pmetric.NewMetricSlice()
					newMetrics.EnsureCapacity(metrics.Len())
					for i := 0; i < metrics.Len(); i++ {
						metric := metrics.At(i)
						newMetric := transform.MetricIncludeFilter.extractMatchedMetric(metric)
						if newMetric == (pmetric.Metric{}) {
							continue
						}
						if newMetric == metric {
							newMetric = pmetric.NewMetric()
							metric.CopyTo(newMetric)
						}
						if transformMetric(newMetric, transform) {
							newMetric.MoveTo(newMetrics.AppendEmpty())
						}
					}
					newMetrics.MoveAndAppendTo(metrics)
				case Update:
					metrics.RemoveIf(func(metric pmetric.Metric) bool {
						if !transform.MetricIncludeFilter.matchMetric(metric) {
							return false
						}

						// Drop the metric if all the data points were dropped after transformations.
						return !transformMetric(metric, transform)
					})
				}
			}

			return metrics.Len() == 0
		})

		return rm.ScopeMetrics().Len() == 0
	})

	groupedRMs.MoveAndAppendTo(rms)

	return md, nil
}

// groupMatchedMetrics groups matched metrics by moving them from matchedMetrics into a new pmetric.ResourceMetrics.
func groupMatchedMetrics(resource pcommon.Resource, scope pcommon.InstrumentationScope, metrics pmetric.MetricSlice,
	transform internalTransform) pmetric.ResourceMetrics {
	rm := pmetric.NewResourceMetrics()
	resource.CopyTo(rm.Resource())

	for k, v := range transform.GroupResourceLabels {
		rm.Resource().Attributes().UpsertString(k, v)
	}

	sm := rm.ScopeMetrics().AppendEmpty()
	scope.CopyTo(sm.Scope())
	metrics.MoveAndAppendTo(sm.Metrics())
	return rm
}

// canBeCombined returns true if all the provided metrics share the same type, unit, and labels
func canBeCombined(metrics []pmetric.Metric) error {
	if len(metrics) <= 1 {
		return nil
	}

	var firstMetric pmetric.Metric
	for _, metric := range metrics {
		if metric.DataType() == pmetric.MetricDataTypeSummary {
			return fmt.Errorf("Summary metrics cannot be combined: %v ", metric.Name())
		}

		if firstMetric == (pmetric.Metric{}) {
			firstMetric = metric
			continue
		}

		if firstMetric.DataType() != metric.DataType() {
			return fmt.Errorf("metrics cannot be combined as they are of different types: %v (%v) and %v (%v)",
				firstMetric.Name(), firstMetric.DataType(), metric.Name(), metric.DataType())
		}
		if firstMetric.Unit() != metric.Unit() {
			return fmt.Errorf("metrics cannot be combined as they have different units: %v (%v) and %v (%v)",
				firstMetric.Name(), firstMetric.Unit(), metric.Name(), metric.Unit())
		}

		firstMetricAttrKeys := metricAttributeKeys(firstMetric)
		metricAttrKeys := metricAttributeKeys(metric)
		if len(firstMetricAttrKeys) != len(metricAttrKeys) {
			return fmt.Errorf("metrics cannot be combined as they have different attributes: %v (%v) and %v (%v)",
				firstMetric.Name(), firstMetricAttrKeys, metric.Name(), metricAttrKeys)
		}

		for attr := range metricAttrKeys {
			if _, ok := firstMetricAttrKeys[attr]; !ok {
				return fmt.Errorf("metrics cannot be combined as they have different attributes: %v (%v) and %v (%v)",
					firstMetric.Name(), firstMetricAttrKeys, metric.Name(), metricAttrKeys)
			}
		}

		switch firstMetric.DataType() {
		case pmetric.MetricDataTypeSum:
			if firstMetric.Sum().AggregationTemporality() != metric.Sum().AggregationTemporality() {
				return fmt.Errorf(
					"metrics cannot be combined as they have different aggregation temporalities: %v (%v) and %v (%v)",
					firstMetric.Name(), firstMetric.Sum().AggregationTemporality(), metric.Name(), metric.Sum().AggregationTemporality())
			}
			if firstMetric.Sum().IsMonotonic() != metric.Sum().IsMonotonic() {
				return fmt.Errorf(
					"metrics cannot be combined as they have different monotonicity: %v (%v) and %v (%v)",
					firstMetric.Name(), firstMetric.Sum().IsMonotonic(), metric.Name(), metric.Sum().IsMonotonic())
			}
		case pmetric.MetricDataTypeHistogram:
			if firstMetric.Histogram().AggregationTemporality() != metric.Histogram().AggregationTemporality() {
				return fmt.Errorf(
					"metrics cannot be combined as they have different aggregation temporalities: %v (%v) and %v (%v)",
					firstMetric.Name(), firstMetric.Histogram().AggregationTemporality(), metric.Name(),
					metric.Histogram().AggregationTemporality())

			}
		case pmetric.MetricDataTypeExponentialHistogram:
			if firstMetric.ExponentialHistogram().AggregationTemporality() != metric.ExponentialHistogram().AggregationTemporality() {
				return fmt.Errorf(
					"metrics cannot be combined as they have different aggregation temporalities: %v (%v) and %v (%v)",
					firstMetric.Name(), firstMetric.ExponentialHistogram().AggregationTemporality(), metric.Name(),
					metric.ExponentialHistogram().AggregationTemporality())

			}
		}
	}

	return nil
}

func metricAttributeKeys(metric pmetric.Metric) map[string]struct{} {
	attrKeys := map[string]struct{}{}
	rangeDataPointAttributes(metric, func(attrs pcommon.Map) bool {
		attrs.Range(func(k string, _ pcommon.Value) bool {
			attrKeys[k] = struct{}{}
			return true
		})
		return true
	})
	return attrKeys
}

// combine combines the metrics based on the supplied filter.
// canBeCombined must be called before.
func combine(transform internalTransform, metrics pmetric.MetricSlice) pmetric.Metric {
	firstMetric := metrics.At(0)

	// create combined metric with relevant name & descriptor
	combinedMetric := pmetric.NewMetric()
	copyMetricDetails(firstMetric, combinedMetric)
	combinedMetric.SetName(transform.NewName)

	// append attribute keys based on the transform filter's named capturing groups
	subexprNames := transform.MetricIncludeFilter.getSubexpNames()
	reAttrKeys := make([]string, len(subexprNames))
	for i := 1; i < len(subexprNames); i++ {
		// if the subexpression is not named, use regexp notation, e.g. $1
		name := subexprNames[i]
		if name == "" {
			name = "$" + strconv.Itoa(i)
		}
		reAttrKeys[i] = name
	}

	for i := 0; i < metrics.Len(); i++ {
		metric := metrics.At(i)

		// append attr values based on regex submatches
		submatches := transform.MetricIncludeFilter.submatches(metric)
		if len(submatches) > 0 {
			rangeDataPointAttributes(metric, func(m pcommon.Map) bool {
				for i := 1; i < len(submatches)/2; i++ {
					submatch := metric.Name()[submatches[2*i]:submatches[2*i+1]]
					submatch = replaceCaseOfSubmatch(transform.SubmatchCase, submatch)
					if submatch != "" {
						m.UpsertString(reAttrKeys[i], submatch)
					}
				}
				return true
			})
		}
	}

	groupMetrics(metrics, transform.AggregationType, combinedMetric)

	return combinedMetric
}

func copyMetricDetails(from, to pmetric.Metric) {
	to.SetName(from.Name())
	to.SetDataType(from.DataType())
	to.SetUnit(from.Unit())
	switch from.DataType() {
	case pmetric.MetricDataTypeSum:
		to.Sum().SetAggregationTemporality(from.Sum().AggregationTemporality())
		to.Sum().SetIsMonotonic(from.Sum().IsMonotonic())
	case pmetric.MetricDataTypeHistogram:
		to.Histogram().SetAggregationTemporality(from.Histogram().AggregationTemporality())
	case pmetric.MetricDataTypeExponentialHistogram:
		to.ExponentialHistogram().SetAggregationTemporality(from.Histogram().AggregationTemporality())
	}
}

// rangeDataPointAttributes calls f sequentially on attributes of every metric data point.
// The iteration terminates if f returns false.
func rangeDataPointAttributes(metric pmetric.Metric, f func(pcommon.Map) bool) {
	switch metric.DataType() {
	case pmetric.MetricDataTypeGauge:
		for i := 0; i < metric.Gauge().DataPoints().Len(); i++ {
			dp := metric.Gauge().DataPoints().At(i)
			if !f(dp.Attributes()) {
				return
			}
		}
	case pmetric.MetricDataTypeSum:
		for i := 0; i < metric.Sum().DataPoints().Len(); i++ {
			dp := metric.Sum().DataPoints().At(i)
			if !f(dp.Attributes()) {
				return
			}
		}
	case pmetric.MetricDataTypeHistogram:
		for i := 0; i < metric.Histogram().DataPoints().Len(); i++ {
			dp := metric.Histogram().DataPoints().At(i)
			if !f(dp.Attributes()) {
				return
			}
		}
	case pmetric.MetricDataTypeExponentialHistogram:
		for i := 0; i < metric.ExponentialHistogram().DataPoints().Len(); i++ {
			dp := metric.ExponentialHistogram().DataPoints().At(i)
			if !f(dp.Attributes()) {
				return
			}
		}
	case pmetric.MetricDataTypeSummary:
		for i := 0; i < metric.Summary().DataPoints().Len(); i++ {
			dp := metric.Summary().DataPoints().At(i)
			if !f(dp.Attributes()) {
				return
			}
		}
	}
}

func countDataPoints(metric pmetric.Metric) int {
	switch metric.DataType() {
	case pmetric.MetricDataTypeGauge:
		return metric.Gauge().DataPoints().Len()
	case pmetric.MetricDataTypeSum:
		return metric.Sum().DataPoints().Len()
	case pmetric.MetricDataTypeHistogram:
		return metric.Histogram().DataPoints().Len()
	case pmetric.MetricDataTypeExponentialHistogram:
		return metric.ExponentialHistogram().DataPoints().Len()
	case pmetric.MetricDataTypeSummary:
		return metric.Summary().DataPoints().Len()
	}
	return 0
}

// transformMetric updates the metric content based on operations indicated in transform and returns a flag
// specifying whether the metric is valid after applying the translations,
// e.g. false is returned if all the data points were removed after applying the translations.
func transformMetric(metric pmetric.Metric, transform internalTransform) bool {
	isMetricEmpty := countDataPoints(metric) == 0
	canChangeMetric := transform.Action != Update || matchAllDps(metric, transform.MetricIncludeFilter)

	if transform.NewName != "" && canChangeMetric {
		if newName := transform.MetricIncludeFilter.expand(transform.NewName, metric.Name()); newName != "" {
			metric.SetName(newName)
		} else {
			metric.SetName(transform.NewName)
		}
	}

	for _, op := range transform.Operations {
		switch op.configOperation.Action {
		case UpdateLabel:
			updateLabelOp(metric, op, transform.MetricIncludeFilter)
		case AggregateLabels:
			if canChangeMetric {
				aggregateLabelsOp(metric, op)
			}
		case AggregateLabelValues:
			if canChangeMetric {
				aggregateLabelValuesOp(metric, op)
			}
		case ToggleScalarDataType:
			toggleScalarDataTypeOp(metric, transform.MetricIncludeFilter)
		case ScaleValue:
			scaleValueOp(metric, op, transform.MetricIncludeFilter)
		case AddLabel:
			if canChangeMetric {
				addLabelOp(metric, op)
			}
		case DeleteLabelValue:
			if canChangeMetric {
				deleteLabelValueOp(metric, op)
			}
		}
	}

	// Consider metric invalid if all its data points were removed after applying the operations.
	return isMetricEmpty || countDataPoints(metric) > 0
}
