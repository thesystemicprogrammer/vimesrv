package job

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewRobfigCronParser tests parser initialization
func TestNewRobfigCronParser(t *testing.T) {
	parser := NewRobfigCronParser()

	assert.NotNil(t, parser)
	assert.NotNil(t, parser.parser)
}

// TestNewSecondBasedCronParser tests second-based parser initialization
func TestNewSecondBasedCronParser(t *testing.T) {
	parser := NewSecondBasedCronParser()

	assert.NotNil(t, parser)
	assert.NotNil(t, parser.parser)
}

// TestRobfigCronParser_Parse_ValidSpecs tests parsing valid cron specs
func TestRobfigCronParser_Parse_ValidSpecs(t *testing.T) {
	parser := NewRobfigCronParser()

	testCases := []struct {
		name string
		spec string
	}{
		{
			name: "Every minute",
			spec: "* * * * *",
		},
		{
			name: "Every 5 minutes",
			spec: "*/5 * * * *",
		},
		{
			name: "Every hour",
			spec: "0 * * * *",
		},
		{
			name: "Daily at midnight",
			spec: "0 0 * * *",
		},
		{
			name: "Every Monday at 9am",
			spec: "0 9 * * 1",
		},
		{
			name: "First day of month",
			spec: "0 0 1 * *",
		},
		{
			name: "Specific time",
			spec: "30 15 * * *", // 3:30 PM daily
		},
		{
			name: "Complex schedule",
			spec: "15,45 */2 * * 1-5", // 15 and 45 minutes past every 2 hours on weekdays
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			schedule, err := parser.Parse(tc.spec)

			require.NoError(t, err, "Failed to parse spec: %s", tc.spec)
			assert.NotNil(t, schedule)

			// Verify the schedule can calculate next run
			now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
			next := schedule.Next(now)
			assert.True(t, next.After(now), "Next run should be after current time")
		})
	}
}

// TestRobfigCronParser_Parse_InvalidSpecs tests parsing invalid cron specs
func TestRobfigCronParser_Parse_InvalidSpecs(t *testing.T) {
	parser := NewRobfigCronParser()

	testCases := []struct {
		name string
		spec string
	}{
		{
			name: "Empty spec",
			spec: "",
		},
		{
			name: "Too few fields",
			spec: "* * *",
		},
		{
			name: "Too many fields",
			spec: "* * * * * * *",
		},
		{
			name: "Invalid characters",
			spec: "abc def ghi jkl mno",
		},
		{
			name: "Out of range minute",
			spec: "60 * * * *",
		},
		{
			name: "Out of range hour",
			spec: "0 24 * * *",
		},
		{
			name: "Invalid range",
			spec: "0 10-5 * * *",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			schedule, err := parser.Parse(tc.spec)

			require.Error(t, err, "Expected error for invalid spec: %s", tc.spec)
			assert.Nil(t, schedule)
		})
	}
}

// TestRobfigCronParser_NextCalculation tests next run calculation
func TestRobfigCronParser_NextCalculation(t *testing.T) {
	parser := NewRobfigCronParser()

	// Every hour at minute 30
	schedule, err := parser.Parse("30 * * * *")
	require.NoError(t, err)

	// Current time: 12:00
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	next := schedule.Next(now)

	// Next should be 12:30
	expected := time.Date(2024, 1, 1, 12, 30, 0, 0, time.UTC)
	assert.Equal(t, expected, next)

	// From 12:45, next should be 13:30
	now2 := time.Date(2024, 1, 1, 12, 45, 0, 0, time.UTC)
	next2 := schedule.Next(now2)
	expected2 := time.Date(2024, 1, 1, 13, 30, 0, 0, time.UTC)
	assert.Equal(t, expected2, next2)
}

// TestRobfigCronParser_DailySchedule tests daily schedule
func TestRobfigCronParser_DailySchedule(t *testing.T) {
	parser := NewRobfigCronParser()

	// Daily at 3:00 AM
	schedule, err := parser.Parse("0 3 * * *")
	require.NoError(t, err)

	// Current time: Jan 1, 2024 at 2:00 AM
	now := time.Date(2024, 1, 1, 2, 0, 0, 0, time.UTC)
	next := schedule.Next(now)

	// Next should be Jan 1, 2024 at 3:00 AM (same day)
	expected := time.Date(2024, 1, 1, 3, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, next)

	// If current time is 4:00 AM, next should be tomorrow at 3:00 AM
	now2 := time.Date(2024, 1, 1, 4, 0, 0, 0, time.UTC)
	next2 := schedule.Next(now2)
	expected2 := time.Date(2024, 1, 2, 3, 0, 0, 0, time.UTC)
	assert.Equal(t, expected2, next2)
}

