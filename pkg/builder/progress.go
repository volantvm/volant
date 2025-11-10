// Copyright (c) 2025 HYPR. PTE. LTD.

package builder

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

// ProgressReporter handles build progress output with beautiful formatting
type ProgressReporter interface {
	// Stage tracking
	StartStage(name string)
	CompleteStage(name string, duration time.Duration)
	FailStage(name string, err error)

	// Step tracking within stages
	StartStep(name string)
	UpdateStep(name string, message string)
	CompleteStep(name string)

	// Cache operations
	CacheHit(layer string)
	CacheMiss(layer string)

	// Progress bars
	StartProgress(name string, total int64)
	UpdateProgress(name string, current int64)
	CompleteProgress(name string)

	// Final summary
	BuildComplete(imagePath string, size int64, duration time.Duration)
	BuildFailed(err error)
}

// ConsoleReporter provides beautiful console output for builds
type ConsoleReporter struct {
	w              io.Writer
	mu             sync.Mutex
	startTime      time.Time
	currentStage   string
	stagesComplete int
	totalStages    int
	spinnerFrames  []string
	spinnerIndex   int
	lastSpinner    time.Time
	progressBars   map[string]*progressBar
}

type progressBar struct {
	name    string
	total   int64
	current int64
	started time.Time
}

var (
	// Colors
	cyan    = color.New(color.FgCyan, color.Bold)
	green   = color.New(color.FgGreen, color.Bold)
	red     = color.New(color.FgRed, color.Bold)
	yellow  = color.New(color.FgYellow, color.Bold)
	blue    = color.New(color.FgBlue)
	gray    = color.New(color.FgHiBlack)
	white   = color.New(color.FgWhite, color.Bold)

	// Emojis and symbols
	sparkles     = "✨"
	rocket       = "🚀"
	gear         = "⚙️"
	check        = "✓"
	cross        = "✗"
	bolt         = "⚡"
	package_icon = "📦"
	mag          = "🔍"
	rocket_up    = "🚀"
	tada         = "🎉"
	fire         = "🔥"
	zap          = "⚡"
	cached       = "💾"
)

// NewConsoleReporter creates a new beautiful console reporter
func NewConsoleReporter(w io.Writer) *ConsoleReporter {
	return &ConsoleReporter{
		w:         w,
		startTime: time.Now(),
		spinnerFrames: []string{
			"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
		},
		progressBars: make(map[string]*progressBar),
	}
}

func (r *ConsoleReporter) StartStage(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.currentStage = name
	elapsed := time.Since(r.startTime).Round(time.Millisecond)

	fmt.Fprintf(r.w, "\n")
	fmt.Fprintf(r.w, "%s %s\n",
		cyan.Sprint("▶"),
		white.Sprintf("[%s] %s", formatDuration(elapsed), name))
}

func (r *ConsoleReporter) CompleteStage(name string, duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.stagesComplete++
	elapsed := time.Since(r.startTime).Round(time.Millisecond)

	fmt.Fprintf(r.w, "%s %s %s\n",
		green.Sprint(check),
		white.Sprintf("[%s]", formatDuration(elapsed)),
		gray.Sprintf("%s (%s)", name, formatDuration(duration)))
}

func (r *ConsoleReporter) FailStage(name string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	elapsed := time.Since(r.startTime).Round(time.Millisecond)

	fmt.Fprintf(r.w, "%s %s %s\n",
		red.Sprint(cross),
		white.Sprintf("[%s]", formatDuration(elapsed)),
		red.Sprintf("%s: %v", name, err))
}

func (r *ConsoleReporter) StartStep(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	fmt.Fprintf(r.w, "  %s %s\n",
		yellow.Sprint(r.getSpinner()),
		blue.Sprint(name))
	r.lastSpinner = time.Now()
}

func (r *ConsoleReporter) UpdateStep(name string, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Show spinner animation if enough time has passed
	if time.Since(r.lastSpinner) > 100*time.Millisecond {
		fmt.Fprintf(r.w, "\r  %s %s: %s",
			yellow.Sprint(r.getSpinner()),
			blue.Sprint(name),
			gray.Sprint(message))
		r.lastSpinner = time.Now()
	}
}

func (r *ConsoleReporter) CompleteStep(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	fmt.Fprintf(r.w, "\r  %s %s\n",
		green.Sprint(check),
		gray.Sprint(name))
}

func (r *ConsoleReporter) CacheHit(layer string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	fmt.Fprintf(r.w, "  %s %s %s\n",
		green.Sprint(cached),
		green.Sprint("CACHED"),
		gray.Sprint(layer))
}

func (r *ConsoleReporter) CacheMiss(layer string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	fmt.Fprintf(r.w, "  %s %s %s\n",
		blue.Sprint(gear),
		blue.Sprint("BUILD "),
		gray.Sprint(layer))
}

func (r *ConsoleReporter) StartProgress(name string, total int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.progressBars[name] = &progressBar{
		name:    name,
		total:   total,
		current: 0,
		started: time.Now(),
	}
}

