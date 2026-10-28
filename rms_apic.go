package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	serverPort = ":8090"

	// ============================================================
	// BHAVCOPY DIRECTORIES
	// ============================================================
	nseEQBhavcopyDir = "/home/rsapp/Rupee_GO_Service/MAINTAIN_ALL_FILES/UPLOADS/BHAVCOPY/NSE/EQUITY"
	nseFOBhavcopyDir = "/home/rsapp/Rupee_GO_Service/MAINTAIN_ALL_FILES/UPLOADS/BHAVCOPY/NSE/DERIVATIVE"

	bseEQBhavcopyDir = "/home/rsapp/Rupee_GO_Service/MAINTAIN_ALL_FILES/UPLOADS/BHAVCOPY/BSE/EQUITY"
	bseFOBhavcopyDir = "/home/rsapp/Rupee_GO_Service/MAINTAIN_ALL_FILES/UPLOADS/BHAVCOPY/BSE/DERIVATIVE"

	// ============================================================
	// DSPAN
	// ============================================================
	// DSPAN folder only.
	// The program will search for:
	//
	//     nsccl.2026*.spn
	//
	// ============================================================
	dspanDir = "/home/rsapp/Rupee_GO_Service/MAINTAIN_ALL_FILES/UPLOADS/SPAN/DSPAN"

	// ============================================================
	// MCX SPAN
	// ============================================================
	mcxSPANDir = "/home/rsapp/Rupee_GO_Service/MAINTAIN_ALL_FILES/UPLOADS/SPAN/MSPAN"

	// ============================================================
	// MCX EXPOSURE
	// ============================================================
	mcxExposureDir = "/home/rsapp/Rupee_GO_Service/MAINTAIN_ALL_FILES/UPLOADS/EXP_MARGIN/MCX"

	// ============================================================
	// EXPOSURE
	// ============================================================
	exposureBaseDir = "/home/rsapp/Rupee_GO_Service/MAINTAIN_ALL_FILES/UPLOADS/EXP_MARGIN/NSE"
)

// ============================================================
// COMMON HTTP HELPERS
// ============================================================

func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "http://172.16.101.74:8080")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func methodAllowed(w http.ResponseWriter, r *http.Request) bool {

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return false
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return false
	}

	return true
}

// ============================================================
// LATEST CSV FILE HELPER
// ============================================================

func latestCSV(dir string) (string, fs.FileInfo, error) {

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", nil, err
	}

	type item struct {
		path string
		info fs.FileInfo
	}

	var files []item

	for _, entry := range entries {

		if entry.IsDir() {
			continue
		}

		if !strings.EqualFold(filepath.Ext(entry.Name()), ".csv") {
			continue
		}

		path := filepath.Join(dir, entry.Name())

		info, err := entry.Info()
		if err != nil {
			continue
		}

		files = append(files, item{
			path: path,
			info: info,
		})
	}

	if len(files) == 0 {
		return "", nil, fmt.Errorf("no CSV file found in %s", dir)
	}

	// Latest modification time first.
	// If timestamps are equal, latest filename wins.
	sort.Slice(files, func(i, j int) bool {

		if files[i].info.ModTime().Equal(files[j].info.ModTime()) {
			return files[i].info.Name() > files[j].info.Name()
		}

		return files[i].info.ModTime().After(files[j].info.ModTime())
	})

	return files[0].path, files[0].info, nil
}

func fileInfoJSON(info fs.FileInfo) map[string]interface{} {

	return map[string]interface{}{
		"name":         info.Name(),
		"modified":     info.ModTime().Format(time.RFC3339),
		"modifiedUnix": info.ModTime().Unix(),
		"size":         info.Size(),
	}
}

// ============================================================
// HEALTH
// ============================================================

