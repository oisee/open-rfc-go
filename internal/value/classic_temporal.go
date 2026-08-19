// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc src/values/classic-temporal.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten in Go. JavaScript bigint raw
// values fit int64 (UTCLONG max 3.15e18 < 9.2e18), so int64 is used throughout;
// thrown TypeError/RangeError became returned, wrapped errors; intrinsic-
// geometry snapshot → len()/copy. The proleptic Gregorian calendar with the
// 1582-10-05..14 gap and the hybrid week-53 table are preserved exactly. See
// docs/provenance.md.

package value

import (
	"encoding/binary"
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

// ErrTemporal reports an invalid temporal value or raw record.
var ErrTemporal = errors.New("value: classic temporal")

var (
	classicDatePattern     = regexp.MustCompile(`^(?:\d{8}| {8}|)$`)
	classicTimePattern     = regexp.MustCompile(`^(?:\d{6}| {6}|)$`)
	classicDateWirePattern = regexp.MustCompile(`^(?:\d{8}| {8})$`)
	classicTimeWirePattern = regexp.MustCompile(`^(?:\d{6}| {6})$`)
)

// AssertClassicDate validates a DATS input (YYYYMMDD, empty, or eight spaces).
func AssertClassicDate(value, path string) error {
	if !classicDatePattern.MatchString(value) {
		return fmt.Errorf("%w: %s expects YYYYMMDD, an empty string, or eight spaces", ErrTemporal, path)
	}
	return nil
}

// AssertClassicTime validates a TIMS input (HHMMSS, empty, or six spaces).
func AssertClassicTime(value, path string) error {
	if !classicTimePattern.MatchString(value) {
		return fmt.Errorf("%w: %s expects HHMMSS, an empty string, or six spaces", ErrTemporal, path)
	}
	return nil
}

// ClassicDateWireText converts a DATE value to the eight-character wire form.
func ClassicDateWireText(value, path string) (string, error) {
	if err := AssertClassicDate(value, path); err != nil {
		return "", err
	}
	if value == "" {
		return "        ", nil
	}
	return value, nil
}

// ClassicDatePublicText converts a wire DATE value to the trimmed public form.
func ClassicDatePublicText(value, path string) (string, error) {
	if !classicDateWirePattern.MatchString(value) {
		return "", fmt.Errorf("%w: %s expects YYYYMMDD or eight spaces from the wire", ErrTemporal, path)
	}
	if value == "        " {
		return "", nil
	}
	return value, nil
}

// ClassicTimeWireText converts a TIME value to the six-character wire form.
func ClassicTimeWireText(value, path string) (string, error) {
	if err := AssertClassicTime(value, path); err != nil {
		return "", err
	}
	if value == "" {
		return "      ", nil
	}
	return value, nil
}

// ClassicTimePublicText converts a wire TIME value to the trimmed public form.
func ClassicTimePublicText(value, path string) (string, error) {
	if !classicTimeWirePattern.MatchString(value) {
		return "", fmt.Errorf("%w: %s expects HHMMSS or six spaces from the wire", ErrTemporal, path)
	}
	if value == "      " {
		return "", nil
	}
	return value, nil
}

// TemporalExid is one classic RFC compact temporal EXID.
type TemporalExid string

type temporalSpec struct {
	name       string
	byteLength int
	maximumRaw int64
}

var temporalSpecs = map[TemporalExid]temporalSpec{
	"p": {"UTCLONG", 8, 3_155_380_704_000_000_000},
	"n": {"UTCSECOND", 8, 315_538_070_400},
	"w": {"UTCMINUTE", 8, 5_258_967_840},
	"d": {"DTDAY", 4, 3_652_061},
	"7": {"DTWEEK", 4, 521_725},
	"x": {"DTMONTH", 4, 119_988},
	"t": {"TSECOND", 4, 86_401},
	"i": {"TMINUTE", 2, 1_441},
	"c": {"CDAY", 2, 366},
}

var daysByMonth = [12]int{31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

const (
	utclongInitial     = "0000-00-00T00:00:00.0000000"
	secondsPerDay      = 86_400
	minutesPerDay      = 1_440
	fractionsPerSecond = 10_000_000
	fractionsPerDay    = 864_000_000_000
)

// IsClassicTemporalExid reports whether value selects a compact temporal codec.
func IsClassicTemporalExid(value string) bool {
	_, ok := temporalSpecs[TemporalExid(value)]
	return ok
}

func specification(exid TemporalExid) (temporalSpec, error) {
	spec, ok := temporalSpecs[exid]
	if !ok {
		return temporalSpec{}, fmt.Errorf("%w: unsupported classic temporal EXID", ErrTemporal)
	}
	return spec, nil
}

// ClassicTemporalByteLength returns the fixed raw width for a compact EXID.
func ClassicTemporalByteLength(exid TemporalExid) (int, error) {
	spec, err := specification(exid)
	if err != nil {
		return 0, err
	}
	return spec.byteLength, nil
}

// ClassicTemporalInitialValue returns the initial string for a compact EXID.
func ClassicTemporalInitialValue(exid TemporalExid) (string, error) {
	if _, err := specification(exid); err != nil {
		return "", err
	}
	if exid == "p" {
		return utclongInitial, nil
	}
	return "", nil
}

func isLeapYear(year int) bool {
	if year < 1582 {
		return year%4 == 0
	}
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

func daysInMonth(year, month int) int {
	if month == 2 {
		if isLeapYear(year) {
			return 29
		}
		return 28
	}
	if year == 1582 && month == 10 {
		return 21
	}
	return daysByMonth[month-1]
}

type calendarDate struct{ year, month, day int }

func parseDateParts(year, month, day int, path, name string) (calendarDate, error) {
	if year < 1 || year > 9999 {
		return calendarDate{}, fmt.Errorf("%w: %s %s year must be in 0001..9999", ErrTemporal, path, name)
	}
	if month < 1 || month > 12 {
		return calendarDate{}, fmt.Errorf("%w: %s %s month must be in 01..12", ErrTemporal, path, name)
	}
	conventionalMax := daysByMonth[month-1]
	if month == 2 {
		if isLeapYear(year) {
			conventionalMax = 29
		} else {
			conventionalMax = 28
		}
	}
	if day < 1 || day > conventionalMax {
		return calendarDate{}, fmt.Errorf("%w: %s %s has invalid day %02d", ErrTemporal, path, name, day)
	}
	if year == 1582 && month == 10 && day >= 5 && day <= 14 {
		return calendarDate{}, fmt.Errorf("%w: %s %s is in the Gregorian calendar gap 1582-10-05..1582-10-14", ErrTemporal, path, name)
	}
	return calendarDate{year, month, day}, nil
}

func daysInPreviousYears(year int) int {
	previousYears := year - 1
	through1600 := previousYears
	if through1600 > 1600 {
		through1600 = 1600
	}
	withinCentury := through1600 % 100
	days := (through1600/100)*36_525 + (withinCentury/4)*1_461 + (withinCentury%4)*365
	if year > 1582 {
		days -= 10
	}
	if previousYears <= 1600 {
		return days
	}
	after1600 := previousYears - 1600
	days += (after1600 / 400) * 146_097
	after1600 %= 400
	finalCentury := after1600 % 100
	return days + (after1600/100)*36_524 + (finalCentury/4)*1_461 + (finalCentury%4)*365
}

func dateOrdinal(date calendarDate) int {
	result := daysInPreviousYears(date.year)
	for month := 1; month < date.month; month++ {
		result += daysInMonth(date.year, month)
	}
	adjustedDay := date.day
	if date.year == 1582 && date.month == 10 && date.day >= 15 {
		adjustedDay = date.day - 10
	}
	return result + adjustedDay - 1
}

func dateFromOrdinal(ordinal int) calendarDate {
	lower, upper := 1, 9999
	for lower <= upper {
		candidate := (lower + upper) / 2
		if daysInPreviousYears(candidate) <= ordinal {
			lower = candidate + 1
		} else {
			upper = candidate - 1
		}
	}
	year := upper
	remaining := ordinal - daysInPreviousYears(year)
	month := 1
	for month <= 12 {
		monthLength := daysInMonth(year, month)
		if remaining < monthLength {
			break
		}
		remaining -= monthLength
		month++
	}
	day := remaining + 1
	if year == 1582 && month == 10 && day > 4 {
		day += 10
	}
	return calendarDate{year, month, day}
}

func twoDigits(v int) string  { return fmt.Sprintf("%02d", v) }
func fourDigits(v int) string { return fmt.Sprintf("%04d", v) }

func formatDate(d calendarDate) string {
	return fmt.Sprintf("%04d-%02d-%02d", d.year, d.month, d.day)
}

type clockTime struct{ hour, minute, second int }

func parseClock(hour, minute, second int, allowEndOfDay bool, path, name string) (clockTime, error) {
	maxHour := 23
	if allowEndOfDay {
		maxHour = 24
	}
	if hour < 0 || hour > maxHour {
		return clockTime{}, fmt.Errorf("%w: %s %s hours must be in 00..%d", ErrTemporal, path, name, maxHour)
	}
	if minute < 0 || minute > 59 {
		return clockTime{}, fmt.Errorf("%w: %s %s minutes must be in 00..59", ErrTemporal, path, name)
	}
	if second < 0 || second > 59 {
		return clockTime{}, fmt.Errorf("%w: %s %s seconds must be in 00..59", ErrTemporal, path, name)
	}
	if hour == 24 && (minute != 0 || second != 0) {
		maximum := "24:00:00"
		if name == "TMINUTE" {
			maximum = "24:00"
		}
		return clockTime{}, fmt.Errorf("%w: %s %s must not exceed %s", ErrTemporal, path, name, maximum)
	}
	return clockTime{hour, minute, second}, nil
}

func clockSeconds(c clockTime) int { return c.hour*3_600 + c.minute*60 + c.second }

func formatClock(seconds int, includeSeconds bool) string {
	hour := seconds / 3_600
	afterHours := seconds % 3_600
	minute := afterHours / 60
	if !includeSeconds {
		return fmt.Sprintf("%02d:%02d", hour, minute)
	}
	return fmt.Sprintf("%02d:%02d:%02d", hour, minute, afterHours%60)
}

func calendarYearHasWeek53(year int) bool {
	januaryFirst := (5 + daysInPreviousYears(year)) % 7
	return januaryFirst == 3 || (januaryFirst == 2 && isLeapYear(year))
}

var yearsWithWeek53 = func() []int {
	var result []int
	for year := 1; year <= 9999; year++ {
		if calendarYearHasWeek53(year) {
			result = append(result, year)
		}
	}
	return result
}()

func priorWeek53Count(year int) int {
	lower, upper := 0, len(yearsWithWeek53)
	for lower < upper {
		middle := (lower + upper) / 2
		if yearsWithWeek53[middle] < year {
			lower = middle + 1
		} else {
			upper = middle
		}
	}
	return lower
}

func weekOrdinal(year, week int, path string) (int, error) {
	if year == 0 {
		if week == 53 {
			return 0, nil
		}
		return 0, fmt.Errorf("%w: %s DTWEEK year zero permits only 0000-W53", ErrTemporal, path)
	}
	priorLongYears := priorWeek53Count(year)
	if week == 53 && (priorLongYears >= len(yearsWithWeek53) || yearsWithWeek53[priorLongYears] != year) {
		return 0, fmt.Errorf("%w: %s DTWEEK year %04d does not have week 53", ErrTemporal, path, year)
	}
	return priorLongYears*53 + (year-1-priorLongYears)*52 + week, nil
}

func weekFromOrdinal(ordinal int) (int, int) {
	if ordinal == 0 {
		return 0, 53
	}
	lower, upper := 1, 9999
	for lower < upper {
		year := (lower + upper) / 2
		throughYear := year*52 + priorWeek53Count(year+1)
		if ordinal <= throughYear {
			upper = year
		} else {
			lower = year + 1
		}
	}
	year := lower
	beforeYear := (year-1)*52 + priorWeek53Count(year)
	return year, ordinal - beforeYear
}

func encodeRaw(exid TemporalExid, raw int64, path string) ([]byte, error) {
	spec := temporalSpecs[exid]
	if raw < 0 || raw > spec.maximumRaw {
		return nil, fmt.Errorf("%w: %s %s is outside its valid raw range", ErrTemporal, path, spec.name)
	}
	out := make([]byte, spec.byteLength)
	switch spec.byteLength {
	case 8:
		binary.LittleEndian.PutUint64(out, uint64(raw))
	case 4:
		binary.LittleEndian.PutUint32(out, uint32(int32(raw)))
	default:
		binary.LittleEndian.PutUint16(out, uint16(int16(raw)))
	}
	return out, nil
}

func rawValue(exid TemporalExid, value []byte, path string) (int64, error) {
	spec := temporalSpecs[exid]
	if len(value) != spec.byteLength {
		return 0, fmt.Errorf("%w: %s %s expects %d raw bytes; received %d", ErrTemporal, path, spec.name, spec.byteLength, len(value))
	}
	var raw int64
	switch spec.byteLength {
	case 8:
		raw = int64(binary.LittleEndian.Uint64(value))
	case 4:
		raw = int64(int32(binary.LittleEndian.Uint32(value)))
	default:
		raw = int64(int16(binary.LittleEndian.Uint16(value)))
	}
	if raw < 0 || raw > spec.maximumRaw {
		return 0, fmt.Errorf("%w: %s %s is outside its valid raw range", ErrTemporal, path, spec.name)
	}
	return raw, nil
}

func matchForm(pattern *regexp.Regexp, value, expected, path, name string) ([]string, error) {
	m := pattern.FindStringSubmatch(value)
	if m == nil {
		return nil, fmt.Errorf("%w: %s %s expects %s", ErrTemporal, path, name, expected)
	}
	return m, nil
}

func atoi(s string) int { n, _ := strconv.Atoi(s); return n }

var (
	pForm = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})\.(\d{7})$`)
	nForm = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})$`)
	wForm = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})$`)
	dForm = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})$`)
	weekF = regexp.MustCompile(`^(\d{4})-W(\d{2})$`)
	xForm = regexp.MustCompile(`^(\d{4})-(\d{2})$`)
	tForm = regexp.MustCompile(`^(\d{2}):(\d{2}):(\d{2})$`)
	iForm = regexp.MustCompile(`^(\d{2}):(\d{2})$`)
	cForm = regexp.MustCompile(`^(\d{2})-(\d{2})$`)
)