// TestRobfigCronParser_WeeklySchedule tests weekly schedule
func TestRobfigCronParser_WeeklySchedule(t *testing.T) {
	parser := NewRobfigCronParser()

	// Every Monday at 9:00 AM
	schedule, err := parser.Parse("0 9 * * 1")
	require.NoError(t, err)

	// Current time: Monday, Jan 1, 2024 at 8:00 AM
	now := time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC)
	next := schedule.Next(now)

	// Next should be same Monday at 9:00 AM
	expected := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, next)

	// If it's Monday 10:00 AM, next should be next Monday
	now2 := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	next2 := schedule.Next(now2)
	expected2 := time.Date(2024, 1, 8, 9, 0, 0, 0, time.UTC)
	assert.Equal(t, expected2, next2)
}

// TestRobfigCronParser_MonthlySchedule tests monthly schedule
func TestRobfigCronParser_MonthlySchedule(t *testing.T) {
	parser := NewRobfigCronParser()

	// First day of month at midnight
	schedule, err := parser.Parse("0 0 1 * *")
	require.NoError(t, err)

	// Current time: Jan 15, 2024
	now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	next := schedule.Next(now)

	// Next should be Feb 1, 2024
	expected := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, next)
}

// TestRobfigCronParser_MultipleExecutions tests multiple consecutive executions
func TestRobfigCronParser_MultipleExecutions(t *testing.T) {
	parser := NewRobfigCronParser()

	// Every 15 minutes
	schedule, err := parser.Parse("*/15 * * * *")
	require.NoError(t, err)

	// Start at 12:00
	current := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Calculate next 5 executions
	expected := []time.Time{
		time.Date(2024, 1, 1, 12, 15, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 12, 30, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 12, 45, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 13, 15, 0, 0, time.UTC),
	}

	for i, exp := range expected {
		next := schedule.Next(current)
		assert.Equal(t, exp, next, "Execution %d mismatch", i+1)
		current = next
	}
}

// TestSecondBasedCronParser_Parse tests second-based parsing
func TestSecondBasedCronParser_Parse(t *testing.T) {
	parser := NewSecondBasedCronParser()

	// Every 5 seconds
	schedule, err := parser.Parse("*/5 * * * * *")
	require.NoError(t, err)
	assert.NotNil(t, schedule)

	// Calculate next from 12:00:00
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	next := schedule.Next(now)

	// Next should be 12:00:05
	expected := time.Date(2024, 1, 1, 12, 0, 5, 0, time.UTC)
	assert.Equal(t, expected, next)
}

// TestSecondBasedCronParser_NextCalculation tests next calculation with seconds
func TestSecondBasedCronParser_NextCalculation(t *testing.T) {
	parser := NewSecondBasedCronParser()

	// Every 10 seconds
	schedule, err := parser.Parse("*/10 * * * * *")
	require.NoError(t, err)

	// Start at 12:00:00
	current := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Calculate next 3 executions
	expected := []time.Time{
		time.Date(2024, 1, 1, 12, 0, 10, 0, time.UTC),
		time.Date(2024, 1, 1, 12, 0, 20, 0, time.UTC),
		time.Date(2024, 1, 1, 12, 0, 30, 0, time.UTC),
	}

	for i, exp := range expected {
		next := schedule.Next(current)
		assert.Equal(t, exp, next, "Execution %d mismatch", i+1)
		current = next
	}
}

// TestRobfigCronParser_EdgeCases tests edge cases
func TestRobfigCronParser_EdgeCases(t *testing.T) {
	parser := NewRobfigCronParser()

	testCases := []struct {
		name     string
		spec     string
		fromTime time.Time
		expected time.Time
	}{
		{
			name:     "End of month",
			spec:     "0 0 31 * *",
			fromTime: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "February 29 (leap year)",
			spec:     "0 0 29 2 *",
			fromTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "End of year",
			spec:     "0 0 31 12 *",
			fromTime: time.Date(2024, 12, 30, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			schedule, err := parser.Parse(tc.spec)
			require.NoError(t, err)

			next := schedule.Next(tc.fromTime)
			assert.Equal(t, tc.expected, next)
		})
	}
}