func apiHealth(w http.ResponseWriter, r *http.Request) {

	setCORS(w)

	if !methodAllowed(w, r) {
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// ============================================================
// GENERIC BHAVCOPY HANDLER
// ============================================================

func serveLatestBhavcopy(
	w http.ResponseWriter,
	r *http.Request,
	dir string,
	label string,
) {

	path, info, err := latestCSV(dir)

	if err != nil {
		http.Error(
			w,
			label+" Bhavcopy not found: "+err.Error(),
			http.StatusNotFound,
		)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")

	w.Header().Set(
		"Content-Disposition",
		`inline; filename="`+info.Name()+`"`,
	)

	http.ServeFile(w, r, path)
}

func serveBhavcopyInfo(
	w http.ResponseWriter,
	dir string,
	label string,
) {

	_, info, err := latestCSV(dir)

	if err != nil {
		http.Error(
			w,
			label+" Bhavcopy not found: "+err.Error(),
			http.StatusNotFound,
		)
		return
	}

	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	_ = json.NewEncoder(w).Encode(fileInfoJSON(info))
}

// ============================================================
// NSE EQUITY BHAVCOPY
// ============================================================

func apiNSEEQBavcopy(w http.ResponseWriter, r *http.Request) {

	setCORS(w)

	if !methodAllowed(w, r) {
		return
	}

	serveLatestBhavcopy(
		w,
		r,
		nseEQBhavcopyDir,
		"NSE EQ",
	)
}

func apiNSEEQBavcopyInfo(w http.ResponseWriter, r *http.Request) {

	setCORS(w)

	if !methodAllowed(w, r) {
		return
	}

	serveBhavcopyInfo(
		w,
		nseEQBhavcopyDir,
		"NSE EQ",
	)
}

// ============================================================
// NSE F&O BHAVCOPY
// ============================================================

func apiNSEFOBhavcopy(w http.ResponseWriter, r *http.Request) {

	setCORS(w)

	if !methodAllowed(w, r) {
		return
	}

	serveLatestBhavcopy(
		w,
		r,
		nseFOBhavcopyDir,
		"NSE FO",
	)
}

func apiNSEFOBhavcopyInfo(w http.ResponseWriter, r *http.Request) {

	setCORS(w)

	if !methodAllowed(w, r) {
		return
	}

	serveBhavcopyInfo(
		w,
		nseFOBhavcopyDir,
		"NSE FO",
	)
}

// ============================================================
// BSE EQUITY BHAVCOPY
// ============================================================

func apiBSEEQBavcopy(w http.ResponseWriter, r *http.Request) {

	setCORS(w)

	if !methodAllowed(w, r) {
		return
	}

	serveLatestBhavcopy(
		w,
		r,
		bseEQBhavcopyDir,
		"BSE EQ",
	)
}

func apiBSEEQBavcopyInfo(w http.ResponseWriter, r *http.Request) {

	setCORS(w)

	if !methodAllowed(w, r) {
		return
	}

	serveBhavcopyInfo(
		w,
		bseEQBhavcopyDir,
		"BSE EQ",
	)
}

// ============================================================
// BSE F&O BHAVCOPY
// ============================================================

func apiBSEFOBavcopy(w http.ResponseWriter, r *http.Request) {

	setCORS(w)

	if !methodAllowed(w, r) {
		return
	}

	serveLatestBhavcopy(
		w,
		r,
		bseFOBhavcopyDir,
		"BSE FO",
	)
}

func apiBSEFOBhavcopyInfo(w http.ResponseWriter, r *http.Request) {

	setCORS(w)

	if !methodAllowed(w, r) {
		return
	}

	serveBhavcopyInfo(
		w,
		bseFOBhavcopyDir,
		"BSE FO",
	)
}

// ============================================================
// DSPAN
// ============================================================

// findDSPANFile
//
// Searches ONLY inside dspanDir.
//
// Matching rule:
//
//     nsccl.2026*.spn
//
// Example matches:
//
//     nsccl.20261001.i03.spn
//     nsccl.20261006.i03.spn
//     nsccl.20261231.i03.spn
//
// It will NOT match:
//
//     nsccl.2025....spn
//     abc.2026....spn
//     test.spn
//
// If multiple matching files exist, the most recently modified
// matching file is selected.

func findDSPANFile() (string, fs.FileInfo, error) {

	entries, err := os.ReadDir(dspanDir)

	if err != nil {
		return "", nil, err
	}

	type item struct {
		path string
		info fs.FileInfo
	}

	var files []item

	for _, entry := range entries {

		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// --------------------------------------------------------
		// MUST START WITH nsccl.2026
		// --------------------------------------------------------
		if !strings.HasPrefix(
			strings.ToLower(name),
			"nsccl.2026",
		) {
			continue
		}

		// --------------------------------------------------------
		// MUST BE .spn
		// --------------------------------------------------------
		if !strings.EqualFold(
			filepath.Ext(name),
			".spn",
		) {
			continue
		}

		path := filepath.Join(dspanDir, name)

		info, err := entry.Info()
		if err != nil {
			continue
		}

		files = append(files, item{
			path: path,
			info: info,
		})
	}

	// ------------------------------------------------------------
	// NO MATCHING FILE
	// ------------------------------------------------------------

	if len(files) == 0 {

		return "",
			nil,
			fmt.Errorf(
				"no DSPAN file matching nsccl.2026*.spn found in %s",
				dspanDir,
			)
	}

	// ------------------------------------------------------------
	// IF MULTIPLE MATCHING FILES EXIST
	// USE MOST RECENTLY MODIFIED
	// ------------------------------------------------------------

	sort.Slice(files, func(i, j int) bool {

		if files[i].info.ModTime().Equal(
			files[j].info.ModTime(),
		) {
			return files[i].info.Name() >
				files[j].info.Name()
		}

		return files[i].info.ModTime().After(
			files[j].info.ModTime(),
		)
	})

	return files[0].path, files[0].info, nil
}

// ============================================================
// DSPAN FILE API
// ============================================================

func apiDSPAN(w http.ResponseWriter, r *http.Request) {

	setCORS(w)

	if !methodAllowed(w, r) {
		return
	}

	// ------------------------------------------------------------
	// FIND nsccl.2026*.spn FROM LINUX
	// ------------------------------------------------------------

	path, info, err := findDSPANFile()

	if err != nil {

		http.Error(
			w,
			"DSPAN file not found: "+err.Error(),
			http.StatusNotFound,
		)

		return
	}

	// ------------------------------------------------------------
	// SERVE FILE
	// ------------------------------------------------------------

	w.Header().Set(
		"Content-Type",
		"text/plain; charset=utf-8",
	)

	w.Header().Set(
		"Content-Disposition",
		`inline; filename="`+info.Name()+`"`,
	)

	http.ServeFile(
		w,
		r,
		path,
	)
}

// ============================================================
// DSPAN INFO API
// ============================================================

func apiDSPANInfo(w http.ResponseWriter, r *http.Request) {

	setCORS(w)

	if !methodAllowed(w, r) {
		return
	}

	// ------------------------------------------------------------
	// FIND nsccl.2026*.spn FROM LINUX
	// ------------------------------------------------------------

	_, info, err := findDSPANFile()

	if err != nil {

		http.Error(
			w,
			"DSPAN file not found: "+err.Error(),
			http.StatusNotFound,
		)

		return
	}

	// ------------------------------------------------------------
	// RETURN FILE INFORMATION
	// ------------------------------------------------------------

	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	_ = json.NewEncoder(w).Encode(
		fileInfoJSON(info),
	)
}

// ============================================================
// MCX SPAN
// ============================================================

func latestSPANFile(dir string) (string, fs.FileInfo, error) {

	entries, err := os.ReadDir(dir)

	if err != nil {
		return "", nil, err
	}

	type item struct {
		path string
		info fs.FileInfo
	}

	var files []item

	for _, entry := range entries {

		if entry.IsDir() {
			continue
		}

		if !strings.EqualFold(
			filepath.Ext(entry.Name()),
			".spn",
		) {
			continue
		}

		path := filepath.Join(
			dir,
			entry.Name(),
		)

		info, err := entry.Info()

		if err != nil {
			continue
		}

		files = append(files, item{
			path: path,
			info: info,
		})
	}

	if len(files) == 0 {

		return "",
			nil,
			fmt.Errorf(
				"no SPN file found in %s",
				dir,
			)
	}

	sort.Slice(files, func(i, j int) bool {

		if files[i].info.ModTime().Equal(
			files[j].info.ModTime(),
		) {

			return files[i].info.Name() >
				files[j].info.Name()
		}

		return files[i].info.ModTime().After(
			files[j].info.ModTime(),
		)
	})

	return files[0].path, files[0].info, nil
}

func apiMCXSPAN(w http.ResponseWriter, r *http.Request) {

	setCORS(w)

	if !methodAllowed(w, r) {
		return
	}

	path, info, err := latestSPANFile(
		mcxSPANDir,
	)

	if err != nil {

		http.Error(
			w,
			"MCX SPAN file not found: "+err.Error(),
			http.StatusNotFound,
		)

		return
	}

	w.Header().Set(
		"Content-Type",
		"text/plain; charset=utf-8",
	)

	w.Header().Set(
		"Content-Disposition",
		`inline; filename="`+info.Name()+`"`,
	)

	http.ServeFile(
		w,
		r,
		path,
	)
}

func apiMCXSPANInfo(w http.ResponseWriter, r *http.Request) {

	setCORS(w)

	if !methodAllowed(w, r) {
		return
	}

	_, info, err := latestSPANFile(
		mcxSPANDir,
	)

	if err != nil {

		http.Error(
			w,
			"MCX SPAN file not found: "+err.Error(),
			http.StatusNotFound,
		)

		return
	}

	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	_ = json.NewEncoder(w).Encode(
		fileInfoJSON(info),
	)
}

// ============================================================
// MCX EXPOSURE
// ============================================================

func latestMCXExposure() (string, fs.FileInfo, error) {

	return latestCSV(
		mcxExposureDir,
	)
}

func apiMCXExposureInfo(w http.ResponseWriter, r *http.Request) {

	setCORS(w)

	if !methodAllowed(w, r) {
		return
	}

	_, info, err := latestMCXExposure()

	if err != nil {

		http.Error(
			w,
			"MCX Exposure file not found: "+err.Error(),
			http.StatusNotFound,
		)

		return
	}

	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	_ = json.NewEncoder(w).Encode(
		fileInfoJSON(info),
	)
}

func apiMCXExposure(w http.ResponseWriter, r *http.Request) {

	setCORS(w)

	if !methodAllowed(w, r) {
		return
	}

	path, info, err := latestMCXExposure()

	if err != nil {

		http.Error(
			w,
			"MCX Exposure file not found: "+err.Error(),
			http.StatusNotFound,
		)

		return
	}

	w.Header().Set(
		"Content-Type",
		"text/csv; charset=utf-8",
	)

	w.Header().Set(
		"Content-Disposition",
		`inline; filename="`+info.Name()+`"`,
	)

	http.ServeFile(
		w,
		r,
		path,
	)
}

// ============================================================
// EXPOSURE
// ============================================================

type exposureFile struct {
	path string
	info fs.FileInfo
}

func getExposureFile(
	folder string,
) (exposureFile, error) {

	path, info, err := latestCSV(
		filepath.Join(
			exposureBaseDir,
			folder,
		),
	)

	if err != nil {
		return exposureFile{}, err
	}

	return exposureFile{
		path: path,
		info: info,
	}, nil
}

func getAllExposureFiles() (
	map[string]exposureFile,
	error,
) {

	folders := []string{
		"NIFTY",
		"OTM",
		"STK",
	}

	result := make(
		map[string]exposureFile,
		len(folders),
	)

	for _, folder := range folders {

		f, err := getExposureFile(
			folder,
		)

		if err != nil {

			return nil,
				fmt.Errorf(
					"%s: %w",
					folder,
					err,
				)
		}

		result[folder] = f
	}

	return result, nil
}

func apiExposureInfo(w http.ResponseWriter, r *http.Request) {

	setCORS(w)

	if !methodAllowed(w, r) {
		return
	}

	files, err := getAllExposureFiles()

	if err != nil {

		http.Error(
			w,
			"Exposure files not found: "+err.Error(),
			http.StatusNotFound,
		)

		return
	}

	response := map[string]interface{}{
		"nifty": fileInfoJSON(
			files["NIFTY"].info,
		),
		"otm": fileInfoJSON(
			files["OTM"].info,
		),
		"stk": fileInfoJSON(
			files["STK"].info,
		),
	}

	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	_ = json.NewEncoder(w).Encode(
		response,
	)
}

func apiExposure(w http.ResponseWriter, r *http.Request) {

	setCORS(w)

	if !methodAllowed(w, r) {
		return
	}

	part := strings.TrimPrefix(
		r.URL.Path,
		"/api/exposure/",
	)

	var folder string

	switch strings.ToLower(
		strings.TrimSpace(part),
	) {

	case "nifty":
		folder = "NIFTY"

	case "otm":
		folder = "OTM"

	case "stk":
		folder = "STK"

	default:

		http.Error(
			w,
			"Use /api/exposure/nifty, /api/exposure/otm or /api/exposure/stk",
			http.StatusBadRequest,
		)

		return
	}

	file, err := getExposureFile(
		folder,
	)

	if err != nil {

		http.Error(
			w,
			"Exposure "+folder+
				" file not found: "+
				err.Error(),
			http.StatusNotFound,
		)

		return
	}

	w.Header().Set(
		"Content-Type",
		"text/csv; charset=utf-8",
	)

	w.Header().Set(
		"Content-Disposition",
		`inline; filename="`+
			file.info.Name()+
			`"`,
	)

	http.ServeFile(
		w,
		r,
		file.path,
	)
}

// ============================================================
// MARKET CONNECTION MONITOR (proxy to Server 170)
// Configure MARKET_MONITOR_TOKEN in the KSH service environment.
// ============================================================
func apiMarketMonitorStatus(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if !methodAllowed(w, r) {
		return
	}

	token := strings.TrimSpace(os.Getenv("MARKET_MONITOR_TOKEN"))
	if token == "" {
		http.Error(w, `{"error":"Market monitoring token is not configured on Server 74"}`, http.StatusServiceUnavailable)
		return
	}

	endpoint := strings.TrimSpace(os.Getenv("MARKET_MONITOR_URL"))
	if endpoint == "" {
		endpoint = "http://172.16.101.170:8095/api/monitoring/status"
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		http.Error(w, `{"error":"Invalid monitoring URL"}`, http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Cannot reach monitoring service on Server 170"})
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// ============================================================
// MARKET CONNECTION MONITOR (proxy to Server 168)
// Configure MARKET_MONITOR_168_TOKEN in the KSH service environment.
// ============================================================
func apiMarketMonitor168Status(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if !methodAllowed(w, r) {
		return
	}

	token := strings.TrimSpace(os.Getenv("MARKET_MONITOR_168_TOKEN"))
	if token == "" {
		http.Error(w, `{"error":"Server 168 monitoring token is not configured on Server 74"}`, http.StatusServiceUnavailable)
		return
	}

	// Mirror the Server 170 proxy pattern, but keep Server 168's own URL/token.
	// This allows the endpoint to be corrected through the existing service environment
	// without changing the HTML or rebuilding the binary.
	endpoint := strings.TrimSpace(os.Getenv("MARKET_MONITOR_168_URL"))
	if endpoint == "" {
		endpoint = "http://172.16.101.168:8095/api/monitoring/status"
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		http.Error(w, `{"error":"Invalid Server 168 monitoring URL"}`, http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "Cannot reach monitoring service on Server 168",
		})
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// ============================================================
// LOCAL MARKET CONNECTION STATUS (Server 74, full status)
// Runs the existing read-only utility from its required directory.
// ============================================================
func apiLocalMarketMonitorStatus(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if !methodAllowed(w, r) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Start a login shell before creating the pseudo-terminal. The legacy C
	// utility behaves differently in a bare service environment; a login shell
	// loads the rsapp account's normal environment without changing systemd.
	// script provides the terminal context expected by the utility.
	command := "cd /home/rsapp/Run && exec /usr/bin/script -q -c './ShowConnectStatus' /dev/null"
	cmd := exec.CommandContext(ctx, "/bin/bash", "-lc", command)
	cmd.Dir = "/home/rsapp/Run"
	output, err := cmd.CombinedOutput()

	// The utility has been observed to return its status table with exit code 1.
	// Accept a complete table regardless of exit status; never hide real failures.
	if !strings.Contains(string(output), "Group Id") {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)
		message := "ShowConnectStatus did not return a status table"
		if err != nil {
			message += ": " + err.Error()
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":  message,
			"output": strings.TrimSpace(string(output)),
		})
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(output)
}

// ============================================================
// SIMPLE READ-ONLY EXCHANGE FILE STATUS
// Uses the existing mysql command-line client; no Go module/driver needed.
// Reads DB74_*, DB168_* and DB166_* from the existing KSH/db.env file.
// ============================================================
func loadProcessLogDBEnv() error {
	path := "/home/rsapp/Application/Web_host/KSH/db.env"
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == 39 && value[len(value)-1] == 39)) {
			value = value[1 : len(value)-1]
		}
		if key != "" {
			_ = os.Setenv(key, value)
		}
	}
	return nil
}

func apiMarketMonitorProcessLog(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	if err := loadProcessLogDBEnv(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"records": []interface{}{}, "errors": []map[string]string{{"server": "configuration", "error": "Cannot read KSH/db.env"}}})
		return
	}
	const query = `SELECT
    COUNT(*) AS TOTAL_COUNT,
    COALESCE(SUM(CASE WHEN SUCCESS_FLAG = 'Y' THEN 1 ELSE 0 END), 0) AS SUCCESS_COUNT,
    COALESCE(SUM(CASE WHEN SUCCESS_FLAG <> 'Y' OR SUCCESS_FLAG IS NULL THEN 1 ELSE 0 END), 0) AS FAIL_COUNT
FROM RMS_PROCESS_LOG
WHERE PROCESS_DATE >= CURDATE()
  AND PROCESS_DATE < CURDATE() + INTERVAL 1 DAY;`
	type record map[string]interface{}
	records := []record{}
	errors := []map[string]string{}
	for _, server := range []string{"74", "168", "166"} {
		prefix := "DB" + server + "_"
		host, port, name, user, pass := os.Getenv(prefix+"HOST"), os.Getenv(prefix+"PORT"), os.Getenv(prefix+"NAME"), os.Getenv(prefix+"USER"), os.Getenv(prefix+"PASSWORD")
		if port == "" {
			port = "3306"
		}
		if host == "" || name == "" || user == "" || pass == "" {
			errors = append(errors, map[string]string{"server": server, "error": "DB settings missing in KSH/db.env"})
			continue
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		cmd := exec.CommandContext(ctx, "mysql", "--batch", "--raw", "--skip-column-names", "--connect-timeout=5", "-h", host, "-P", port, "-u", user, name, "-e", query)
		cmd.Env = append(os.Environ(), "MYSQL_PWD="+pass)
		output, err := cmd.CombinedOutput()
		cancel()
		if err != nil {
			errors = append(errors, map[string]string{"server": server, "error": "MySQL query failed; check connection, credentials, and RMS_PROCESS_LOG access"})
			continue
		}
		fields := strings.Split(strings.TrimSpace(string(output)), "\t")
		if len(fields) != 3 {
			errors = append(errors, map[string]string{"server": server, "error": "Unexpected count query result"})
			continue
		}
		records = append(records, record{
			"server":        server,
			"total_count":   strings.TrimSpace(fields[0]),
			"success_count": strings.TrimSpace(fields[1]),
			"fail_count":    strings.TrimSpace(fields[2]),
		})
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"records": records, "errors": errors})
}

// Return today's failed/incomplete RMS_PROCESS_LOG rows for one configured server.
func apiMarketMonitorProcessLogFailures(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	if err := loadProcessLogDBEnv(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"records": []interface{}{}, "error": "Cannot read KSH/db.env"})
		return
	}
	server := r.URL.Query().Get("server")
	if server != "74" && server != "166" && server != "168" {
		http.Error(w, `{"error":"Use server=74, 166 or 168"}`, http.StatusBadRequest)
		return
	}
	prefix := "DB" + server + "_"
	host, port, name, user, pass := os.Getenv(prefix+"HOST"), os.Getenv(prefix+"PORT"), os.Getenv(prefix+"NAME"), os.Getenv(prefix+"USER"), os.Getenv(prefix+"PASSWORD")
	if port == "" {
		port = "3306"
	}
	if host == "" || name == "" || user == "" || pass == "" {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"records": []interface{}{}, "error": "DB settings missing in KSH/db.env for server " + server})
		return
	}
	query := `SELECT PROCESS_DATE, EXCH_ID, SEGMENT, PROCESS, SUCCESS_FLAG, PROCESS_STATUS, SUCCESS, REJECTED, UPLOAD_FILE_NAME, FILE_NAME FROM RMS_PROCESS_LOG WHERE PROCESS_DATE >= CURDATE() AND PROCESS_DATE < CURDATE() + INTERVAL 1 DAY AND (SUCCESS_FLAG <> 'Y' OR SUCCESS_FLAG IS NULL) ORDER BY PROCESS_DATE DESC`
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "mysql", "--batch", "--raw", "--skip-column-names", "--connect-timeout=5", "-h", host, "-P", port, "-u", user, name, "-e", query)
	cmd.Env = append(os.Environ(), "MYSQL_PWD="+pass)
	output, err := cmd.CombinedOutput()
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"records": []interface{}{}, "error": "MySQL query failed for server " + server + "; check connection, credentials, and table permissions"})
		return
	}
	keys := []string{"process_date", "exch_id", "segment", "process", "success_flag", "process_status", "success", "rejected", "upload_file_name", "file_name"}
	records := []map[string]interface{}{}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != len(keys) {
			continue
		}
		row := map[string]interface{}{"server": server}
		for i, k := range keys {
			if fields[i] == "NULL" {
				row[k] = nil
			} else {
				row[k] = fields[i]
			}
		}
		records = append(records, row)
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"records": records})
}

