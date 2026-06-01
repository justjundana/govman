package progress

import (
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	testCases := []struct {
		name        string
		total       int64
		description string
	}{
		{"Basic progress bar", 100, "Test download"},
		{"Zero total", 0, "Empty file"},
		{"Large total", 1000000, "Large file"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pb := New(tc.total, tc.description)

			if pb.total != tc.total {
				t.Errorf("Expected total %d, got %d", tc.total, pb.total)
			}
			if pb.current != 0 {
				t.Errorf("Expected current 0, got %d", pb.current)
			}
			if pb.description != tc.description {
				t.Errorf("Expected description %s, got %s", tc.description, pb.description)
			}
			if pb.finished {
				t.Error("Expected finished false")
			}
			if pb.lastPct != -1 {
				t.Errorf("Expected lastPct -1, got %d", pb.lastPct)
			}
		})
	}
}

func TestProgressBar_Write(t *testing.T) {
	testCases := []struct {
		name     string
		data     []byte
		expected int64
	}{
		{"Write small data", []byte("hello"), 5},
		{"Write empty data", []byte{}, 0},
		{"Write large data", make([]byte, 1000), 1000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pb := New(2000, "Test write")

			n, err := pb.Write(tc.data)

			if err != nil {
				t.Errorf("Write returned error: %v", err)
			}
			if n != len(tc.data) {
				t.Errorf("Expected to write %d bytes, got %d", len(tc.data), n)
			}
			if pb.current != tc.expected {
				t.Errorf("Expected current %d, got %d", tc.expected, pb.current)
			}
		})
	}
}

func TestProgressBar_Add(t *testing.T) {
	testCases := []struct {
		name     string
		total    int64
		initial  int64
		add      int64
		expected int64
	}{
		{"Add normal amount", 100, 10, 20, 30},
		{"Add exceeding total", 100, 90, 20, 100},
		{"Add zero", 100, 50, 0, 50},
		{"Add negative", 100, 50, -10, 40},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pb := New(tc.total, "Test add")
			pb.current = tc.initial

			pb.Add(tc.add)

			if pb.current != tc.expected {
				t.Errorf("Expected current %d, got %d", tc.expected, pb.current)
			}
		})
	}
}

func TestProgressBar_Set(t *testing.T) {
	testCases := []struct {
		name     string
		total    int64
		set      int64
		expected int64
	}{
		{"Set normal value", 100, 50, 50},
		{"Set exceeding total", 100, 150, 100},
		{"Set zero", 100, 0, 0},
		{"Set negative", 100, -10, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pb := New(tc.total, "Test set")

			pb.Set(tc.set)

			if pb.current != tc.expected {
				t.Errorf("Expected current %d, got %d", tc.expected, pb.current)
			}
		})
	}
}

func TestProgressBar_Finish(t *testing.T) {
	testCases := []struct {
		name      string
		total     int64
		current   int64
		callTwice bool
	}{
		{"Finish incomplete bar", 100, 50, false},
		{"Finish complete bar", 100, 100, false},
		{"Finish called twice", 100, 50, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pb := New(tc.total, "Test finish")
			pb.current = tc.current

			pb.Finish()

			if pb.current != tc.total {
				t.Errorf("Expected current %d, got %d", tc.total, pb.current)
			}
			if !pb.finished {
				t.Error("Expected finished true")
			}

			if tc.callTwice {
				originalCurrent := pb.current
				pb.Finish()
				if pb.current != originalCurrent {
					t.Errorf("Second finish changed current from %d to %d", originalCurrent, pb.current)
				}
			}
		})
	}
}

func TestProgressBar_ConcurrentAccess(t *testing.T) {
	pb := New(1000, "Concurrent test")

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				pb.Add(1)
				time.Sleep(time.Millisecond)
			}
		}()
	}

	wg.Wait()

	expected := int64(numGoroutines * numOperations)
	if pb.current != expected {
		t.Errorf("Expected current %d, got %d", expected, pb.current)
	}
}