// EncodeClassicTemporal encodes a compact SAP temporal string to its raw value.
func EncodeClassicTemporal(exid TemporalExid, value, path string) ([]byte, error) {
	if path == "" {
		path = "classic temporal value"
	}
	spec, err := specification(exid)
	if err != nil {
		return nil, err
	}
	if value == "" || (exid == "p" && value == utclongInitial) {
		return encodeRaw(exid, 0, path)
	}

	var raw int64
	switch exid {
	case "p":
		m, err := matchForm(pForm, value, "YYYY-MM-DDTHH:MM:SS.fffffff", path, spec.name)
		if err != nil {
			return nil, err
		}
		date, err := parseDateParts(atoi(m[1]), atoi(m[2]), atoi(m[3]), path, spec.name)
		if err != nil {
			return nil, err
		}
		clock, err := parseClock(atoi(m[4]), atoi(m[5]), atoi(m[6]), false, path, spec.name)
		if err != nil {
			return nil, err
		}
		raw = int64(dateOrdinal(date)*secondsPerDay+clockSeconds(clock))*fractionsPerSecond + int64(atoi(m[7])) + 1
	case "n":
		m, err := matchForm(nForm, value, "YYYY-MM-DDTHH:MM:SS", path, spec.name)
		if err != nil {
			return nil, err
		}
		date, err := parseDateParts(atoi(m[1]), atoi(m[2]), atoi(m[3]), path, spec.name)
		if err != nil {
			return nil, err
		}
		clock, err := parseClock(atoi(m[4]), atoi(m[5]), atoi(m[6]), false, path, spec.name)
		if err != nil {
			return nil, err
		}
		raw = int64(dateOrdinal(date)*secondsPerDay+clockSeconds(clock)) + 1
	case "w":
		m, err := matchForm(wForm, value, "YYYY-MM-DDTHH:MM", path, spec.name)
		if err != nil {
			return nil, err
		}
		date, err := parseDateParts(atoi(m[1]), atoi(m[2]), atoi(m[3]), path, spec.name)
		if err != nil {
			return nil, err
		}
		clock, err := parseClock(atoi(m[4]), atoi(m[5]), 0, false, path, spec.name)
		if err != nil {
			return nil, err
		}
		raw = int64(dateOrdinal(date)*minutesPerDay+clock.hour*60+clock.minute) + 1
	case "d":
		m, err := matchForm(dForm, value, "YYYY-MM-DD", path, spec.name)
		if err != nil {
			return nil, err
		}
		date, err := parseDateParts(atoi(m[1]), atoi(m[2]), atoi(m[3]), path, spec.name)
		if err != nil {
			return nil, err
		}
		raw = int64(dateOrdinal(date)) + 1
	case "7":
		m, err := matchForm(weekF, value, "YYYY-Www", path, spec.name)
		if err != nil {
			return nil, err
		}
		year, week := atoi(m[1]), atoi(m[2])
		if week < 1 || week > 53 {
			return nil, fmt.Errorf("%w: %s %s week must be in 01..53", ErrTemporal, path, spec.name)
		}
		ordinal, err := weekOrdinal(year, week, path)
		if err != nil {
			return nil, err
		}
		raw = int64(ordinal) + 1
	case "x":
		m, err := matchForm(xForm, value, "YYYY-MM", path, spec.name)
		if err != nil {
			return nil, err
		}
		year, month := atoi(m[1]), atoi(m[2])
		if year < 1 || year > 9999 {
			return nil, fmt.Errorf("%w: %s %s year must be in 0001..9999", ErrTemporal, path, spec.name)
		}
		if month < 1 || month > 12 {
			return nil, fmt.Errorf("%w: %s %s month must be in 01..12", ErrTemporal, path, spec.name)
		}
		raw = int64((year-1)*12 + month)
	case "t":
		m, err := matchForm(tForm, value, "HH:MM:SS", path, spec.name)
		if err != nil {
			return nil, err
		}
		clock, err := parseClock(atoi(m[1]), atoi(m[2]), atoi(m[3]), true, path, spec.name)
		if err != nil {
			return nil, err
		}
		raw = int64(clockSeconds(clock)) + 1
	case "i":
		m, err := matchForm(iForm, value, "HH:MM", path, spec.name)
		if err != nil {
			return nil, err
		}
		clock, err := parseClock(atoi(m[1]), atoi(m[2]), 0, true, path, spec.name)
		if err != nil {
			return nil, err
		}
		raw = int64(clock.hour*60+clock.minute) + 1
	case "c":
		m, err := matchForm(cForm, value, "MM-DD", path, spec.name)
		if err != nil {
			return nil, err
		}
		month, day := atoi(m[1]), atoi(m[2])
		if month < 1 || month > 12 {
			return nil, fmt.Errorf("%w: %s %s month must be in 01..12", ErrTemporal, path, spec.name)
		}
		if day < 1 || day > daysByMonth[month-1] {
			return nil, fmt.Errorf("%w: %s %s has invalid day %02d", ErrTemporal, path, spec.name, day)
		}
		ordinal := day
		for candidate := 1; candidate < month; candidate++ {
			ordinal += daysByMonth[candidate-1]
		}
		raw = int64(ordinal)
	}
	return encodeRaw(exid, raw, path)
}