// ============================================================
// MAIN
// ============================================================

func main() {

	// ============================================================
	// HEALTH
	// ============================================================

	http.HandleFunc("/api/market-monitor/status", apiMarketMonitorStatus)
	http.HandleFunc("/api/market-monitor/168/status", apiMarketMonitor168Status)
	http.HandleFunc("/api/local-market-monitor/status", apiLocalMarketMonitorStatus)
	http.HandleFunc("/api/market-monitor/process-log", apiMarketMonitorProcessLog)
	http.HandleFunc("/api/market-monitor/process-log/failures", apiMarketMonitorProcessLogFailures)

	// DB Query page (additive routes; existing handlers remain unchanged).
	http.HandleFunc("/api/db-query/connect", apiDBQueryConnect)
	http.HandleFunc("/api/db-query/data", apiDBQueryData)
	http.HandleFunc("/api/db-query/library", apiDBQueryLibrary)
	http.HandleFunc("/api/db-query/manual", apiDBQueryManual)
	http.HandleFunc("/api/db-query/update-expiry", apiDBQueryUpdateExpiry)
	http.HandleFunc("/api/db-query/update-ltp", apiDBQueryUpdateLTP)

	http.HandleFunc(
		"/api/health",
		apiHealth,
	)

	// ============================================================
	// BHAVCOPY
	// ============================================================

	// NSE Equity
	http.HandleFunc(
		"/api/nse-eq-bhavcopy",
		apiNSEEQBavcopy,
	)

	http.HandleFunc(
		"/api/nse-eq-bhavcopy-info",
		apiNSEEQBavcopyInfo,
	)

	// NSE Futures & Options
	http.HandleFunc(
		"/api/nse-fo-bhavcopy",
		apiNSEFOBhavcopy,
	)

	http.HandleFunc(
		"/api/nse-fo-bhavcopy-info",
		apiNSEFOBhavcopyInfo,
	)

	// BSE Equity
	http.HandleFunc(
		"/api/bse-eq-bhavcopy",
		apiBSEEQBavcopy,
	)

	http.HandleFunc(
		"/api/bse-eq-bhavcopy-info",
		apiBSEEQBavcopyInfo,
	)

	// BSE Futures & Options
	http.HandleFunc(
		"/api/bse-fo-bhavcopy",
		apiBSEFOBavcopy,
	)

	http.HandleFunc(
		"/api/bse-fo-bhavcopy-info",
		apiBSEFOBhavcopyInfo,
	)

	// ============================================================
	// DSPAN
	// ============================================================

	http.HandleFunc(
		"/api/dspan-info",
		apiDSPANInfo,
	)

	http.HandleFunc(
		"/api/dspan",
		apiDSPAN,
	)

	// ============================================================
	// MCX SPAN
	// ============================================================

	http.HandleFunc(
		"/api/mcx-span-info",
		apiMCXSPANInfo,
	)

	http.HandleFunc(
		"/api/mcx-span",
		apiMCXSPAN,
	)

	http.HandleFunc(
		"/api/mcx-exposure-info",
		apiMCXExposureInfo,
	)

	http.HandleFunc(
		"/api/mcx-exposure",
		apiMCXExposure,
	)

	// ============================================================
	// EXPOSURE
	// ============================================================

	http.HandleFunc(
		"/api/exposure-info",
		apiExposureInfo,
	)

	http.HandleFunc(
		"/api/exposure/",
		apiExposure,
	)

	// ============================================================
	// KSH WEB FILES
	// ============================================================

	// Keep db.env in KSH as requested, but never serve it to browsers.
	http.HandleFunc("/KSH/db.env", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	http.Handle(
		"/KSH/",
		http.StripPrefix(
			"/KSH/",
			http.FileServer(
				http.Dir(
					"/home/rsapp/Application/Web_host/KSH",
				),
			),
		),
	)

	// ============================================================
	// SERVER START
	// ============================================================

	fmt.Println("============================================================")
	fmt.Println(" RMS API SERVER")
	fmt.Println("============================================================")
	fmt.Println("Listening on", serverPort)
	fmt.Println("")

	fmt.Println("KSH WEB:")
	fmt.Println("  GET /KSH/")
	fmt.Println("  Directory: /home/rsapp/Application/Web_host/KSH")
	fmt.Println("")

	fmt.Println("Bhavcopy APIs:")
	fmt.Println("  GET /api/nse-eq-bhavcopy")
	fmt.Println("  GET /api/nse-eq-bhavcopy-info")
	fmt.Println("  GET /api/nse-fo-bhavcopy")
	fmt.Println("  GET /api/nse-fo-bhavcopy-info")
	fmt.Println("  GET /api/bse-eq-bhavcopy")
	fmt.Println("  GET /api/bse-eq-bhavcopy-info")
	fmt.Println("  GET /api/bse-fo-bhavcopy")
	fmt.Println("  GET /api/bse-fo-bhavcopy-info")
	fmt.Println("")

	fmt.Println("DSPAN APIs:")
	fmt.Println("  GET /api/dspan-info")
	fmt.Println("  GET /api/dspan")
	fmt.Println("  Matching: nsccl.2026*.spn")
	fmt.Println("")

	fmt.Println("MCX SPAN APIs:")
	fmt.Println("  GET /api/mcx-span-info")
	fmt.Println("  GET /api/mcx-span")
	fmt.Println("  GET /api/mcx-exposure-info")
	fmt.Println("  GET /api/mcx-exposure")
	fmt.Println("")

	fmt.Println("Exposure APIs:")
	fmt.Println("  GET /api/exposure-info")
	fmt.Println("  GET /api/exposure/nifty")
	fmt.Println("  GET /api/exposure/otm")
	fmt.Println("  GET /api/exposure/stk")
	fmt.Println("")

	fmt.Println("Health:")
	fmt.Println("  GET /api/health")
	fmt.Println("  POST /api/db-query/connect")
	fmt.Println("  GET /api/db-query/data")
	fmt.Println("============================================================")

	if err := http.ListenAndServe(
		serverPort,
		nil,
	); err != nil {

		fmt.Println(
			"Server stopped:",
			err,
		)
	}
}

// ============================================================
// DB QUERY PAGE — ADDITIVE FEATURE ON THE EXISTING :8090 SERVER
// Reads only DB74_*, DB168_* and DB166_* from KSH/db.env.
// Uses the installed mysql CLI, so no new Go module is required.
// ============================================================
const dbQueryEnvPath = "/home/rsapp/Application/Web_host/KSH/db.env"

type dbQuerySession struct {
	Server  string
	Expires time.Time
}

var dbQuerySessions = struct {
	sync.RWMutex
	m map[string]dbQuerySession
}{m: make(map[string]dbQuerySession)}

func dbQueryJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": msg})
}

func dbQueryLoadEnv() error { return loadProcessLogDBEnv() }

func dbQueryConfig(server string) (host, port, name, user, password string, err error) {
	if server != "74" && server != "168" && server != "166" {
		err = fmt.Errorf("Choose Server 74, Server 168 or Server 166")
		return
	}
	p := "DB" + server + "_"
	host, port, name, user, password = os.Getenv(p+"HOST"), os.Getenv(p+"PORT"), os.Getenv(p+"NAME"), os.Getenv(p+"USER"), os.Getenv(p+"PASSWORD")
	if port == "" {
		port = "3306"
	}
	if host == "" || name == "" || user == "" || password == "" {
		err = fmt.Errorf("Database settings are missing for Server %s in KSH/db.env", server)
	}
	return
}

func dbQueryRun(server, query string) ([][]string, error) {
	host, port, name, user, pass, err := dbQueryConfig(server)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "mysql", "--batch", "--raw", "--skip-column-names", "--connect-timeout=5", "-h", host, "-P", port, "-u", user, name, "-e", query)
	cmd.Env = append(os.Environ(), "MYSQL_PWD="+pass)
	output, runErr := cmd.CombinedOutput()
	if runErr != nil {
		return nil, fmt.Errorf("Database query failed; check MySQL client, connection, credentials and table permissions")
	}
	out := strings.TrimSpace(string(output))
	if out == "" {
		return [][]string{}, nil
	}
	rows := make([][]string, 0)
	for _, line := range strings.Split(out, "\n") {
		rows = append(rows, strings.Split(line, "\t"))
	}
	return rows, nil
}

func dbQueryRows(server, query string, keys []string) ([]map[string]interface{}, error) {
	lines, err := dbQueryRun(server, query)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(lines))
	for _, fields := range lines {
		if len(fields) != len(keys) {
			continue
		}
		row := make(map[string]interface{}, len(keys))
		for i, key := range keys {
			if fields[i] == "NULL" {
				row[key] = nil
			} else {
				row[key] = fields[i]
			}
		}
		out = append(out, row)
	}
	return out, nil
}