func TestProgressBar_Render(t *testing.T) {
	testCases := []struct {
		name        string
		total       int64
		current     int64
		description string
		elapsed     time.Duration
	}{
		{"Normal render", 100, 50, "Test render", 2 * time.Second},
		{"Zero total (no render)", 0, 0, "Zero total", 0},
		{"Negative total (no render)", -1, 0, "Negative total", 0},
		{"Complete progress", 100, 100, "Complete", 5 * time.Second},
		{"Fast progress", 1000, 500, "Fast", 1 * time.Second},
		{"Slow progress no speed", 1000, 10, "Slow", 100 * time.Millisecond},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pb := New(tc.total, tc.description)
			pb.current = tc.current
			if tc.elapsed > 0 {
				pb.startTime = time.Now().Add(-tc.elapsed)
			}

			pb.render()

			if pb.current != tc.current {
				t.Errorf("Render changed current from %d to %d", tc.current, pb.current)
			}
		})
	}
}

func TestProgressBar_RenderSkipsDuplicatePercentage(t *testing.T) {
	pb := New(1000, "Dup test")
	pb.current = 100
	pb.startTime = time.Now().Add(-2 * time.Second)

	pb.render()
	if pb.lastPct != 10 {
		t.Errorf("Expected lastPct 10, got %d", pb.lastPct)
	}

	// Same percentage should be skipped
	pb.current = 105
	pb.render()
	if pb.lastPct != 10 {
		t.Errorf("Expected lastPct still 10, got %d", pb.lastPct)
	}

	// Different percentage should render
	pb.current = 200
	pb.render()
	if pb.lastPct != 20 {
		t.Errorf("Expected lastPct 20, got %d", pb.lastPct)
	}
}

func TestProgressBar_RenderEdgeCases(t *testing.T) {
	pb := New(100, "Fast completion")
	pb.current = 100
	pb.startTime = time.Now().Add(-10 * time.Millisecond)
	pb.render()

	pb2 := New(100, "Zero elapsed")
	pb2.current = 50
	pb2.startTime = time.Now()
	pb2.render()

	pb3 := New(100, "Over progress")
	pb3.current = 150
	pb3.render()
	if pb3.current != 150 {
		t.Errorf("Expected current 150, got %d", pb3.current)
	}
}

func TestProgressBar_AddThrottling(t *testing.T) {
	pb := New(1000, "Throttling test")

	for i := 0; i < 10; i++ {
		pb.Add(1)
		time.Sleep(10 * time.Millisecond)
	}

	pb.Add(990)

	if pb.current != 1000 {
		t.Errorf("Expected current 1000, got %d", pb.current)
	}
}

func TestProgressBar_RenderSpeedAndETA(t *testing.T) {
	pb := New(1000, "Speed test")
	pb.current = 500
	pb.startTime = time.Now().Add(-5 * time.Second)
	pb.render()

	pb2 := New(1000, "No ETA")
	pb2.current = 1000
	pb2.startTime = time.Now().Add(-5 * time.Second)
	pb2.render()

	pb3 := New(100, "Zero speed")
	pb3.current = 0
	pb3.startTime = time.Now().Add(-5 * time.Second)
	pb3.render()
}

func TestProgressBar_RenderNegativeCurrent(t *testing.T) {
	pb := New(100, "Negative")
	pb.current = -10
	pb.startTime = time.Now().Add(-2 * time.Second)
	pb.render()

	pb2 := New(-100, "Negative total")
	pb2.current = 50
	pb2.render()
}

func TestProgressBar_RenderMultipleCalls(t *testing.T) {
	pb := New(1000, "Multi render")
	pb.startTime = time.Now().Add(-5 * time.Second)

	pb.current = 100
	pb.render()

	pb.current = 200
	pb.render()

	pb.current = 500
	pb.render()
}

func TestProgressBar_RenderComprehensiveEdgeCases(t *testing.T) {
	testCases := []struct {
		name    string
		total   int64
		current int64
		elapsed time.Duration
	}{
		{"Small values", 10, 5, 2 * time.Second},
		{"Large values", 10000, 500000, 10 * time.Second},
		{"Minimal progress", 100, 1, 5 * time.Second},
		{"Near completion", 100, 99, 3 * time.Second},
		{"Zero elapsed", 100, 50, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pb := New(tc.total, tc.name)
			pb.current = tc.current
			if tc.elapsed > 0 {
				pb.startTime = time.Now().Add(-tc.elapsed)
			}
			pb.render()
		})
	}
}
