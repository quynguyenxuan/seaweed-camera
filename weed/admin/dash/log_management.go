package dash

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/seaweedfs/seaweedfs/weed/glog"
)

// LogFileInfo represents information about a log file
type LogFileInfo struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	ModifiedTime time.Time `json:"modified_time"`
	Level        string    `json:"level"`
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	File      string    `json:"file,omitempty"`
	Line      int       `json:"line,omitempty"`
}

// LogSearchResult represents search results
type LogSearchResult struct {
	Entries []LogEntry `json:"entries"`
	Total   int        `json:"total"`
	Query   string     `json:"query"`
}

// GetSystemLogs returns list of available log files
func (s *AdminServer) GetSystemLogs(c *gin.Context) {
	glog.V(2).Infof("QUYNGUYEN: GetSystemLogs called")

	// Get log directory from glog
	// Priority: 1. LOG_DIR env var, 2. os.TempDir()
	logDir := os.TempDir() // Default
	if os.Getenv("LOG_DIR") != "" {
		logDir = os.Getenv("LOG_DIR")
	}

	var logFiles []LogFileInfo

	// Look for log files with common patterns
	patterns := []string{
		"weed.INFO.*",
		"weed.WARNING.*",
		"weed.ERROR.*",
		"weed.FATAL.*",
		"weed.*.INFO.*",
		"weed.*.WARNING.*",
		"weed.*.ERROR.*",
		"weed.*.FATAL.*",
	}

	for _, pattern := range patterns {
		files, err := filepath.Glob(filepath.Join(logDir, pattern))
		if err != nil {
			glog.Errorf("QUYNGUYEN: Failed to glob pattern %s: %v", pattern, err)
			continue
		}

		for _, file := range files {
			stat, err := os.Stat(file)
			if err != nil {
				glog.Errorf("QUYNGUYEN: Failed to stat file %s: %v", file, err)
				continue
			}

			// Extract log level from filename
			level := "INFO"
			if strings.Contains(file, "WARNING") {
				level = "WARNING"
			} else if strings.Contains(file, "ERROR") {
				level = "ERROR"
			}

			logFiles = append(logFiles, LogFileInfo{
				Name:         filepath.Base(file),
				Path:         file,
				Size:         stat.Size(),
				ModifiedTime: stat.ModTime(),
				Level:        level,
			})
		}
	}

	// Sort by modified time (newest first)
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].ModifiedTime.After(logFiles[j].ModifiedTime)
	})

	c.JSON(http.StatusOK, gin.H{
		"logs":  logFiles,
		"count": len(logFiles),
	})
}

// GetLogFile returns content of a specific log file
func (s *AdminServer) GetLogFile(c *gin.Context) {
	glog.V(2).Infof("QUYNGUYEN: GetLogFile called")

	filename := c.Query("file")
	linesStr := c.DefaultQuery("lines", "100")

	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File parameter is required"})
		return
	}

	lines := 100
	if parsed, err := strconv.Atoi(linesStr); err == nil && parsed > 0 {
		lines = parsed
	}

	// Get log directory
	logDir := os.TempDir()
	if os.Getenv("LOG_DIR") != "" {
		logDir = os.Getenv("LOG_DIR")
	}

	filePath := filepath.Join(logDir, filename)

	// Security check - ensure file is in log directory and has valid extension
	if !strings.HasPrefix(filePath, logDir) ||
		!strings.HasSuffix(filename, ".log") &&
			!strings.Contains(filename, ".INFO.") &&
			!strings.Contains(filename, ".WARNING.") &&
			!strings.Contains(filename, ".ERROR.") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file path"})
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("File not found: %v", err)})
		return
	}
	defer file.Close()

	// Read last N lines
	content, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to read file: %v", err)})
		return
	}

	fileLines := strings.Split(string(content), "\n")

	// Get last N lines (handle case where file has fewer lines than requested)
	start := 0
	if len(fileLines) > lines {
		start = len(fileLines) - lines
	}

	recentLines := fileLines[start:]

	// Filter out empty lines
	var filteredLines []string
	for _, line := range recentLines {
		if strings.TrimSpace(line) != "" {
			filteredLines = append(filteredLines, line)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"filename": filename,
		"lines":    len(filteredLines),
		"content":  filteredLines,
	})
}

// SearchLogs searches across log files for specific patterns
func (s *AdminServer) SearchLogs(c *gin.Context) {
	glog.V(2).Infof("QUYNGUYEN: SearchLogs called")

	query := c.Query("query")
	level := c.Query("level")
	maxResultsStr := c.DefaultQuery("max", "50")

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter is required"})
		return
	}

	maxResults := 50
	if parsed, err := strconv.Atoi(maxResultsStr); err == nil && parsed > 0 && parsed <= 1000 {
		maxResults = parsed
	}

	// Get log directory
	logDir := os.TempDir()
	if os.Getenv("LOG_DIR") != "" {
		logDir = os.Getenv("LOG_DIR")
	}

	var allEntries []LogEntry

	// Search in recent log files
	patterns := []string{
		"weed.INFO.*",
		"weed.WARNING.*",
		"weed.ERROR.*",
		"weed.FATAL.*",
		"weed.*.INFO.*",
		"weed.*.WARNING.*",
		"weed.*.ERROR.*",
		"weed.*.FATAL.*",
	}

	for _, pattern := range patterns {
		// Skip if level filter doesn't match
		if level != "" {
			if !strings.Contains(pattern, strings.ToUpper(level)) {
				continue
			}
		}

		files, err := filepath.Glob(filepath.Join(logDir, pattern))
		if err != nil {
			continue
		}

		// Search only in most recent file for each pattern
		if len(files) > 0 {
			sort.Slice(files, func(i, j int) bool {
				statI, _ := os.Stat(files[i])
				statJ, _ := os.Stat(files[j])
				return statI.ModTime().After(statJ.ModTime())
			})

			entries := s.searchInFile(files[0], query, maxResults/len(patterns))
			allEntries = append(allEntries, entries...)
		}
	}

	// Sort by timestamp (newest first) and limit results
	sort.Slice(allEntries, func(i, j int) bool {
		return allEntries[i].Timestamp.After(allEntries[j].Timestamp)
	})

	if len(allEntries) > maxResults {
		allEntries = allEntries[:maxResults]
	}

	c.JSON(http.StatusOK, LogSearchResult{
		Entries: allEntries,
		Total:   len(allEntries),
		Query:   query,
	})
}

