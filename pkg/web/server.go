package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/pterm/pterm"
	"github.com/tosih/ecu-reader/pkg/models"
	"github.com/tosih/ecu-reader/pkg/reader"
)

//go:embed templates/*
var templates embed.FS

type MapResponse struct {
	Name   string      `json:"name"`
	Offset int64       `json:"offset"`
	Rows   int         `json:"rows"`
	Cols   int         `json:"cols"`
	Unit   string      `json:"unit"`
	Data   [][]float64 `json:"data"`
}

type Server struct {
	filename  string
	filename2 string
	port      int
}

func NewServer(filename string, port int) *Server {
	return &Server{
		filename:  filename,
		filename2: "",
		port:      port,
	}
}

func NewCompareServer(filename1, filename2 string, port int) *Server {
	return &Server{
		filename:  filename1,
		filename2: filename2,
		port:      port,
	}
}

func (s *Server) Start() error {
	http.HandleFunc("/", s.handleIndex)
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

	// Read the map
	ecuMap, err := reader.ReadMap(s.filename, cfg)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading map: %v", err), http.StatusInternalServerError)
		return
	}

	response := MapResponse{
		Name:   cfg.Name,
		Offset: cfg.Offset,
		Rows:   cfg.Rows,
		Cols:   cfg.Cols,
		Unit:   cfg.Unit,
		Data:   ecuMap.Data,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleMode(w http.ResponseWriter, r *http.Request) {
	mode := "single"
	if s.filename2 != "" {
		mode = "compare"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"mode": mode,
		"file1": s.filename,
		"file2": s.filename2,
	})
}

type CompareResponse struct {
	Name   string      `json:"name"`
	Offset int64       `json:"offset"`
	Rows   int         `json:"rows"`
	Cols   int         `json:"cols"`
	Unit   string      `json:"unit"`
	Data1  [][]float64 `json:"data1"`
	Data2  [][]float64 `json:"data2"`
	Diff   [][]float64 `json:"diff"`
}

func (s *Server) handleCompareData(w http.ResponseWriter, r *http.Request) {
	if s.filename2 == "" {
		http.Error(w, "Comparison mode not enabled", http.StatusBadRequest)
		return
	}

	// Extract map index from URL path
	idxStr := r.URL.Path[len("/api/compare/"):]
	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 0 || idx >= len(models.MapConfigs) {
		http.Error(w, "Invalid map index", http.StatusBadRequest)
		return
	}

	cfg := models.MapConfigs[idx]

	// Read both maps
	ecuMap1, err1 := reader.ReadMap(s.filename, cfg)
	ecuMap2, err2 := reader.ReadMap(s.filename2, cfg)

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
		Name:   cfg.Name,
		Offset: cfg.Offset,
		Rows:   cfg.Rows,
		Cols:   cfg.Cols,
		Unit:   cfg.Unit,
		Data1:  ecuMap1.Data,
		Data2:  ecuMap2.Data,
		Diff:   diff,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