func apiDBQueryConnect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		dbQueryJSONError(w, http.StatusMethodNotAllowed, "Use POST to connect")
		return
	}
	if err := dbQueryLoadEnv(); err != nil {
		dbQueryJSONError(w, http.StatusInternalServerError, "Cannot read KSH/db.env")
		return
	}
	var req struct {
		DB       string `json:"db"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		dbQueryJSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	server := strings.TrimSpace(req.DB)
	_, _, _, _, expected, err := dbQueryConfig(server)
	if err != nil {
		dbQueryJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Password == "" || req.Password != expected {
		dbQueryJSONError(w, http.StatusUnauthorized, "Incorrect database password")
		return
	}
	if _, err := dbQueryRun(server, "SELECT 1"); err != nil {
		dbQueryJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		dbQueryJSONError(w, http.StatusInternalServerError, "Could not create session")
		return
	}
	token := hex.EncodeToString(b)
	dbQuerySessions.Lock()
	dbQuerySessions.m[token] = dbQuerySession{Server: server, Expires: time.Now().Add(30 * time.Minute)}
	dbQuerySessions.Unlock()
	http.SetCookie(w, &http.Cookie{Name: "rms_db_session", Value: token, Path: "/", HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteStrictMode, MaxAge: 1800})
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "db": server, "message": "Database connected"})
}

func dbQueryCurrentSession(w http.ResponseWriter, r *http.Request) (dbQuerySession, bool) {
	c, err := r.Cookie("rms_db_session")
	if err != nil {
		dbQueryJSONError(w, http.StatusUnauthorized, "Connect to the selected database first")
		return dbQuerySession{}, false
	}
	dbQuerySessions.Lock()
	s, ok := dbQuerySessions.m[c.Value]
	if ok && time.Now().After(s.Expires) {
		delete(dbQuerySessions.m, c.Value)
		ok = false
	}
	dbQuerySessions.Unlock()
	if !ok {
		dbQueryJSONError(w, http.StatusUnauthorized, "Session expired. Connect again.")
		return dbQuerySession{}, false
	}
	return s, true
}

func apiDBQueryData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodGet {
		dbQueryJSONError(w, http.StatusMethodNotAllowed, "Use GET")
		return
	}
	if err := dbQueryLoadEnv(); err != nil {
		dbQueryJSONError(w, http.StatusInternalServerError, "Cannot read KSH/db.env")
		return
	}
	sess, ok := dbQueryCurrentSession(w, r)
	if !ok {
		return
	}
	if r.URL.Query().Get("db") != sess.Server {
		dbQueryJSONError(w, http.StatusForbidden, "Selected server does not match connected session")
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("security_code"))
	if code == "" || len(code) > 40 {
		dbQueryJSONError(w, http.StatusBadRequest, "Enter a valid Security Code")
		return
	}
	// Escape the value for a quoted SQL literal; no user-supplied SQL is accepted.
	code = strings.ReplaceAll(code, "\\", "\\\\")
	code = strings.ReplaceAll(code, "'", "''")
	segment := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("segment")))
	resp := map[string]interface{}{"l1_watch": []interface{}{}, "eq_l1_watch": []interface{}{}, "drv_l1_watch": []interface{}{}, "security_master": []interface{}{}, "fno_span": []interface{}{}, "var_elm": []interface{}{}}
	var rows []map[string]interface{}
	var err error
	if segment == "EQ" {
		rows, err = dbQueryRows(sess.Server, "SELECT L1_SCRIP_CODE,L1_LTP,L1_CLOSE,L1_UPPER_CKT,L1_LOWER_CKT FROM EQ_L1_WATCH WHERE L1_SCRIP_CODE='"+code+"'", []string{"scrip_code", "ltp", "close", "upper_ckt", "lower_ckt"})
		if err != nil {
			dbQueryJSONError(w, 502, "EQ_L1_WATCH query failed; verify table and columns")
			return
		}
		resp["eq_l1_watch"] = rows
		rows, err = dbQueryRows(sess.Server, "SELECT SCRIP_ID,VAR_MARGIN,EXTREME_LOSS_RATE,ADHOC_MARGIN,APPLICABLE_MARGIN_RATE FROM RMS_EQ_VAR_ELM WHERE SCRIP_ID='"+code+"'", []string{"scrip_id", "var_margin", "extreme_loss_rate", "adhoc_margin", "applicable_margin_rate"})
		if err != nil {
			dbQueryJSONError(w, 502, "RMS_EQ_VAR_ELM query failed; verify table and columns")
			return
		}
		resp["var_elm"] = rows
	} else if segment == "FO" {
		rows, err = dbQueryRows(sess.Server, "SELECT DL1_SCRIP_CODE,DL1_LTP,DL1_CLOSE FROM DRV_L1_WATCH WHERE DL1_SCRIP_CODE='"+code+"'", []string{"scrip_code", "ltp", "close"})
		if err != nil {
			dbQueryJSONError(w, 502, "DRV_L1_WATCH query failed; verify table and columns")
			return
		}
		resp["drv_l1_watch"] = rows
		q := "SELECT SM_SCRIP_CODE,SM_SYMBOL,SM_EXPIRY_DATE,SM_STRIKE_PRICE,SM_OPTION_TYPE,SM_UNDERLAYING_SCRIP_CODE,SM_UPPER_LIMIT,SM_LOWER_LIMIT,SM_EXPOSER_PERC,SM_BUY_ADDITIONAL_MRG,SM_SELL_ADDITIONAL_MRG,SM_SELL_EXPOSER_PERC,SM_ADD_LONG_EXP_PERC,SM_ADD_SHORT_EXP_PERC,SM_NEAR_FUT_SCRIP_CODE,SM_MATURITY_DATE FROM SECURITY_MASTER WHERE SM_SCRIP_CODE='" + code + "'"
		keys := []string{"scrip_code", "symbol", "expiry_date", "strike_price", "option_type", "underlaying_scrip_code", "upper_limit", "lower_limit", "exposer_perc", "buy_additional_mrg", "sell_additional_mrg", "sell_exposer_perc", "add_long_exp_perc", "add_short_exp_perc", "near_fut_scrip_code", "maturity_date"}
		rows, err = dbQueryRows(sess.Server, q, keys)
		if err != nil {
			dbQueryJSONError(w, 502, "SECURITY_MASTER query failed; verify table and columns")
			return
		}
		resp["security_master"] = rows
		spanExpr := "GREATEST(IFNULL(RSM_SPAN_1,0),IFNULL(RSM_SPAN_2,0),IFNULL(RSM_SPAN_3,0),IFNULL(RSM_SPAN_4,0),IFNULL(RSM_SPAN_5,0),IFNULL(RSM_SPAN_6,0),IFNULL(RSM_SPAN_7,0),IFNULL(RSM_SPAN_8,0),IFNULL(RSM_SPAN_9,0),IFNULL(RSM_SPAN_10,0),IFNULL(RSM_SPAN_11,0),IFNULL(RSM_SPAN_12,0),IFNULL(RSM_SPAN_13,0),IFNULL(RSM_SPAN_14,0),IFNULL(RSM_SPAN_15,0),IFNULL(RSM_SPAN_16,0))"
		lowExpr := "LEAST(IFNULL(RSM_SPAN_1,0),IFNULL(RSM_SPAN_2,0),IFNULL(RSM_SPAN_3,0),IFNULL(RSM_SPAN_4,0),IFNULL(RSM_SPAN_5,0),IFNULL(RSM_SPAN_6,0),IFNULL(RSM_SPAN_7,0),IFNULL(RSM_SPAN_8,0),IFNULL(RSM_SPAN_9,0),IFNULL(RSM_SPAN_10,0),IFNULL(RSM_SPAN_11,0),IFNULL(RSM_SPAN_12,0),IFNULL(RSM_SPAN_13,0),IFNULL(RSM_SPAN_14,0),IFNULL(RSM_SPAN_15,0),IFNULL(RSM_SPAN_16,0))"
		q = "SELECT RSM_SECURITY_ID,RSM_SYMBOL_NAME," + spanExpr + "," + lowExpr + " FROM RMS_FNO_SPAN_MARGIN WHERE RSM_SECURITY_ID='" + code + "'"
		rows, err = dbQueryRows(sess.Server, q, []string{"security_id", "symbol_name", "highest_span", "lowest_span"})
		if err != nil {
			dbQueryJSONError(w, 502, "RMS_FNO_SPAN_MARGIN query failed; verify table and columns")
			return
		}
		resp["fno_span"] = rows
	} else {
		dbQueryJSONError(w, http.StatusBadRequest, "Choose EQ or FO. Commodity/CD source table names were not provided.")
		return
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// ============================================================
// DB QUERY LIBRARY (merged into this file so the existing service
// can continue building from rms_api.go alone).
// ============================================================

const dbQueryMaxRows = 100

type dbLibraryQuery struct {
	ID         string
	Table      string
	Column     string
	NeedsValue bool
}

var dbLibraryQueries = map[string]dbLibraryQuery{
	"preview_ban_net_position":                                    {ID: "preview_ban_net_position", Table: "BAN_NET_POSITION", Column: "", NeedsValue: false},
	"preview_bhav_copy":                                           {ID: "preview_bhav_copy", Table: "BHAV_COPY", Column: "", NeedsValue: false},
	"preview_bse_gsm_temp":                                        {ID: "preview_bse_gsm_temp", Table: "BSE_GSM_TEMP", Column: "", NeedsValue: false},
	"preview_clientwise_mtf_exposer_limit":                        {ID: "preview_clientwise_mtf_exposer_limit", Table: "CLIENTWISE_MTF_EXPOSER_LIMIT", Column: "", NeedsValue: false},
	"preview_comm_orders":                                         {ID: "preview_comm_orders", Table: "COMM_ORDERS", Column: "", NeedsValue: false},
	"preview_drv_bhav_copy":                                       {ID: "preview_drv_bhav_copy", Table: "DRV_BHAV_COPY", Column: "", NeedsValue: false},
	"preview_drv_l1_watch":                                        {ID: "preview_drv_l1_watch", Table: "DRV_L1_WATCH", Column: "", NeedsValue: false},
	"preview_drv_orders":                                          {ID: "preview_drv_orders", Table: "DRV_ORDERS", Column: "", NeedsValue: false},
	"preview_entity_master":                                       {ID: "preview_entity_master", Table: "ENTITY_MASTER", Column: "", NeedsValue: false},
	"preview_eq_l1_watch":                                         {ID: "preview_eq_l1_watch", Table: "EQ_L1_WATCH", Column: "", NeedsValue: false},
	"preview_eq_orders":                                           {ID: "preview_eq_orders", Table: "EQ_ORDERS", Column: "", NeedsValue: false},
	"preview_exch_administration_master":                          {ID: "preview_exch_administration_master", Table: "EXCH_ADMINISTRATION_MASTER", Column: "", NeedsValue: false},
	"preview_file_error_log":                                      {ID: "preview_file_error_log", Table: "FILE_ERROR_LOG", Column: "", NeedsValue: false},
	"preview_go_client_allowed_immem":                             {ID: "preview_go_client_allowed_immem", Table: "GO_CLIENT_ALLOWED_IMMEM", Column: "", NeedsValue: false},
	"preview_go_client_mail_info":                                 {ID: "preview_go_client_mail_info", Table: "GO_CLIENT_MAIL_INFO", Column: "", NeedsValue: false},
	"preview_go_fileupload_config":                                {ID: "preview_go_fileupload_config", Table: "GO_FILEUPLOAD_CONFIG", Column: "", NeedsValue: false},
	"preview_go_user_master":                                      {ID: "preview_go_user_master", Table: "GO_USER_MASTER", Column: "", NeedsValue: false},
	"preview_l1_watch_active":                                     {ID: "preview_l1_watch_active", Table: "L1_WATCH_ACTIVE", Column: "", NeedsValue: false},
	"preview_mcx_security_info":                                   {ID: "preview_mcx_security_info", Table: "MCX_SECURITY_INFO", Column: "", NeedsValue: false},
	"preview_mrv_orders":                                          {ID: "preview_mrv_orders", Table: "MRV_ORDERS", Column: "", NeedsValue: false},
	"preview_net_position_table":                                  {ID: "preview_net_position_table", Table: "NET_POSITION_TABLE", Column: "", NeedsValue: false},
	"preview_nse_gsm_temp":                                        {ID: "preview_nse_gsm_temp", Table: "NSE_GSM_TEMP", Column: "", NeedsValue: false},
	"preview_rms_approve_pledg_script":                            {ID: "preview_rms_approve_pledg_script", Table: "RMS_APPROVE_PLEDG_SCRIPT", Column: "", NeedsValue: false},
	"preview_rms_clientwise_uploaded_limit":                       {ID: "preview_rms_clientwise_uploaded_limit", Table: "RMS_CLIENTWISE_UPLOADED_LIMIT", Column: "", NeedsValue: false},
	"preview_rms_client_fno_pos_limit_detail":                     {ID: "preview_rms_client_fno_pos_limit_detail", Table: "RMS_CLIENT_FNO_POS_LIMIT_DETAIL", Column: "", NeedsValue: false},
	"preview_rms_client_security_limit":                           {ID: "preview_rms_client_security_limit", Table: "RMS_CLIENT_SECURITY_LIMIT", Column: "", NeedsValue: false},
	"preview_rms_early_payin":                                     {ID: "preview_rms_early_payin", Table: "RMS_EARLY_PAYIN", Column: "", NeedsValue: false},
	"preview_rms_eq_var_elm":                                      {ID: "preview_rms_eq_var_elm", Table: "RMS_EQ_VAR_ELM", Column: "", NeedsValue: false},
	"preview_rms_fno_delta":                                       {ID: "preview_rms_fno_delta", Table: "RMS_FNO_DELTA", Column: "", NeedsValue: false},
	"preview_rms_fno_span_margin":                                 {ID: "preview_rms_fno_span_margin", Table: "RMS_FNO_SPAN_MARGIN", Column: "", NeedsValue: false},
	"preview_rms_fund_limit":                                      {ID: "preview_rms_fund_limit", Table: "RMS_FUND_LIMIT", Column: "", NeedsValue: false},
	"preview_rms_ipo_allotment_temp":                              {ID: "preview_rms_ipo_allotment_temp", Table: "RMS_IPO_ALLOTMENT_TEMP", Column: "", NeedsValue: false},
	"preview_rms_limit_config":                                    {ID: "preview_rms_limit_config", Table: "RMS_LIMIT_CONFIG", Column: "", NeedsValue: false},
	"preview_rms_mcx_exp_margin":                                  {ID: "preview_rms_mcx_exp_margin", Table: "RMS_MCX_EXP_MARGIN", Column: "", NeedsValue: false},
	"preview_rms_member_fno_pos_limit_detail":                     {ID: "preview_rms_member_fno_pos_limit_detail", Table: "RMS_MEMBER_FNO_POS_LIMIT_DETAIL", Column: "", NeedsValue: false},
	"preview_rms_memwise_uploaded_limit":                          {ID: "preview_rms_memwise_uploaded_limit", Table: "RMS_MEMWISE_UPLOADED_LIMIT", Column: "", NeedsValue: false},
	"preview_rms_process_log":                                     {ID: "preview_rms_process_log", Table: "RMS_PROCESS_LOG", Column: "", NeedsValue: false},
	"preview_rms_risk_profile_det":                                {ID: "preview_rms_risk_profile_det", Table: "RMS_RISK_PROFILE_DET", Column: "", NeedsValue: false},
	"preview_rms_risk_profile_mast":                               {ID: "preview_rms_risk_profile_mast", Table: "RMS_RISK_PROFILE_MAST", Column: "", NeedsValue: false},
	"preview_rms_screen_column_config":                            {ID: "preview_rms_screen_column_config", Table: "RMS_SCREEN_COLUMN_CONFIG", Column: "", NeedsValue: false},
	"preview_rms_scrip_margin_basket":                             {ID: "preview_rms_scrip_margin_basket", Table: "RMS_SCRIP_MARGIN_BASKET", Column: "", NeedsValue: false},
	"preview_rms_scrip_margin_basket_fno":                         {ID: "preview_rms_scrip_margin_basket_fno", Table: "RMS_SCRIP_MARGIN_BASKET_FNO", Column: "", NeedsValue: false},
	"preview_security_master":                                     {ID: "preview_security_master", Table: "SECURITY_MASTER", Column: "", NeedsValue: false},
	"preview_security_rms_block_detail":                           {ID: "preview_security_rms_block_detail", Table: "SECURITY_RMS_BLOCK_DETAIL", Column: "", NeedsValue: false},
	"preview_sem_active":                                          {ID: "preview_sem_active", Table: "SEM_ACTIVE", Column: "", NeedsValue: false},
	"preview_surveillance_msg_details":                            {ID: "preview_surveillance_msg_details", Table: "SURVEILLANCE_MSG_DETAILS", Column: "", NeedsValue: false},
	"preview_sys_parameters":                                      {ID: "preview_sys_parameters", Table: "SYS_PARAMETERS", Column: "", NeedsValue: false},
	"preview_user_master":                                         {ID: "preview_user_master", Table: "USER_MASTER", Column: "", NeedsValue: false},
	"preview_view_client_limit":                                   {ID: "preview_view_client_limit", Table: "VIEW_CLIENT_LIMIT", Column: "", NeedsValue: false},
	"filter_001_ban_net_position_bnp_client_id":                   {ID: "filter_001_ban_net_position_bnp_client_id", Table: "BAN_NET_POSITION", Column: "BNP_CLIENT_ID", NeedsValue: true},
	"filter_002_bhav_copy_bc_scrip_code":                          {ID: "filter_002_bhav_copy_bc_scrip_code", Table: "BHAV_COPY", Column: "BC_SCRIP_CODE", NeedsValue: true},
	"filter_003_bse_gsm_temp_v_id_1":                              {ID: "filter_003_bse_gsm_temp_v_id_1", Table: "BSE_GSM_TEMP", Column: "V_ID_1", NeedsValue: true},
	"filter_004_clientwise_mtf_exposer_limit_cme_client_id":       {ID: "filter_004_clientwise_mtf_exposer_limit_cme_client_id", Table: "CLIENTWISE_MTF_EXPOSER_LIMIT", Column: "CME_CLIENT_ID", NeedsValue: true},
	"filter_005_comm_orders_comm_client_id":                       {ID: "filter_005_comm_orders_comm_client_id", Table: "COMM_ORDERS", Column: "COMM_CLIENT_ID", NeedsValue: true},
	"filter_006_drv_bhav_copy_dbc_scrip_code":                     {ID: "filter_006_drv_bhav_copy_dbc_scrip_code", Table: "DRV_BHAV_COPY", Column: "DBC_SCRIP_CODE", NeedsValue: true},
	"filter_007_drv_l1_watch_dl1_scrip_code":                      {ID: "filter_007_drv_l1_watch_dl1_scrip_code", Table: "DRV_L1_WATCH", Column: "DL1_SCRIP_CODE", NeedsValue: true},
	"filter_008_drv_orders_drv_client_id":                         {ID: "filter_008_drv_orders_drv_client_id", Table: "DRV_ORDERS", Column: "DRV_CLIENT_ID", NeedsValue: true},
	"filter_009_entity_master_entity_code":                        {ID: "filter_009_entity_master_entity_code", Table: "ENTITY_MASTER", Column: "ENTITY_CODE", NeedsValue: true},
	"filter_010_eq_l1_watch_l1_scrip_code":                        {ID: "filter_010_eq_l1_watch_l1_scrip_code", Table: "EQ_L1_WATCH", Column: "L1_SCRIP_CODE", NeedsValue: true},
	"filter_011_eq_orders_eq_client_id":                           {ID: "filter_011_eq_orders_eq_client_id", Table: "EQ_ORDERS", Column: "EQ_CLIENT_ID", NeedsValue: true},
	"filter_012_exch_administration_master_eam_broker_status":     {ID: "filter_012_exch_administration_master_eam_broker_status", Table: "EXCH_ADMINISTRATION_MASTER", Column: "EAM_BROKER_STATUS", NeedsValue: true},
	"filter_013_file_error_log_upload_time":                       {ID: "filter_013_file_error_log_upload_time", Table: "FILE_ERROR_LOG", Column: "UPLOAD_TIME", NeedsValue: true},
	"filter_014_go_client_allowed_immem_client_id":                {ID: "filter_014_go_client_allowed_immem_client_id", Table: "GO_CLIENT_ALLOWED_IMMEM", Column: "CLIENT_ID", NeedsValue: true},
	"filter_015_go_client_mail_info_g_client_id":                  {ID: "filter_015_go_client_mail_info_g_client_id", Table: "GO_CLIENT_MAIL_INFO", Column: "G_CLIENT_ID", NeedsValue: true},
	"filter_016_go_fileupload_config_date_flag":                   {ID: "filter_016_go_fileupload_config_date_flag", Table: "GO_FILEUPLOAD_CONFIG", Column: "DATE_FLAG", NeedsValue: true},
	"filter_017_go_user_master_user_id":                           {ID: "filter_017_go_user_master_user_id", Table: "GO_USER_MASTER", Column: "USER_ID", NeedsValue: true},
	"filter_018_l1_watch_active_l1_scrip_code":                    {ID: "filter_018_l1_watch_active_l1_scrip_code", Table: "L1_WATCH_ACTIVE", Column: "L1_SCRIP_CODE", NeedsValue: true},
	"filter_019_mcx_security_info_mcx_symbol":                     {ID: "filter_019_mcx_security_info_mcx_symbol", Table: "MCX_SECURITY_INFO", Column: "MCX_SYMBOL", NeedsValue: true},
	"filter_020_mrv_orders_mrv_client_id":                         {ID: "filter_020_mrv_orders_mrv_client_id", Table: "MRV_ORDERS", Column: "MRV_CLIENT_ID", NeedsValue: true},
	"filter_021_net_position_table_npt_client_id":                 {ID: "filter_021_net_position_table_npt_client_id", Table: "NET_POSITION_TABLE", Column: "NPT_CLIENT_ID", NeedsValue: true},
	"filter_022_nse_gsm_temp_v_id_1":                              {ID: "filter_022_nse_gsm_temp_v_id_1", Table: "NSE_GSM_TEMP", Column: "V_ID_1", NeedsValue: true},
	"filter_023_rms_approve_pledg_script_raps_isin":               {ID: "filter_023_rms_approve_pledg_script_raps_isin", Table: "RMS_APPROVE_PLEDG_SCRIPT", Column: "RAPS_ISIN", NeedsValue: true},
	"filter_024_rms_clientwise_uploaded_limit_rcul_symbol":        {ID: "filter_024_rms_clientwise_uploaded_limit_rcul_symbol", Table: "RMS_CLIENTWISE_UPLOADED_LIMIT", Column: "RCUL_SYMBOL", NeedsValue: true},
	"filter_025_rms_client_fno_pos_limit_detail_client_id":        {ID: "filter_025_rms_client_fno_pos_limit_detail_client_id", Table: "RMS_CLIENT_FNO_POS_LIMIT_DETAIL", Column: "CLIENT_ID", NeedsValue: true},
	"filter_026_rms_client_security_limit_client_id":              {ID: "filter_026_rms_client_security_limit_client_id", Table: "RMS_CLIENT_SECURITY_LIMIT", Column: "CLIENT_ID", NeedsValue: true},
	"filter_027_rms_early_payin_rep_client_id":                    {ID: "filter_027_rms_early_payin_rep_client_id", Table: "RMS_EARLY_PAYIN", Column: "REP_CLIENT_ID", NeedsValue: true},
	"filter_028_rms_eq_var_elm_sec_symbol":                        {ID: "filter_028_rms_eq_var_elm_sec_symbol", Table: "RMS_EQ_VAR_ELM", Column: "SEC_SYMBOL", NeedsValue: true},
	"filter_029_rms_fno_delta_rfd_scrip_code":                     {ID: "filter_029_rms_fno_delta_rfd_scrip_code", Table: "RMS_FNO_DELTA", Column: "RFD_SCRIP_CODE", NeedsValue: true},
	"filter_030_rms_fno_span_margin_rsm_security_id":              {ID: "filter_030_rms_fno_span_margin_rsm_security_id", Table: "RMS_FNO_SPAN_MARGIN", Column: "RSM_SECURITY_ID", NeedsValue: true},
	"filter_031_rms_fund_limit_client_id":                         {ID: "filter_031_rms_fund_limit_client_id", Table: "RMS_FUND_LIMIT", Column: "CLIENT_ID", NeedsValue: true},
	"filter_032_rms_ipo_allotment_temp_symbol":                    {ID: "filter_032_rms_ipo_allotment_temp_symbol", Table: "RMS_IPO_ALLOTMENT_TEMP", Column: "SYMBOL", NeedsValue: true},
	"filter_033_rms_limit_config_rlc_segment":                     {ID: "filter_033_rms_limit_config_rlc_segment", Table: "RMS_LIMIT_CONFIG", Column: "RLC_SEGMENT", NeedsValue: true},
	"filter_034_rms_mcx_exp_margin_rmem_security_id":              {ID: "filter_034_rms_mcx_exp_margin_rmem_security_id", Table: "RMS_MCX_EXP_MARGIN", Column: "RMEM_SECURITY_ID", NeedsValue: true},
	"filter_035_rms_member_fno_pos_limit_detail_exch_symbol":      {ID: "filter_035_rms_member_fno_pos_limit_detail_exch_symbol", Table: "RMS_MEMBER_FNO_POS_LIMIT_DETAIL", Column: "EXCH_SYMBOL", NeedsValue: true},
	"filter_036_rms_memwise_uploaded_limit_rmul_symbol":           {ID: "filter_036_rms_memwise_uploaded_limit_rmul_symbol", Table: "RMS_MEMWISE_UPLOADED_LIMIT", Column: "RMUL_SYMBOL", NeedsValue: true},
	"filter_037_rms_process_log_process_date":                     {ID: "filter_037_rms_process_log_process_date", Table: "RMS_PROCESS_LOG", Column: "PROCESS_DATE", NeedsValue: true},
	"filter_038_rms_risk_profile_det_rpd_scrip_code":              {ID: "filter_038_rms_risk_profile_det_rpd_scrip_code", Table: "RMS_RISK_PROFILE_DET", Column: "RPD_SCRIP_CODE", NeedsValue: true},
	"filter_039_rms_risk_profile_mast_rpm_product_enabled":        {ID: "filter_039_rms_risk_profile_mast_rpm_product_enabled", Table: "RMS_RISK_PROFILE_MAST", Column: "RPM_PRODUCT_ENABLED", NeedsValue: true},
	"filter_040_rms_screen_column_config_rscc_screen_name":        {ID: "filter_040_rms_screen_column_config_rscc_screen_name", Table: "RMS_SCREEN_COLUMN_CONFIG", Column: "RSCC_SCREEN_NAME", NeedsValue: true},
	"filter_041_rms_scrip_margin_basket_rsmb_scrip_code":          {ID: "filter_041_rms_scrip_margin_basket_rsmb_scrip_code", Table: "RMS_SCRIP_MARGIN_BASKET", Column: "RSMB_SCRIP_CODE", NeedsValue: true},
	"filter_042_rms_scrip_margin_basket_fno_rsmbf_scrip_code":     {ID: "filter_042_rms_scrip_margin_basket_fno_rsmbf_scrip_code", Table: "RMS_SCRIP_MARGIN_BASKET_FNO", Column: "RSMBF_SCRIP_CODE", NeedsValue: true},
	"filter_043_security_master_sm_scrip_code":                    {ID: "filter_043_security_master_sm_scrip_code", Table: "SECURITY_MASTER", Column: "SM_SCRIP_CODE", NeedsValue: true},
	"filter_044_security_rms_block_detail_client_id":              {ID: "filter_044_security_rms_block_detail_client_id", Table: "SECURITY_RMS_BLOCK_DETAIL", Column: "CLIENT_ID", NeedsValue: true},
	"filter_045_sem_active_sm_scrip_code":                         {ID: "filter_045_sem_active_sm_scrip_code", Table: "SEM_ACTIVE", Column: "SM_SCRIP_CODE", NeedsValue: true},
	"filter_046_surveillance_msg_details_smd_client_id":           {ID: "filter_046_surveillance_msg_details_smd_client_id", Table: "SURVEILLANCE_MSG_DETAILS", Column: "SMD_CLIENT_ID", NeedsValue: true},
	"filter_047_sys_parameters_param_name":                        {ID: "filter_047_sys_parameters_param_name", Table: "SYS_PARAMETERS", Column: "PARAM_NAME", NeedsValue: true},
	"filter_048_user_master_user_entity_code":                     {ID: "filter_048_user_master_user_entity_code", Table: "USER_MASTER", Column: "USER_ENTITY_CODE", NeedsValue: true},
	"filter_049_view_client_limit_client_id":                      {ID: "filter_049_view_client_limit_client_id", Table: "VIEW_CLIENT_LIMIT", Column: "CLIENT_ID", NeedsValue: true},
	"filter_050_ban_net_position_bnp_underlying_scrip_code":       {ID: "filter_050_ban_net_position_bnp_underlying_scrip_code", Table: "BAN_NET_POSITION", Column: "BNP_UNDERLYING_SCRIP_CODE", NeedsValue: true},
	"filter_051_bhav_copy_bc_segment":                             {ID: "filter_051_bhav_copy_bc_segment", Table: "BHAV_COPY", Column: "BC_SEGMENT", NeedsValue: true},
	"filter_052_bse_gsm_temp_v_id_2":                              {ID: "filter_052_bse_gsm_temp_v_id_2", Table: "BSE_GSM_TEMP", Column: "V_ID_2", NeedsValue: true},
	"filter_053_clientwise_mtf_exposer_limit_cme_symbol":          {ID: "filter_053_clientwise_mtf_exposer_limit_cme_symbol", Table: "CLIENTWISE_MTF_EXPOSER_LIMIT", Column: "CME_SYMBOL", NeedsValue: true},
	"filter_054_comm_orders_comm_scrip_code":                      {ID: "filter_054_comm_orders_comm_scrip_code", Table: "COMM_ORDERS", Column: "COMM_SCRIP_CODE", NeedsValue: true},
	"filter_055_drv_bhav_copy_dbc_segment":                        {ID: "filter_055_drv_bhav_copy_dbc_segment", Table: "DRV_BHAV_COPY", Column: "DBC_SEGMENT", NeedsValue: true},
	"filter_056_drv_l1_watch_dl1_exch_scrip_code":                 {ID: "filter_056_drv_l1_watch_dl1_exch_scrip_code", Table: "DRV_L1_WATCH", Column: "DL1_EXCH_SCRIP_CODE", NeedsValue: true},
	"filter_057_drv_orders_drv_scrip_code":                        {ID: "filter_057_drv_orders_drv_scrip_code", Table: "DRV_ORDERS", Column: "DRV_SCRIP_CODE", NeedsValue: true},
	"filter_058_entity_master_entity_risk_profile":                {ID: "filter_058_entity_master_entity_risk_profile", Table: "ENTITY_MASTER", Column: "ENTITY_RISK_PROFILE", NeedsValue: true},
	"filter_059_eq_l1_watch_l1_exch_scrip_code":                   {ID: "filter_059_eq_l1_watch_l1_exch_scrip_code", Table: "EQ_L1_WATCH", Column: "L1_EXCH_SCRIP_CODE", NeedsValue: true},
	"filter_060_eq_orders_eq_scrip_code":                          {ID: "filter_060_eq_orders_eq_scrip_code", Table: "EQ_ORDERS", Column: "EQ_SCRIP_CODE", NeedsValue: true},
	"filter_061_exch_administration_master_eam_logon_status":      {ID: "filter_061_exch_administration_master_eam_logon_status", Table: "EXCH_ADMINISTRATION_MASTER", Column: "EAM_LOGON_STATUS", NeedsValue: true},
	"filter_062_file_error_log_err_code":                          {ID: "filter_062_file_error_log_err_code", Table: "FILE_ERROR_LOG", Column: "ERR_CODE", NeedsValue: true},
	"filter_063_go_client_allowed_immem_segment":                  {ID: "filter_063_go_client_allowed_immem_segment", Table: "GO_CLIENT_ALLOWED_IMMEM", Column: "SEGMENT", NeedsValue: true},
	"filter_064_go_client_mail_info_g_email_id":                   {ID: "filter_064_go_client_mail_info_g_email_id", Table: "GO_CLIENT_MAIL_INFO", Column: "G_EMAIL_ID", NeedsValue: true},
	"filter_065_go_fileupload_config_priority":                    {ID: "filter_065_go_fileupload_config_priority", Table: "GO_FILEUPLOAD_CONFIG", Column: "PRIORITY", NeedsValue: true},
	"filter_066_go_user_master_last_update_time":                  {ID: "filter_066_go_user_master_last_update_time", Table: "GO_USER_MASTER", Column: "LAST_UPDATE_TIME", NeedsValue: true},
	"filter_067_l1_watch_active_l1_exch_scrip_code":               {ID: "filter_067_l1_watch_active_l1_exch_scrip_code", Table: "L1_WATCH_ACTIVE", Column: "L1_EXCH_SCRIP_CODE", NeedsValue: true},
	"filter_068_mcx_security_info_mcx_expiry_process_date":        {ID: "filter_068_mcx_security_info_mcx_expiry_process_date", Table: "MCX_SECURITY_INFO", Column: "MCX_EXPIRY_PROCESS_DATE", NeedsValue: true},
	"filter_069_mrv_orders_mrv_scrip_code":                        {ID: "filter_069_mrv_orders_mrv_scrip_code", Table: "MRV_ORDERS", Column: "MRV_SCRIP_CODE", NeedsValue: true},
	"filter_070_net_position_table_npt_security_id":               {ID: "filter_070_net_position_table_npt_security_id", Table: "NET_POSITION_TABLE", Column: "NPT_SECURITY_ID", NeedsValue: true},
	"filter_071_nse_gsm_temp_v_id_2":                              {ID: "filter_071_nse_gsm_temp_v_id_2", Table: "NSE_GSM_TEMP", Column: "V_ID_2", NeedsValue: true},
	"filter_072_rms_clientwise_uploaded_limit_rcul_exch_id":       {ID: "filter_072_rms_clientwise_uploaded_limit_rcul_exch_id", Table: "RMS_CLIENTWISE_UPLOADED_LIMIT", Column: "RCUL_EXCH_ID", NeedsValue: true},
	"filter_073_rms_client_fno_pos_limit_detail_exch_symbol":      {ID: "filter_073_rms_client_fno_pos_limit_detail_exch_symbol", Table: "RMS_CLIENT_FNO_POS_LIMIT_DETAIL", Column: "EXCH_SYMBOL", NeedsValue: true},
	"filter_074_rms_client_security_limit_nse_scrip_code":         {ID: "filter_074_rms_client_security_limit_nse_scrip_code", Table: "RMS_CLIENT_SECURITY_LIMIT", Column: "NSE_SCRIP_CODE", NeedsValue: true},
	"filter_075_rms_early_payin_rep_isin_reqid":                   {ID: "filter_075_rms_early_payin_rep_isin_reqid", Table: "RMS_EARLY_PAYIN", Column: "REP_ISIN_REQID", NeedsValue: true},
	"filter_076_rms_eq_var_elm_isin":                              {ID: "filter_076_rms_eq_var_elm_isin", Table: "RMS_EQ_VAR_ELM", Column: "ISIN", NeedsValue: true},
	"filter_077_rms_fno_delta_rfd_underlying_scrip_code":          {ID: "filter_077_rms_fno_delta_rfd_underlying_scrip_code", Table: "RMS_FNO_DELTA", Column: "RFD_UNDERLYING_SCRIP_CODE", NeedsValue: true},
	"filter_078_rms_fno_span_margin_rsm_symbol_name":              {ID: "filter_078_rms_fno_span_margin_rsm_symbol_name", Table: "RMS_FNO_SPAN_MARGIN", Column: "RSM_SYMBOL_NAME", NeedsValue: true},
	"filter_079_rms_fund_limit_segment":                           {ID: "filter_079_rms_fund_limit_segment", Table: "RMS_FUND_LIMIT", Column: "SEGMENT", NeedsValue: true},
	"filter_080_rms_ipo_allotment_temp_upi_last_status":           {ID: "filter_080_rms_ipo_allotment_temp_upi_last_status", Table: "RMS_IPO_ALLOTMENT_TEMP", Column: "UPI_LAST_STATUS", NeedsValue: true},
	"filter_081_rms_limit_config_rcl_segment_include":             {ID: "filter_081_rms_limit_config_rcl_segment_include", Table: "RMS_LIMIT_CONFIG", Column: "RCL_SEGMENT_INCLUDE", NeedsValue: true},
	"filter_082_rms_mcx_exp_margin_rmem_symbol_name":              {ID: "filter_082_rms_mcx_exp_margin_rmem_symbol_name", Table: "RMS_MCX_EXP_MARGIN", Column: "RMEM_SYMBOL_NAME", NeedsValue: true},
	"filter_083_rms_member_fno_pos_limit_detail_tot_gross_qty":    {ID: "filter_083_rms_member_fno_pos_limit_detail_tot_gross_qty", Table: "RMS_MEMBER_FNO_POS_LIMIT_DETAIL", Column: "TOT_GROSS_QTY", NeedsValue: true},
	"filter_084_rms_memwise_uploaded_limit_rmul_prop_allowed_qty": {ID: "filter_084_rms_memwise_uploaded_limit_rmul_prop_allowed_qty", Table: "RMS_MEMWISE_UPLOADED_LIMIT", Column: "RMUL_PROP_ALLOWED_QTY", NeedsValue: true},
	"filter_085_rms_process_log_upload_file_name":                 {ID: "filter_085_rms_process_log_upload_file_name", Table: "RMS_PROCESS_LOG", Column: "UPLOAD_FILE_NAME", NeedsValue: true},
	"filter_086_rms_risk_profile_det_rpd_symbol":                  {ID: "filter_086_rms_risk_profile_det_rpd_symbol", Table: "RMS_RISK_PROFILE_DET", Column: "RPD_SYMBOL", NeedsValue: true},
	"filter_087_rms_risk_profile_mast_product_ql_check":           {ID: "filter_087_rms_risk_profile_mast_product_ql_check", Table: "RMS_RISK_PROFILE_MAST", Column: "PRODUCT_QL_CHECK", NeedsValue: true},
	"filter_088_rms_screen_column_config_rscc_column_names":       {ID: "filter_088_rms_screen_column_config_rscc_column_names", Table: "RMS_SCREEN_COLUMN_CONFIG", Column: "RSCC_COLUMN_NAMES", NeedsValue: true},
	"filter_089_rms_scrip_margin_basket_rsmb_segment":             {ID: "filter_089_rms_scrip_margin_basket_rsmb_segment", Table: "RMS_SCRIP_MARGIN_BASKET", Column: "RSMB_SEGMENT", NeedsValue: true},
	"filter_090_rms_scrip_margin_basket_fno_rsmbf_segment":        {ID: "filter_090_rms_scrip_margin_basket_fno_rsmbf_segment", Table: "RMS_SCRIP_MARGIN_BASKET_FNO", Column: "RSMBF_SEGMENT", NeedsValue: true},
	"filter_091_security_master_sm_exch_scrip_code":               {ID: "filter_091_security_master_sm_exch_scrip_code", Table: "SECURITY_MASTER", Column: "SM_EXCH_SCRIP_CODE", NeedsValue: true},
	"filter_092_security_rms_block_detail_scrip_code":             {ID: "filter_092_security_rms_block_detail_scrip_code", Table: "SECURITY_RMS_BLOCK_DETAIL", Column: "SCRIP_CODE", NeedsValue: true},
	"filter_093_sem_active_sm_exch_scrip_code":                    {ID: "filter_093_sem_active_sm_exch_scrip_code", Table: "SEM_ACTIVE", Column: "SM_EXCH_SCRIP_CODE", NeedsValue: true},
	"filter_094_surveillance_msg_details_smd_scrip_code":          {ID: "filter_094_surveillance_msg_details_smd_scrip_code", Table: "SURVEILLANCE_MSG_DETAILS", Column: "SMD_SCRIP_CODE", NeedsValue: true},
	"filter_095_sys_parameters_param_value":                       {ID: "filter_095_sys_parameters_param_value", Table: "SYS_PARAMETERS", Column: "PARAM_VALUE", NeedsValue: true},
	"filter_096_user_master_user_status":                          {ID: "filter_096_user_master_user_status", Table: "USER_MASTER", Column: "USER_STATUS", NeedsValue: true},
	"filter_097_view_client_limit_segment":                        {ID: "filter_097_view_client_limit_segment", Table: "VIEW_CLIENT_LIMIT", Column: "SEGMENT", NeedsValue: true},
	"filter_098_ban_net_position_bnp_security_id":                 {ID: "filter_098_ban_net_position_bnp_security_id", Table: "BAN_NET_POSITION", Column: "BNP_SECURITY_ID", NeedsValue: true},
	"filter_099_bhav_copy_bc_date":                                {ID: "filter_099_bhav_copy_bc_date", Table: "BHAV_COPY", Column: "BC_DATE", NeedsValue: true},
	"filter_100_bse_gsm_temp_v_id_3":                              {ID: "filter_100_bse_gsm_temp_v_id_3", Table: "BSE_GSM_TEMP", Column: "V_ID_3", NeedsValue: true},
	"filter_101_clientwise_mtf_exposer_limit_cme_tot_qty":         {ID: "filter_101_clientwise_mtf_exposer_limit_cme_tot_qty", Table: "CLIENTWISE_MTF_EXPOSER_LIMIT", Column: "CME_TOT_QTY", NeedsValue: true},
}

// dbQueryEscape escapes a user-supplied value; SQL identifiers are never user-supplied.
func dbQueryEscape(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	return strings.ReplaceAll(value, "'", "''")
}

func dbQueryQuote(value string) string { return "'" + dbQueryEscape(value) + "'" }

// dbQueryBuildSQL only builds SQL from the fixed allowlist above. Every query has LIMIT 100.
func dbQueryBuildSQL(id, value string) (string, error) {
	q, ok := dbLibraryQueries[id]
	if !ok {
		return "", fmt.Errorf("Unknown predefined query")
	}
	table := "`" + q.Table + "`"
	if !q.NeedsValue {
		return "SELECT * FROM " + table + " LIMIT 100", nil
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("Enter a value for " + q.Column)
	}
	if len(value) > 100 {
		return "", fmt.Errorf("Parameter too long (maximum 100 characters)")
	}
	column := "`" + q.Column + "`"
	return "SELECT * FROM " + table + " WHERE " + column + " = " + dbQueryQuote(value) + " LIMIT 100", nil
}

// apiDBQueryLibrary executes only allowlisted read-only SELECT queries.
func apiDBQueryLibrary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		dbQueryJSONError(w, http.StatusMethodNotAllowed, "Use POST")
		return
	}
	if err := dbQueryLoadEnv(); err != nil {
		dbQueryJSONError(w, http.StatusInternalServerError, "Cannot read database configuration")
		return
	}
	sess, ok := dbQueryCurrentSession(w, r)
	if !ok {
		return
	}
	var req struct {
		DB    string `json:"db"`
		ID    string `json:"id"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		dbQueryJSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	if req.DB != sess.Server {
		dbQueryJSONError(w, http.StatusForbidden, "Selected server does not match connected session")
		return
	}
	query, err := dbQueryBuildSQL(req.ID, req.Value)
	if err != nil {
		dbQueryJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	host, port, name, user, pass, err := dbQueryConfig(sess.Server)
	if err != nil {
		dbQueryJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "mysql", "--batch", "--raw", "--connect-timeout=5", "-h", host, "-P", port, "-u", user, name, "-e", query)
	cmd.Env = append(os.Environ(), "MYSQL_PWD="+pass)
	output, runErr := cmd.CombinedOutput()
	if runErr != nil {
		dbQueryJSONError(w, http.StatusBadGateway, "Query failed; verify table, columns, permissions and MySQL client")
		return
	}
	out := strings.TrimSpace(string(output))
	if out == "" {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"columns": []string{}, "rows": []interface{}{}, "count": 0, "max_rows": dbQueryMaxRows, "query": query})
		return
	}
	lines := strings.Split(out, "\n")
	columns := strings.Split(lines[0], "\t")
	rows := make([]map[string]interface{}, 0, minInt(len(lines)-1, dbQueryMaxRows))
	for _, line := range lines[1:] {
		if len(rows) >= dbQueryMaxRows {
			break
		}
		fields := strings.Split(line, "\t")
		if len(fields) != len(columns) {
			continue
		}
		row := make(map[string]interface{}, len(columns))
		for i, col := range columns {
			if fields[i] == "NULL" {
				row[col] = nil
			} else {
				row[col] = fields[i]
			}
		}
		rows = append(rows, row)
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"columns": columns, "rows": rows, "count": len(rows), "max_rows": dbQueryMaxRows, "query": query})
}

