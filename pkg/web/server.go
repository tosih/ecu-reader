package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/tosih/motronic-m21-tool/pkg/models"
	"github.com/tosih/motronic-m21-tool/pkg/reader"
)

//go:embed templates/*
var templates embed.FS

type MapResponse struct {
	Name     string      `json:"name"`
	Offset   int64       `json:"offset"`
	Rows     int         `json:"rows"`
	Cols     int         `json:"cols"`
	Unit     string      `json:"unit"`
	Data     [][]float64 `json:"data"`
	Filename string      `json:"filename"`
}

type Server struct {
	binFolder string
	binFiles  []string
	port      int
}

func NewServer(filename string, port int) *Server {
	// If filename is a directory, use it as binFolder
	// If it's a file, use its directory as binFolder
	var binFolder string
	fileInfo, err := os.Stat(filename)
	if err == nil && fileInfo.IsDir() {
		binFolder = filename
	} else {
		binFolder = filepath.Dir(filename)
	}

	// Scan for all .bin files in the folder
	binFiles, err := findBinFiles(binFolder)
	if err != nil {
		pterm.Warning.Printf("Error scanning for bin files: %v\n", err)
		binFiles = []string{}
		if !fileInfo.IsDir() {
			binFiles = append(binFiles, filename)
		}
	}

	if len(binFiles) == 0 {
		pterm.Warning.Println("No .bin files found in directory")
	} else {
		pterm.Info.Printf("Found %d .bin file(s) in %s\n", len(binFiles), binFolder)
	}

	return &Server{
		binFolder: binFolder,
		binFiles:  binFiles,
		port:      port,
	}
}

func NewCompareServer(filename1, filename2 string, port int) *Server {
	// For compare mode, use the directory of the first file
	binFolder := filepath.Dir(filename1)
	binFiles, err := findBinFiles(binFolder)
	if err != nil {
		binFiles = []string{filename1, filename2}
	}

	return &Server{
		binFolder: binFolder,
		binFiles:  binFiles,
		port:      port,
	}
}

func findBinFiles(dir string) ([]string, error) {
	var binFiles []string

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".bin") {
			binFiles = append(binFiles, filepath.Join(dir, file.Name()))
		}
	}

	return binFiles, nil
}