// GetCollectionCleanupLogs returns logs specific to collection cleanup operations
func (s *AdminServer) GetCollectionCleanupLogs(c *gin.Context) {
	glog.V(2).Infof("QUYNGUYEN: GetCollectionCleanupLogs called")

	collectionName := c.Param("name")
	maxResultsStr := c.DefaultQuery("max", "100")

	if collectionName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collection name is required"})
		return
	}

	maxResults := 100
	if parsed, err := strconv.Atoi(maxResultsStr); err == nil && parsed > 0 && parsed <= 1000 {
		maxResults = parsed
	}

	// Search for collection cleanup related logs
	searchQueries := []string{
		fmt.Sprintf("AdminServer.CleanupCollectionHandler - collection: %s", collectionName),
		fmt.Sprintf("CleanupCollection - collection: %s", collectionName),
		fmt.Sprintf("CollectionCleanup - collection: %s", collectionName),
		fmt.Sprintf("collection.*%s.*cleanup", collectionName),
		fmt.Sprintf("cleanup.*collection.*%s", collectionName),
	}

	var allEntries []LogEntry

	for _, query := range searchQueries {
		// Use existing search functionality
		entries := s.searchCollectionCleanupLogs(query, collectionName, maxResults/len(searchQueries))
		allEntries = append(allEntries, entries...)
	}

	// Sort by timestamp (newest first) and limit results
	sort.Slice(allEntries, func(i, j int) bool {
		return allEntries[i].Timestamp.After(allEntries[j].Timestamp)
	})

	if len(allEntries) > maxResults {
		allEntries = allEntries[:maxResults]
	}

	c.JSON(http.StatusOK, gin.H{
		"collection": collectionName,
		"entries":    allEntries,
		"total":      len(allEntries),
		"query":      "collection cleanup",
	})
}

// searchInFile searches for query in a single log file
func (s *AdminServer) searchInFile(filePath, query string, maxResults int) []LogEntry {
	var entries []LogEntry

	file, err := os.Open(filePath)
	if err != nil {
		return entries
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if strings.Contains(strings.ToLower(line), strings.ToLower(query)) {
			// Parse log entry
			entry := s.parseLogLine(line, filePath, lineNum)
			if entry.Timestamp.IsZero() {
				continue // Skip invalid lines
			}
			entries = append(entries, entry)

			if len(entries) >= maxResults {
				break
			}
		}
	}

	return entries
}

// searchCollectionCleanupLogs searches for collection cleanup specific logs
func (s *AdminServer) searchCollectionCleanupLogs(query, collectionName string, maxResults int) []LogEntry {
	var entries []LogEntry

	// Get log directory
	logDir := os.TempDir()
	if os.Getenv("LOG_DIR") != "" {
		logDir = os.Getenv("LOG_DIR")
	}

	// Search in all log files
	patterns := []string{
		"weed.INFO.*",
		"weed.WARNING.*",
		"weed.ERROR.*",
		"weed.FATAL.*",
		"weed.*.INFO.*",
		"weed.*.WARNING.*",
		"weed.*.ERROR.*",
		"weed.*.FATAL.*",
	}

	for _, pattern := range patterns {
		files, err := filepath.Glob(filepath.Join(logDir, pattern))
		if err != nil {
			continue
		}

		// Search in most recent files
		if len(files) > 0 {
			sort.Slice(files, func(i, j int) bool {
				statI, _ := os.Stat(files[i])
				statJ, _ := os.Stat(files[j])
				return statI.ModTime().After(statJ.ModTime())
			})

			// Search in top 3 most recent files
			maxFiles := 3
			if len(files) < maxFiles {
				maxFiles = len(files)
			}

			for i := 0; i < maxFiles; i++ {
				fileEntries := s.searchInFile(files[i], query, maxResults/maxFiles)
				entries = append(entries, fileEntries...)
			}
		}
	}

	return entries
}

// parseLogLine parses a single log line into LogEntry
func (s *AdminServer) parseLogLine(line, filename string, lineNum int) LogEntry {
	entry := LogEntry{
		File: filename,
		Line: lineNum,
	}

	// Try to parse glog format: I2025/12/08 10:30:00.123456 12345 file.go:123] message
	parts := strings.SplitN(line, "]", 2)
	if len(parts) < 2 {
		return entry
	}

	header := strings.TrimSpace(parts[0])
	message := strings.TrimSpace(parts[1])

	// Extract level from first character
	if len(header) > 0 {
		levelChar := header[0]
		switch levelChar {
		case 'I':
			entry.Level = "INFO"
		case 'W':
			entry.Level = "WARNING"
		case 'E':
			entry.Level = "ERROR"
		case 'F':
			entry.Level = "FATAL"
		default:
			entry.Level = "UNKNOWN"
		}
	}

	// Extract timestamp
	if len(header) > 2 {
		timestampStr := header[1:] // Remove level character
		if timestamp, err := time.Parse("2006/01/02 15:04:05.999999", timestampStr[:26]); err == nil {
			entry.Timestamp = timestamp
		}
	}

	entry.Message = message
	return entry
}