// apiDBQueryUpdateExpiry is a narrowly allowlisted write operation for expiry dates.
// It only updates SECURITY_MASTER and RMS_FNO_SPAN_MARGIN for segment D.
// SELECT snapshots from SECURITY_MASTER, RMS_FNO_SPAN_MARGIN and SEM_ACTIVE
// are returned both before and after the update.
func apiDBQueryUpdateExpiry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		dbQueryJSONError(w, http.StatusMethodNotAllowed, "Use POST")
		return
	}
	if err := dbQueryLoadEnv(); err != nil {
		dbQueryJSONError(w, http.StatusInternalServerError, "Cannot read database configuration")
		return
	}
	sess, ok := dbQueryCurrentSession(w, r)
	if !ok {
		return
	}
	var req struct {
		DB         string `json:"db"`
		SecurityID string `json:"security_id"`
		ExpiryDate string `json:"expiry_date"` // DDMMYYYY
		ExpiryTime string `json:"expiry_time"` // HH:MM:SS
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		dbQueryJSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	if req.DB != sess.Server {
		dbQueryJSONError(w, http.StatusForbidden, "Selected server does not match connected session")
		return
	}
	securityID := strings.TrimSpace(req.SecurityID)
	if securityID == "" || len(securityID) > 20 {
		dbQueryJSONError(w, http.StatusBadRequest, "Enter a valid numeric Security ID")
		return
	}
	for _, ch := range securityID {
		if ch < '0' || ch > '9' {
			dbQueryJSONError(w, http.StatusBadRequest, "Security ID must contain digits only")
			return
		}
	}
	expiryTime := strings.TrimSpace(req.ExpiryTime)
	if expiryTime == "" {
		expiryTime = "23:30:00"
	}
	parsed, err := time.Parse("02012006 15:04:05", strings.TrimSpace(req.ExpiryDate)+" "+expiryTime)
	if err != nil {
		dbQueryJSONError(w, http.StatusBadRequest, "Enter a valid expiry date in DDMMYYYY and time in HH:MM:SS")
		return
	}
	expiry := parsed.Format("2006-01-02 15:04:05")
	idSQL := dbQueryQuote(securityID)
	expirySQL := dbQueryQuote(expiry)

	queries := []struct {
		title string
		query string
		cols  []string
	}{
		{"SECURITY_MASTER", "SELECT SM_SCRIP_CODE, SM_SYMBOL, SM_EXPIRY_DATE, SM_MATURITY_DATE FROM SECURITY_MASTER WHERE SM_SCRIP_CODE = " + idSQL + " AND SM_SEGMENT = 'D'", []string{"SM_SCRIP_CODE", "SM_SYMBOL", "SM_EXPIRY_DATE", "SM_MATURITY_DATE"}},
		{"RMS_FNO_SPAN_MARGIN", "SELECT RSM_SECURITY_ID, RSM_SYMBOL_NAME, RSM_EXPIRY_DATE FROM RMS_FNO_SPAN_MARGIN WHERE RSM_SECURITY_ID = " + idSQL + " AND RSM_SEGMENT = 'D'", []string{"RSM_SECURITY_ID", "RSM_SYMBOL_NAME", "RSM_EXPIRY_DATE"}},
		{"SEM_ACTIVE", "SELECT SM_SCRIP_CODE, SM_SYMBOL, SM_EXPIRY_DATE, SM_MATURITY_DATE FROM SEM_ACTIVE WHERE SM_SCRIP_CODE = " + idSQL, []string{"SM_SCRIP_CODE", "SM_SYMBOL", "SM_EXPIRY_DATE", "SM_MATURITY_DATE"}},
	}
	readSections := func(phase string) ([]map[string]interface{}, error) {
		sections := make([]map[string]interface{}, 0, len(queries))
		for _, q := range queries {
			rows, queryErr := dbQueryRows(sess.Server, q.query, q.cols)
			if queryErr != nil {
				return nil, queryErr
			}
			sections = append(sections, map[string]interface{}{
				"title": phase + " — " + q.title, "query": q.query, "columns": q.cols, "rows": rows,
			})
		}
		return sections, nil
	}

	before, err := readSections("BEFORE UPDATE")
	if err != nil {
		dbQueryJSONError(w, http.StatusBadGateway, "Could not read expiry records before update")
		return
	}
	// Do not write if either target record is absent.
	if len(before[0]["rows"].([]map[string]interface{})) == 0 {
		dbQueryJSONError(w, http.StatusConflict, "No matching SECURITY_MASTER record for this Security ID and segment D. Nothing was updated.")
		return
	}
	if len(before[1]["rows"].([]map[string]interface{})) == 0 {
		dbQueryJSONError(w, http.StatusConflict, "No matching RMS_FNO_SPAN_MARGIN record for this Security ID and segment D. Nothing was updated.")
		return
	}

	// Only these two fixed UPDATE statements are accepted by this endpoint.
	updateSQL := "START TRANSACTION; " +
		"UPDATE SECURITY_MASTER SET SM_EXPIRY_DATE = " + expirySQL + ", SM_MATURITY_DATE = " + expirySQL + " WHERE SM_SCRIP_CODE = " + idSQL + " AND SM_SEGMENT = 'D'; " +
		"UPDATE RMS_FNO_SPAN_MARGIN SET RSM_EXPIRY_DATE = " + expirySQL + " WHERE RSM_SECURITY_ID = " + idSQL + " AND RSM_SEGMENT = 'D'; " +
		"COMMIT;"
	if _, err = dbQueryRun(sess.Server, updateSQL); err != nil {
		dbQueryJSONError(w, http.StatusBadGateway, "Expiry update failed. Check table permissions and database connection; verify the records before retrying.")
		return
	}
	after, err := readSections("AFTER UPDATE")
	if err != nil {
		dbQueryJSONError(w, http.StatusBadGateway, "Update command completed, but could not read records afterward. Verify the expiry values before retrying.")
		return
	}
	updatedAt := time.Now().Format("2006-01-02 15:04:05")
	// Use the database server's clock for the displayed update timestamp.
	if stampRows, stampErr := dbQueryRows(sess.Server, "SELECT DATE_FORMAT(NOW(), '%Y-%m-%d %H:%i:%s') AS updated_at", []string{"updated_at"}); stampErr == nil && len(stampRows) > 0 {
		if stamp, ok := stampRows[0]["updated_at"]; ok && stamp != nil && strings.TrimSpace(fmt.Sprint(stamp)) != "" {
			updatedAt = fmt.Sprint(stamp)
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok": true, "updated": true,
		"message":    "Expiry update completed for Security ID " + securityID + " with expiry " + expiry,
		"updated_at": updatedAt,
		"before":     before, "after": after,
	})
}