// DecodeClassicTemporal decodes a fixed-width compact SAP temporal value.
func DecodeClassicTemporal(exid TemporalExid, value []byte, path string) (string, error) {
	if path == "" {
		path = "classic temporal value"
	}
	if _, err := specification(exid); err != nil {
		return "", err
	}
	raw, err := rawValue(exid, value, path)
	if err != nil {
		return "", err
	}
	if raw == 0 {
		return ClassicTemporalInitialValue(exid)
	}
	ordinal := raw - 1
	switch exid {
	case "p":
		dayOrdinal := int(ordinal / fractionsPerDay)
		withinDay := ordinal % fractionsPerDay
		seconds := int(withinDay / fractionsPerSecond)
		fraction := withinDay % fractionsPerSecond
		return formatDate(dateFromOrdinal(dayOrdinal)) + "T" + formatClock(seconds, true) + "." + fmt.Sprintf("%07d", fraction), nil
	case "n":
		dayOrdinal := int(ordinal / secondsPerDay)
		seconds := int(ordinal % secondsPerDay)
		return formatDate(dateFromOrdinal(dayOrdinal)) + "T" + formatClock(seconds, true), nil
	case "w":
		dayOrdinal := int(ordinal / minutesPerDay)
		minutes := int(ordinal % minutesPerDay)
		return formatDate(dateFromOrdinal(dayOrdinal)) + "T" + fmt.Sprintf("%02d:%02d", minutes/60, minutes%60), nil
	case "d":
		return formatDate(dateFromOrdinal(int(ordinal))), nil
	case "7":
		year, week := weekFromOrdinal(int(ordinal))
		return fmt.Sprintf("%04d-W%02d", year, week), nil
	case "x":
		monthOrdinal := int(ordinal)
		return fmt.Sprintf("%04d-%02d", monthOrdinal/12+1, monthOrdinal%12+1), nil
	case "t":
		return formatClock(int(ordinal), true), nil
	case "i":
		minutes := int(ordinal)
		return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60), nil
	case "c":
		remaining := int(ordinal)
		month := 1
		for remaining >= daysByMonth[month-1] {
			remaining -= daysByMonth[month-1]
			month++
		}
		return fmt.Sprintf("%02d-%02d", month, remaining+1), nil
	}
	return "", fmt.Errorf("%w: unreachable", ErrTemporal)
}
