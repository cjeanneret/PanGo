package debug

import (
	"fmt"
	"log"
	"os"
)

// Debug levels
const (
	LevelOff     = 0 // No output
	LevelInfo    = 1 // Important info (photo count, grid)
	LevelLive    = 2 // Live info (steps taken, photos taken)
	LevelVerbose = 3 // Verbose (calculation details, steps)
	LevelTrace   = 4 // Trace (GPIO, very low level)
)

var (
	level  int
	logger *log.Logger
)

// Init initializes the debug system with a level (0-4).
// 0 = no output
// 1 = important info (grid, total photo count)
// 2 = live info (movements, photos taken)
// 3 = verbose (calculation details, steps, FOV, angles)
// 4 = trace (GPIO, very low level)
func Init(debugLevel int) {
	level = debugLevel
	if level > LevelOff {
		logger = log.New(os.Stdout, "[PanGo] ", log.LstdFlags|log.Lmicroseconds)
	}
}

// Level returns the current debug level.
func Level() int {
	return level
}

// IsEnabled returns true if debug level is >= the requested level.
func IsEnabled(minLevel int) bool {
	return level >= minLevel
}

// --- Level 1 functions (Info): important info ---

// Info prints a level 1 message (important info).
func Info(format string, args ...interface{}) {
	if level >= LevelInfo && logger != nil {
		logger.Printf("[INFO] "+format, args...)
	}
}

// Summary prints an important summary (level 1).
func Summary(title string) {
	if level >= LevelOff && logger != nil {
		logger.Printf("═══════════════════════════════════════")
		logger.Printf("  %s", title)
		logger.Printf("═══════════════════════════════════════")
	}
}

// Grid prints important grid info (level 1).
func Grid(columns, rows, totalPhotos int) {
	if level >= LevelInfo && logger != nil {
		logger.Printf("[INFO] Grid: %d columns x %d rows = %d photos total", columns, rows, totalPhotos)
	}
}

// --- Level 2 functions (Live): real-time info ---

// Live prints a level 2 message (live info).
func Live(format string, args ...interface{}) {
	if level >= LevelLive && logger != nil {
		logger.Printf("[LIVE] "+format, args...)
	}
}

// Move prints a motor movement (level 2).
func Move(motor string, steps int, direction string) {
	if level >= LevelLive && logger != nil {
		logger.Printf("[LIVE] Motor %s: %d steps (%s)", motor, steps, direction)
	}
}

// Shot prints a photo capture (level 2).
func Shot(col, row int) {
	if level >= LevelLive && logger != nil {
		logger.Printf("[LIVE] Photo taken at position (col=%d, row=%d)", col, row)
	}
}

// Column prints the start of a column (level 2).
func Column(col, totalCols int, direction string) {
	if level >= LevelLive && logger != nil {
		logger.Printf("[LIVE] Starting column %d/%d (direction: %s)", col, totalCols, direction)
	}
}

// --- Level 3 functions (Verbose): everything ---

// Verbose prints a level 3 message (verbose).
func Verbose(format string, args ...interface{}) {
	if level >= LevelVerbose && logger != nil {
		logger.Printf("[VERBOSE] "+format, args...)
	}
}

// Print prints a level 3 message (alias for Verbose).
func Print(format string, args ...interface{}) {
	Verbose(format, args...)
}

// Printf is an alias for Print for compatibility.
func Printf(format string, args ...interface{}) {
	Verbose(format, args...)
}

// Println prints a level 3 message followed by a newline.
func Println(args ...interface{}) {
	if level >= LevelVerbose && logger != nil {
		logger.Println(args...)
	}
}

// PrintStruct prints a struct in formatted form (level 3).
func PrintStruct(name string, v interface{}) {
	if level >= LevelVerbose && logger != nil {
		logger.Printf("[VERBOSE] %s: %+v", name, v)
	}
}

// Section prints a section separator (level 3).
func Section(name string) {
	if level >= LevelVerbose && logger != nil {
		logger.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		logger.Printf("  %s", name)
		logger.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	}
}

// Step prints a numbered step (level 3).
func Step(num int, description string) {
	if level >= LevelVerbose && logger != nil {
		logger.Printf("[VERBOSE] Step %d: %s", num, description)
	}
}

// Value prints a named value in formatted form (level 3).
func Value(name string, value interface{}) {
	if level >= LevelInfo && logger != nil {
		logger.Printf("[INFO]   %s = %v", name, value)
	}
}

// --- Level 4 functions (Trace): very low level ---

// Trace prints a level 4 message (trace, GPIO).
func Trace(format string, args ...interface{}) {
	if level >= LevelTrace && logger != nil {
		logger.Printf("[TRACE] "+format, args...)
	}
}

// GPIO prints a GPIO operation (level 4).
func GPIO(operation string, pin int, value interface{}) {
	if level >= LevelTrace && logger != nil {
		logger.Printf("[GPIO] %s pin=%d value=%v", operation, pin, value)
	}
}

// --- General functions ---

// Error prints a debug error (level 1+).
func Error(err error) {
	if level >= LevelInfo && logger != nil {
		logger.Printf("[ERROR] %v", err)
	}
}

// Fmt is a helper function that returns a formatted string
// only if debug is enabled (to avoid unnecessary allocations).
func Fmt(format string, args ...interface{}) string {
	if level > 0 {
		return fmt.Sprintf(format, args...)
	}
	return ""
}