// apiDBQueryUpdateLTP updates only explicitly selected, allowlisted LTP tables.
func apiDBQueryUpdateLTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		dbQueryJSONError(w, http.StatusMethodNotAllowed, "Use POST")
		return
	}
	if err := dbQueryLoadEnv(); err != nil {
		dbQueryJSONError(w, http.StatusInternalServerError, "Cannot read database configuration")
		return
	}
	sess, ok := dbQueryCurrentSession(w, r)
	if !ok {
		return
	}
	var req struct {
		DB        string      `json:"db"`
		ScripCode string      `json:"scrip_code"`
		LTP       json.Number `json:"ltp"`
		Tables    []string    `json:"tables"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	dec.UseNumber()
	if err := dec.Decode(&req); err != nil {
		dbQueryJSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	if req.DB != sess.Server {
		dbQueryJSONError(w, http.StatusForbidden, "Selected server does not match connected session")
		return
	}
	scrip := strings.TrimSpace(req.ScripCode)
	if scrip == "" || len(scrip) > 30 {
		dbQueryJSONError(w, http.StatusBadRequest, "Enter a valid Scrip Code")
		return
	}
	ltpText := strings.TrimSpace(req.LTP.String())
	ltp, err := strconv.ParseFloat(ltpText, 64)
	if err != nil || math.IsNaN(ltp) || math.IsInf(ltp, 0) || ltp < 0 {
		dbQueryJSONError(w, http.StatusBadRequest, "Enter a valid non-negative LTP")
		return
	}
	allowed := map[string]struct {
		codeCol string
		cols    []string
	}{
		"L1_WATCH_ACTIVE": {"L1_SCRIP_CODE", []string{"L1_SCRIP_CODE", "L1_EXCHANGE", "L1_SEGMENT", "DATE_FORMAT(L1_ENTRY_TIME, '%Y-%m-%d %H:%i:%s') AS L1_ENTRY_TIME", "L1_LTP"}},
		"EQ_L1_WATCH":     {"L1_SCRIP_CODE", []string{"L1_SCRIP_CODE", "L1_EXCHANGE", "L1_SEGMENT", "DATE_FORMAT(L1_ENTRY_TIME, '%Y-%m-%d %H:%i:%s') AS L1_ENTRY_TIME", "L1_LTP"}},
		"DRV_L1_WATCH":    {"DL1_SCRIP_CODE", []string{"DL1_SCRIP_CODE", "DL1_EXCHANGE", "DL1_SEGMENT", "DATE_FORMAT(DL1_ENTRY_TIME, '%Y-%m-%d %H:%i:%s') AS DL1_ENTRY_TIME", "DL1_LTP"}},
	}
	selected := make([]string, 0, 3)
	seen := map[string]bool{}
	for _, t := range req.Tables {
		if _, ok := allowed[t]; !ok {
			dbQueryJSONError(w, http.StatusBadRequest, "Invalid LTP table selection")
			return
		}
		if !seen[t] {
			selected = append(selected, t)
			seen[t] = true
		}
	}
	if len(selected) == 0 {
		dbQueryJSONError(w, http.StatusBadRequest, "Select at least one table")
		return
	}
	idSQL, ltpSQL := dbQueryQuote(scrip), strconv.FormatFloat(ltp, 'f', -1, 64)
	before := make([]map[string]interface{}, 0, len(selected))
	for _, t := range selected {
		meta := allowed[t]
		q := "SELECT " + strings.Join(meta.cols, ", ") + " FROM " + t + " WHERE " + meta.codeCol + " = " + idSQL
		rows, e := dbQueryRows(sess.Server, q, meta.cols)
		if e != nil {
			dbQueryJSONError(w, http.StatusBadGateway, "Could not read "+t+" before update; nothing was changed")
			return
		}
		if len(rows) == 0 {
			dbQueryJSONError(w, http.StatusConflict, "No matching scrip in "+t+"; nothing was changed. Select only tables containing this scrip.")
			return
		}
		before = append(before, map[string]interface{}{"title": t, "columns": meta.cols, "rows": rows})
	}
	stmts := make([]string, 0, len(selected))
	for _, t := range selected {
		meta := allowed[t]
		ltpCol := "L1_LTP"
		if t == "DRV_L1_WATCH" {
			ltpCol = "DL1_LTP"
		}
		entryCol := "L1_ENTRY_TIME"
		if t == "DRV_L1_WATCH" {
			entryCol = "DL1_ENTRY_TIME"
		}
		stmts = append(stmts, "UPDATE "+t+" SET "+ltpCol+" = "+ltpSQL+", "+entryCol+" = NOW() WHERE "+meta.codeCol+" = "+idSQL)
	}
	if _, err = dbQueryRun(sess.Server, "START TRANSACTION; "+strings.Join(stmts, "; ")+"; COMMIT;"); err != nil {
		dbQueryJSONError(w, http.StatusBadGateway, "LTP update failed; verify table permissions and values before retrying")
		return
	}
	after := make([]map[string]interface{}, 0, len(selected))
	for _, t := range selected {
		meta := allowed[t]
		q := "SELECT " + strings.Join(meta.cols, ", ") + " FROM " + t + " WHERE " + meta.codeCol + " = " + idSQL
		rows, e := dbQueryRows(sess.Server, q, meta.cols)
		if e != nil {
			dbQueryJSONError(w, http.StatusBadGateway, "LTP update ran, but after-update verification failed; verify the values before retrying")
			return
		}
		after = append(after, map[string]interface{}{"title": t, "columns": meta.cols, "rows": rows})
	}
	updatedAt := time.Now().Format("2006-01-02 15:04:05")
	// Report the same database-server clock used by NOW() in the entry-time updates.
	if stampRows, stampErr := dbQueryRows(sess.Server, "SELECT DATE_FORMAT(NOW(), '%Y-%m-%d %H:%i:%s') AS updated_at", []string{"updated_at"}); stampErr == nil && len(stampRows) > 0 {
		if stamp, ok := stampRows[0]["updated_at"]; ok && stamp != nil && strings.TrimSpace(fmt.Sprint(stamp)) != "" {
			updatedAt = fmt.Sprint(stamp)
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "updated": true, "message": "LTP updated to " + ltpSQL + " for scrip " + scrip, "updated_at": updatedAt, "after": after})
}

// apiDBQueryManual executes one bounded, read-only SELECT statement. Cancellation
// propagates from the HTTP request to CommandContext, allowing the Stop button to stop mysql.
func apiDBQueryManual(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		dbQueryJSONError(w, http.StatusMethodNotAllowed, "Use POST")
		return
	}
	if err := dbQueryLoadEnv(); err != nil {
		dbQueryJSONError(w, http.StatusInternalServerError, "Cannot read database configuration")
		return
	}
	sess, ok := dbQueryCurrentSession(w, r)
	if !ok {
		return
	}
	var req struct {
		DB    string `json:"db"`
		Query string `json:"query"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 12000)).Decode(&req); err != nil {
		dbQueryJSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	if req.DB != sess.Server {
		dbQueryJSONError(w, http.StatusForbidden, "Selected server does not match connected session")
		return
	}
	query, err := dbQueryValidateManualSQL(req.Query)
	if err != nil {
		dbQueryJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	host, port, name, user, pass, err := dbQueryConfig(sess.Server)
	if err != nil {
		dbQueryJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "mysql", "--batch", "--raw", "--connect-timeout=5", "-h", host, "-P", port, "-u", user, name, "-e", query)
	cmd.Env = append(os.Environ(), "MYSQL_PWD="+pass)
	output, runErr := cmd.CombinedOutput()
	if ctx.Err() == context.Canceled {
		dbQueryJSONError(w, http.StatusRequestTimeout, "Query stopped")
		return
	}
	if ctx.Err() == context.DeadlineExceeded {
		dbQueryJSONError(w, http.StatusGatewayTimeout, "Query timed out after 15 seconds")
		return
	}
	if runErr != nil {
		dbQueryJSONError(w, http.StatusBadGateway, "Query failed; verify SQL, MySQL client, connection and permissions")
		return
	}
	out := strings.TrimSpace(string(output))
	if out == "" {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"columns": []string{}, "rows": []interface{}{}, "count": 0, "max_rows": dbQueryMaxRows, "query": query})
		return
	}
	lines := strings.Split(out, "\n")
	columns := strings.Split(lines[0], "\t")
	rows := make([]map[string]interface{}, 0, minInt(len(lines)-1, dbQueryMaxRows))
	for _, line := range lines[1:] {
		if len(rows) >= dbQueryMaxRows {
			break
		}
		fields := strings.Split(line, "\t")
		if len(fields) != len(columns) {
			continue
		}
		row := make(map[string]interface{}, len(columns))
		for i, col := range columns {
			if fields[i] == "NULL" {
				row[col] = nil
			} else {
				row[col] = fields[i]
			}
		}
		rows = append(rows, row)
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"columns": columns, "rows": rows, "count": len(rows), "max_rows": dbQueryMaxRows, "query": query})
}