func (r *ConsoleReporter) UpdateProgress(name string, current int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	pb, ok := r.progressBars[name]
	if !ok {
		return
	}

	pb.current = current
	percent := float64(current) / float64(pb.total) * 100

	// Draw progress bar
	barWidth := 30
	filled := int(float64(barWidth) * percent / 100)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	fmt.Fprintf(r.w, "\r  %s %s [%s] %.1f%% %s",
		yellow.Sprint(bolt),
		blue.Sprint(name),
		cyan.Sprint(bar),
		percent,
		gray.Sprintf("(%s / %s)", formatBytes(current), formatBytes(pb.total)))
}

func (r *ConsoleReporter) CompleteProgress(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	pb, ok := r.progressBars[name]
	if !ok {
		return
	}

	duration := time.Since(pb.started)
	fmt.Fprintf(r.w, "\r  %s %s %s %s\n",
		green.Sprint(check),
		gray.Sprint(name),
		gray.Sprintf("%s", formatBytes(pb.total)),
		gray.Sprintf("(%s)", formatDuration(duration)))

	delete(r.progressBars, name)
}

func (r *ConsoleReporter) BuildComplete(imagePath string, size int64, duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	fmt.Fprintf(r.w, "\n")
	fmt.Fprintf(r.w, "%s %s\n",
		green.Sprint(strings.Repeat("━", 60)),
		"")
	fmt.Fprintf(r.w, "%s %s\n\n",
		tada,
		green.Sprint("Build Complete!"))

	fmt.Fprintf(r.w, "  %s Image:     %s\n", package_icon, white.Sprint(imagePath))
	fmt.Fprintf(r.w, "  %s Size:      %s\n", mag, cyan.Sprint(formatBytes(size)))
	fmt.Fprintf(r.w, "  %s Duration:  %s\n", zap, yellow.Sprint(formatDuration(duration)))

	fmt.Fprintf(r.w, "\n%s Ready to run:\n", rocket_up)
	imageName := extractImageName(imagePath)
	fmt.Fprintf(r.w, "  %s\n",
		cyan.Sprintf("volant vms create my-vm --image %s", imageName))

	fmt.Fprintf(r.w, "\n")
}

func (r *ConsoleReporter) BuildFailed(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	duration := time.Since(r.startTime)

	fmt.Fprintf(r.w, "\n")
	fmt.Fprintf(r.w, "%s %s\n",
		red.Sprint(strings.Repeat("━", 60)),
		"")
	fmt.Fprintf(r.w, "%s %s\n\n",
		cross,
		red.Sprint("Build Failed"))

	fmt.Fprintf(r.w, "  Error: %s\n", red.Sprint(err))
	fmt.Fprintf(r.w, "  Duration: %s\n", gray.Sprint(formatDuration(duration)))

	fmt.Fprintf(r.w, "\n%s Troubleshooting:\n", mag)
	fmt.Fprintf(r.w, "  • Check Dockerfile syntax\n")
	fmt.Fprintf(r.w, "  • Ensure required dependencies are installed (skopeo, umoci, mksquashfs)\n")
	fmt.Fprintf(r.w, "  • Review error message above\n")
	fmt.Fprintf(r.w, "  • Run with VOLANT_DEBUG=1 for detailed logs\n")

	fmt.Fprintf(r.w, "\n")
}

// Helper methods

func (r *ConsoleReporter) getSpinner() string {
	frame := r.spinnerFrames[r.spinnerIndex]
	r.spinnerIndex = (r.spinnerIndex + 1) % len(r.spinnerFrames)
	return frame
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func extractImageName(path string) string {
	// Extract image name from path like "nginx-latest.squashfs" → "nginx:latest"
	base := path
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		base = path[idx+1:]
	}
	// Remove .squashfs extension
	if idx := strings.LastIndex(base, ".squashfs"); idx >= 0 {
		base = base[:idx]
	}
	// Replace - with : for tag separator
	if idx := strings.LastIndex(base, "-"); idx >= 0 {
		return base[:idx] + ":" + base[idx+1:]
	}
	return base + ":latest"
}

// SilentReporter implements ProgressReporter but produces no output
type SilentReporter struct{}

func NewSilentReporter() *SilentReporter {
	return &SilentReporter{}
}

func (r *SilentReporter) StartStage(name string)                                  {}
func (r *SilentReporter) CompleteStage(name string, duration time.Duration)       {}
func (r *SilentReporter) FailStage(name string, err error)                        {}
func (r *SilentReporter) StartStep(name string)                                   {}
func (r *SilentReporter) UpdateStep(name string, message string)                  {}
func (r *SilentReporter) CompleteStep(name string)                                {}
func (r *SilentReporter) CacheHit(layer string)                                   {}
func (r *SilentReporter) CacheMiss(layer string)                                  {}
func (r *SilentReporter) StartProgress(name string, total int64)                  {}
func (r *SilentReporter) UpdateProgress(name string, current int64)               {}
func (r *SilentReporter) CompleteProgress(name string)                            {}
func (r *SilentReporter) BuildComplete(imagePath string, size int64, duration time.Duration) {}
func (r *SilentReporter) BuildFailed(err error)                                   {}