func (s *Server) Start() error {
	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/api/files", s.handleFileList)
	http.HandleFunc("/api/config", s.handleConfigData)
	http.HandleFunc("/api/map/", s.handleMapData)
	http.HandleFunc("/api/compare/", s.handleCompareData)
	http.HandleFunc("/api/mode", s.handleMode)

	addr := fmt.Sprintf(":%d", s.port)
	url := fmt.Sprintf("http://localhost%s", addr)

	pterm.DefaultHeader.WithFullWidth().
		WithBackgroundStyle(pterm.NewStyle(pterm.BgCyan)).
		WithTextStyle(pterm.NewStyle(pterm.FgBlack)).
		Println("🌐 ECU Web Viewer Started")

	pterm.Info.Printf("Opening web interface at %s\n", url)
	pterm.Info.Println("Press Ctrl+C to stop the server")
	pterm.Println()

	// Try to open browser
	openBrowser(url)

	server := &http.Server{
		Addr:         addr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return server.ListenAndServe()
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	content, err := templates.ReadFile("templates/index.html")
	if err != nil {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(content)
}

func (s *Server) handleFileList(w http.ResponseWriter, r *http.Request) {
	fileList := make([]map[string]string, len(s.binFiles))
	for i, fullPath := range s.binFiles {
		fileList[i] = map[string]string{
			"path": fullPath,
			"name": filepath.Base(fullPath),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fileList)
}

func (s *Server) handleConfigData(w http.ResponseWriter, r *http.Request) {
	// Get filename from query parameter
	filename := r.URL.Query().Get("file")
	if filename == "" {
		if len(s.binFiles) > 0 {
			filename = s.binFiles[0]
		} else {
			http.Error(w, "No bin files available", http.StatusBadRequest)
			return
		}
	}

	// Read config parameters
	config, err := reader.ReadConfigParams(filename)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading config: %v", err), http.StatusInternalServerError)
		return
	}

	// Build response with params and values
	response := map[string]interface{}{
		"params":   config.Params,
		"values":   config.Values,
		"filename": filepath.Base(filename),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleMapData(w http.ResponseWriter, r *http.Request) {
	// Extract map index from URL path
	idxStr := r.URL.Path[len("/api/map/"):]
	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 0 || idx >= len(models.MapConfigs) {
		http.Error(w, "Invalid map index", http.StatusBadRequest)
		return
	}

	cfg := models.MapConfigs[idx]

	// Check for custom offset in query parameters
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		var customOffset int64
		if _, err := fmt.Sscanf(offsetStr, "0x%x", &customOffset); err == nil {
			cfg.Offset = customOffset
		}
	}

	// Get filename from query parameter, or use first file
	filename := r.URL.Query().Get("file")
	if filename == "" {
		if len(s.binFiles) > 0 {
			filename = s.binFiles[0]
		} else {
			http.Error(w, "No bin files available", http.StatusBadRequest)
			return
		}
	}

	// Read the map
	ecuMap, err := reader.ReadMap(filename, cfg)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading map: %v", err), http.StatusInternalServerError)
		return
	}

	response := MapResponse{
		Name:     cfg.Name,
		Offset:   cfg.Offset,
		Rows:     cfg.Rows,
		Cols:     cfg.Cols,
		Unit:     cfg.Unit,
		Data:     ecuMap.Data,
		Filename: filepath.Base(filename),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleMode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"mode":      "multi",
		"binFolder": s.binFolder,
		"fileCount": len(s.binFiles),
	})
}

type CompareResponse struct {
	Name      string      `json:"name"`
	Offset    int64       `json:"offset"`
	Rows      int         `json:"rows"`
	Cols      int         `json:"cols"`
	Unit      string      `json:"unit"`
	Data1     [][]float64 `json:"data1"`
	Data2     [][]float64 `json:"data2"`
	Diff      [][]float64 `json:"diff"`
	Filename1 string      `json:"filename1"`
	Filename2 string      `json:"filename2"`
}

func (s *Server) handleCompareData(w http.ResponseWriter, r *http.Request) {
	// Extract map index from URL path
	idxStr := r.URL.Path[len("/api/compare/"):]
	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 0 || idx >= len(models.MapConfigs) {
		http.Error(w, "Invalid map index", http.StatusBadRequest)
		return
	}

	// Get filenames from query parameters
	file1 := r.URL.Query().Get("file1")
	file2 := r.URL.Query().Get("file2")

	if file1 == "" || file2 == "" {
		http.Error(w, "Both file1 and file2 parameters required", http.StatusBadRequest)
		return
	}

	cfg := models.MapConfigs[idx]

	// Read both maps
	ecuMap1, err1 := reader.ReadMap(file1, cfg)
	ecuMap2, err2 := reader.ReadMap(file2, cfg)

	if err1 != nil || err2 != nil {
		http.Error(w, fmt.Sprintf("Error reading maps: %v, %v", err1, err2), http.StatusInternalServerError)
		return
	}

	// Calculate differences
	diff := make([][]float64, cfg.Rows)
	for i := 0; i < cfg.Rows; i++ {
		diff[i] = make([]float64, cfg.Cols)
		for j := 0; j < cfg.Cols; j++ {
			diff[i][j] = ecuMap2.Data[i][j] - ecuMap1.Data[i][j]
		}
	}

	response := CompareResponse{
		Name:      cfg.Name,
		Offset:    cfg.Offset,
		Rows:      cfg.Rows,
		Cols:      cfg.Cols,
		Unit:      cfg.Unit,
		Data1:     ecuMap1.Data,
		Data2:     ecuMap2.Data,
		Diff:      diff,
		Filename1: filepath.Base(file1),
		Filename2: filepath.Base(file2),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