func dbQueryValidateManualSQL(input string) (string, error) {
	q := strings.TrimSpace(input)
	if q == "" {
		return "", fmt.Errorf("Enter a SQL query")
	}
	if len(q) > 10000 {
		return "", fmt.Errorf("Query is too long (maximum 10,000 characters)")
	}
	// Keep one statement only; reject comments and statement separators to avoid hiding commands.
	if strings.Contains(q, ";") || strings.Contains(q, "--") || strings.Contains(q, "/*") || strings.Contains(q, "*/") || strings.Contains(q, "#") {
		return "", fmt.Errorf("Only one plain SELECT statement is allowed; comments and semicolons are blocked")
	}
	upper := strings.ToUpper(q)
	if !strings.HasPrefix(upper, "SELECT ") && !strings.HasPrefix(upper, "SELECT\n") && !strings.HasPrefix(upper, "SELECT\t") {
		return "", fmt.Errorf("Manual queries must start with SELECT")
	}
	for _, token := range []string{" INSERT ", " UPDATE ", " DELETE ", " DROP ", " ALTER ", " TRUNCATE ", " CREATE ", " REPLACE ", " GRANT ", " REVOKE ", " CALL ", " LOAD_FILE", " INTO OUTFILE", " INTO DUMPFILE", " LOCK TABLE", " UNLOCK TABLE", " FOR UPDATE", " FOR SHARE", " GET_LOCK(", " RELEASE_LOCK(", " SLEEP(", " BENCHMARK("} {
		if strings.Contains(" "+upper+" ", token) {
			return "", fmt.Errorf("Blocked SQL keyword or function detected: %s", strings.TrimSpace(token))
		}
	}
	// Add a row cap if one was not supplied; cap an explicit LIMIT to 100.
	if !strings.Contains(upper, " LIMIT ") {
		q += " LIMIT 100"
	} else {
		// Only numeric LIMIT forms are accepted so the cap can be enforced safely.
		loc := strings.LastIndex(upper, " LIMIT ")
		tail := strings.TrimSpace(upper[loc+7:])
		n := 0
		for _, ch := range tail {
			if ch < '0' || ch > '9' {
				break
			}
			n = n*10 + int(ch-'0')
			if n > 100 {
				n = 101
				break
			}
		}
		if n == 0 {
			return "", fmt.Errorf("Use a numeric LIMIT between 1 and 100")
		}
		if n > 100 {
			q = strings.TrimSpace(q[:loc]) + " LIMIT 100"
		}
	}
	return q, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
