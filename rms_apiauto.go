package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
	"html/template"
	"io"
	"io/fs"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
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
// AUTO FILE UPLOADER QA — merged as an additive feature
// UI remains a separate file: /KSH/auto-uploader.html
// API prefix: /api/auto-uploader/
// ============================================================

// TestCase is one uploader test: a single file, the dropdown selections that
// route it, and what the tester expects to happen.
type TestCase struct {
	ID       string `json:"id"`
	Exchange string `json:"exchange,omitempty"` // first toolbar dropdown; defaults to BO
	Segment  string `json:"segment"`
	FileType string `json:"fileType"`
	FilePath string `json:"filePath"`
	Expected string `json:"expected"` // "success" or "reject"
	// Optional exact assertions. nil / empty means "not checked".
	ExpectedSuccess *int   `json:"expectedSuccess,omitempty"`
	ExpectedReject  *int   `json:"expectedReject,omitempty"`
	ReasonContains  string `json:"reasonContains,omitempty"`
}

// TestResult is the outcome of one test case.
type TestResult struct {
	ID                      string `json:"id"`
	Segment                 string `json:"segment"`
	FileType                string `json:"fileType"`
	FileName                string `json:"fileName"`
	Expected                string `json:"expected"`
	Actual                  string `json:"actual"`
	Success                 int    `json:"successCount"`
	Reject                  int    `json:"rejectCount"`
	RejectionReason         string `json:"rejectionReason"`
	RejectionScreenshotPath string `json:"rejectionScreenshotPath"`
	ErrorText               string `json:"errorText"`
	ScreenshotPath          string `json:"screenshotPath"`
	Status                  string `json:"status"` // PASS, FAIL, ERROR, SKIPPED
	StartedAt               string `json:"startedAt"`
	FinishedAt              string `json:"finishedAt"`
}

// RunOptions are the per-run settings sent by the dashboard.
type RunOptions struct {
	Username          string
	Password          string
	UploaderURL       string
	CapturePass       bool
	CaptureFail       bool
	CaptureRejection  bool
	CaptureFinal      bool
	UploadTimeout     time.Duration // upload dialog must close within this
	ResultTimeout     time.Duration // new history row must appear within this
	RefreshEvery      time.Duration // how often the Refresh button is re-clicked while waiting
	SelectSettleDelay time.Duration // pause after each dropdown change
}

func defaultOptions() RunOptions {
	return RunOptions{
		UploadTimeout:     2 * time.Minute,
		ResultTimeout:     5 * time.Minute,
		RefreshEvery:      3 * time.Second,
		SelectSettleDelay: 700 * time.Millisecond,
	}
}

// App holds runner state shared with the HTTP handlers.
type AutoUploaderApp struct {
	mu              sync.Mutex
	busy            bool
	stop            chan struct{}
	results         []TestResult
	finalScreenshot string
	browser         string // Chrome remote-debugging base URL
	reportsDir      string
	defaultURL      string // uploader URL used if no uploader tab is open
	port            string
}

func (a *AutoUploaderApp) status(w http.ResponseWriter, r *http.Request) {
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(a.browser + "/json/version")
	if err != nil {
		writeJSON(w, map[string]any{"connected": false, "message": "Browser debugging endpoint is not reachable at " + a.browser + "."})
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	writeJSON(w, map[string]any{"connected": resp.StatusCode == 200, "message": "Browser debugging endpoint responded.", "defaultUrl": a.defaultURL})
}

func (a *AutoUploaderApp) inspect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	data, err := a.inspectPage(ctx)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, data)
}

func (a *AutoUploaderApp) run(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	var req struct {
		Tests                       []TestCase `json:"tests"`
		Username                    string     `json:"username"`
		Password                    string     `json:"password"`
		TargetServer                string     `json:"targetServer"`
		UploadTimeoutSec            int        `json:"uploadTimeoutSec"`
		ResultTimeoutSec            int        `json:"resultTimeoutSec"`
		CaptureSuccessScreenshots   bool       `json:"captureSuccessScreenshots"`
		CaptureFailureScreenshots   bool       `json:"captureFailureScreenshots"`
		CaptureRejectionScreenshots bool       `json:"captureRejectionScreenshots"`
		CaptureFinalScreenshot      bool       `json:"captureFinalScreenshot"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), 400)
		return
	}
	if len(req.Tests) == 0 {
		http.Error(w, "Add at least one test case", 400)
		return
	}
	if len(req.Tests) > 200 {
		http.Error(w, "Maximum 200 test cases per run", 400)
		return
	}
	opts := defaultOptions()
	opts.Username, opts.Password = req.Username, req.Password
	serverURLs := map[string]string{
		"74":  "http://172.16.101.74:8080/RupeeFileUpload/#/login",
		"168": "http://172.16.101.168:8080/RupeeFileUpload/#/login",
		"170": "http://172.16.101.170:8080/RupeeFileUpload/#/login",
	}
	selectedURL, ok := serverURLs[strings.TrimSpace(req.TargetServer)]
	if !ok {
		http.Error(w, "Select target server 74, 168 or 170", http.StatusBadRequest)
		return
	}
	opts.UploaderURL = selectedURL
	opts.CapturePass, opts.CaptureFail = req.CaptureSuccessScreenshots, req.CaptureFailureScreenshots
	opts.CaptureRejection, opts.CaptureFinal = req.CaptureRejectionScreenshots, req.CaptureFinalScreenshot
	if req.UploadTimeoutSec > 0 {
		opts.UploadTimeout = time.Duration(min(req.UploadTimeoutSec, 3600)) * time.Second
	}
	if req.ResultTimeoutSec > 0 {
		opts.ResultTimeout = time.Duration(min(req.ResultTimeoutSec, 7200)) * time.Second
	}

	a.mu.Lock()
	if a.busy {
		a.mu.Unlock()
		http.Error(w, "A test run is already in progress", 409)
		return
	}
	a.busy = true
	a.stop = make(chan struct{})
	stop := a.stop
	a.results = nil
	a.finalScreenshot = ""
	a.mu.Unlock()
	go func() {
		defer func() { a.mu.Lock(); a.busy = false; a.mu.Unlock() }()
		a.execute(req.Tests, stop, opts)
	}()
	writeJSON(w, map[string]any{"started": true, "count": len(req.Tests)})
}

func (a *AutoUploaderApp) stopRun(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.busy && a.stop != nil {
		close(a.stop)
		a.stop = nil
	}
	writeJSON(w, map[string]any{"stopping": true})
}

func (a *AutoUploaderApp) clearResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, map[string]any{"error": "POST only"})
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.busy {
		writeJSON(w, map[string]any{"error": "Cannot clear results while a test run is in progress"})
		return
	}
	a.results = nil
	a.finalScreenshot = ""
	// Evidence files stay on disk; this only clears dashboard results.
	writeJSON(w, map[string]any{"cleared": true})
}

func (a *AutoUploaderApp) getResults(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	res := a.results
	if res == nil {
		res = []TestResult{}
	}
	writeJSON(w, map[string]any{"busy": a.busy, "results": res, "finalScreenshotPath": a.finalScreenshot})
}

func (a *AutoUploaderApp) uploadVideo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	const limit = 800 << 20
	if r.ContentLength > limit {
		http.Error(w, "Video exceeds 800 MB limit", 413)
		return
	}
	dir := filepath.Join(a.reportsDir, "videos")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	name := "screen_recording_" + time.Now().Format("20060102_150405") + ".webm"
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer f.Close()
	n, err := io.Copy(f, io.LimitReader(r.Body, limit))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if n == 0 {
		http.Error(w, "Empty video", 400)
		return
	}
	writeJSON(w, map[string]any{"saved": true, "path": "videos/" + name, "bytes": n})
}

func (a *AutoUploaderApp) addResult(r TestResult) {
	a.mu.Lock()
	a.results = append(a.results, r)
	a.mu.Unlock()
	// Persist after every test so reports survive an interrupted run.
	_ = a.writeReports()
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON: %v", err)
	}
}

func now() string { return time.Now().Format(time.RFC3339) }

func itoa(n int) string { return fmt.Sprint(n) }

// ---------------------------------------------------------------------------
// Browser attachment
// ---------------------------------------------------------------------------

type devtoolsTarget struct {
	ID   string `json:"id"`
	URL  string `json:"url"`
	Type string `json:"type"`
}

func (a *AutoUploaderApp) listTargets() ([]devtoolsTarget, error) {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(a.browser + "/json/list")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var targets []devtoolsTarget
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return nil, err
	}
	return targets, nil
}

// findUploaderTarget returns the id of the RupeeFileUpload tab. If none is open
// and uploaderURL is set, a new tab is opened for it.
func (a *AutoUploaderApp) findUploaderTarget(uploaderURL string) (string, error) {
	targets, err := a.listTargets()
	if err != nil {
		return "", fmt.Errorf("cannot reach the browser debugging endpoint %s: %w (start Chrome/Edge with --remote-debugging-port=9222)", a.browser, err)
	}
	targetHost := ""
	if parsed, parseErr := url.Parse(uploaderURL); parseErr == nil {
		targetHost = strings.ToLower(parsed.Host)
	}
	for _, t := range targets {
		// Match the selected UAT server as well as the uploader path; do not
		// accidentally run Server 74 test cases in an already-open Server 170 tab.
		if t.Type == "page" && strings.Contains(strings.ToLower(t.URL), "rupeefileupload") {
			if targetHost == "" || strings.Contains(strings.ToLower(t.URL), targetHost) {
				return t.ID, nil
			}
		}
	}
	if uploaderURL == "" {
		return "", errors.New("no RupeeFileUpload tab found in the debugging browser. Open the uploader in that browser, or select a target server in the dashboard so the runner can open it")
	}
	req, _ := http.NewRequest(http.MethodPut, a.browser+"/json/new?"+uploaderURL, nil)
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not open the uploader URL in the browser: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var t devtoolsTarget
	if err := json.Unmarshal(body, &t); err != nil || t.ID == "" {
		return "", fmt.Errorf("browser refused to open a new tab (%s): %s", resp.Status, strings.TrimSpace(string(body)))
	}
	time.Sleep(1500 * time.Millisecond)
	return t.ID, nil
}

// newUploaderContext attaches to the uploader tab. The returned cancel function
// only detaches this runner; it never closes the user's tab.
func (a *AutoUploaderApp) newUploaderContext(uploaderURL string) (context.Context, context.CancelFunc, error) {
	id, err := a.findUploaderTarget(uploaderURL)
	if err != nil {
		return nil, nil, err
	}
	allocCtx, allocCancel := chromedp.NewRemoteAllocator(context.Background(), a.browser)
	ctx, ctxCancel := chromedp.NewContext(allocCtx, chromedp.WithTargetID(target.ID(id)))
	// Cancelling a context attached to an existing target closes that tab, so
	// cancellation is deliberately a no-op; the websocket is released when the
	// process exits. Only a handful of connections are ever created.
	_ = allocCancel
	_ = ctxCancel
	return ctx, func() {}, nil
}

func (a *AutoUploaderApp) inspectPage(ctx context.Context) (map[string]any, error) {
	browserCtx, cancel, err := a.newUploaderContext(a.defaultURL)
	if err != nil {
		return nil, err
	}
	defer cancel()
	var data map[string]any
	js := `(() => ({url:location.href,title:document.title,
	  selects:Array.from(document.querySelectorAll('select')).map((s,i)=>({index:i,value:s.value,text:(s.options[s.selectedIndex]||{}).text||'',options:Array.from(s.options).map(o=>({text:o.text.trim(),value:o.value})).slice(0,150)})),
	  fileInputs:Array.from(document.querySelectorAll('input[type=file]')).map((e,i)=>({index:i,accept:e.accept,multiple:e.multiple,outerHTML:e.outerHTML.slice(0,300)})),
	  buttons:Array.from(document.querySelectorAll('button,[role=button],input[type=button],input[type=submit]')).map((e,i)=>({index:i,text:(e.innerText||e.value||e.getAttribute('aria-label')||'').trim(),disabled:!!e.disabled})).filter(x=>x.text).slice(0,100),
	  tableText:Array.from(document.querySelectorAll('table')).map(t=>t.innerText.slice(0,1500)).slice(0,5)}))()`
	if err := chromedp.Run(browserCtx, chromedp.Evaluate(js, &data)); err != nil {
		return nil, err
	}
	return data, nil
}

// ---------------------------------------------------------------------------
// Run orchestration
// ---------------------------------------------------------------------------

func stopped(stop <-chan struct{}) bool {
	select {
	case <-stop:
		return true
	default:
		return false
	}
}

// sleepOrStop sleeps for d and reports false if the run was stopped meanwhile.
func sleepOrStop(stop <-chan struct{}, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-stop:
		return false
	case <-t.C:
		return true
	}
}

func normExpected(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "reject", "rejected", "invalid", "fail", "failed":
		return "reject"
	default:
		return "success"
	}
}

func (a *AutoUploaderApp) execute(tests []TestCase, stop <-chan struct{}, opts RunOptions) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("run panicked: %v", r)
			a.addResult(TestResult{ID: "RUNNER", Expected: "Run completes", Actual: "Runner crashed", Status: "ERROR", ErrorText: fmt.Sprint(r), StartedAt: now(), FinishedAt: now()})
		}
		_ = a.writeReports()
	}()

	ctx, cancel, err := a.newUploaderContext(opts.UploaderURL)
	if err != nil {
		a.addResult(TestResult{ID: "CONNECTION", Expected: "Uploader tab available", Actual: "Not connected", Status: "ERROR", ErrorText: err.Error(), StartedAt: now(), FinishedAt: now()})
		return
	}
	defer cancel()

	if err := ensureUploaderLogin(ctx, opts.Username, opts.Password); err != nil {
		res := TestResult{ID: "LOGIN", Expected: "Uploader login successful", Actual: "Login failed", Status: "ERROR", ErrorText: err.Error(), StartedAt: now(), FinishedAt: now()}
		if shot, e := a.captureScreenshot(ctx, "LOGIN"); e == nil {
			res.ScreenshotPath = shot
		}
		a.addResult(res)
		return
	}
	if err := waitForUploader(ctx, stop, 45*time.Second); err != nil {
		res := TestResult{ID: "UPLOADER", Expected: "Uploader page ready", Actual: "Not ready", Status: "ERROR", ErrorText: err.Error(), StartedAt: now(), FinishedAt: now()}
		if shot, e := a.captureScreenshot(ctx, "UPLOADER"); e == nil {
			res.ScreenshotPath = shot
		}
		a.addResult(res)
		return
	}

	for i, tc := range tests {
		id := tc.ID
		if id == "" {
			id = fmt.Sprintf("TC%03d", i+1)
		}
		res := TestResult{ID: id, Segment: tc.Segment, FileType: tc.FileType, FileName: filepath.Base(tc.FilePath), Expected: normExpected(tc.Expected), StartedAt: now(), Status: "ERROR"}
		if stopped(stop) {
			for _, rest := range tests[i:] {
				rid := rest.ID
				if rid == "" {
					rid = "TC"
				}
				a.addResult(TestResult{ID: rid, Segment: rest.Segment, FileType: rest.FileType, FileName: filepath.Base(rest.FilePath), Expected: normExpected(rest.Expected), Status: "SKIPPED", ErrorText: "Run stopped by user before this test started.", StartedAt: now(), FinishedAt: now()})
			}
			break
		}

		// Pre-flight validation: fail fast with a clear message, never touch the UI.
		if msg := validateCase(tc); msg != "" {
			res.ErrorText = msg
			a.attachShot(ctx, &res, id+"_ERROR", opts.CaptureFail)
			res.FinishedAt = now()
			a.addResult(res)
			continue
		}

		resetUI(ctx)
		oc := a.executeOne(ctx, tc, opts, stop)
		res.Actual, res.Success, res.Reject, res.ErrorText = oc.Actual, oc.Success, oc.Reject, oc.ErrorText
		if oc.Err != nil && strings.TrimSpace(res.ErrorText) == "" {
			res.ErrorText = oc.Err.Error()
		}

		// Open the Rejection dialog for exactly this run's history row.
		if oc.Err == nil && oc.Reject > 0 {
			res.RejectionReason, res.RejectionScreenshotPath = a.readRejectionReasons(ctx, oc.RowKey, opts.CaptureRejection, id)
			if res.RejectionReason == "" {
				res.RejectionReason = "Reject count is greater than zero, but the rejection reason could not be read from the dialog."
			}
		}

		switch {
		case oc.Err != nil:
			res.Status = "ERROR"
		default:
			if ok, why := evaluate(tc, oc.Actual, oc.Success, oc.Reject, res.RejectionReason); ok {
				res.Status = "PASS"
				res.ErrorText = ""
			} else {
				res.Status = "FAIL"
				res.ErrorText = why
			}
		}

		// Evidence for every completed test, captured before the next test starts.
		if (res.Status == "PASS" && opts.CapturePass) || ((res.Status == "FAIL" || res.Status == "ERROR") && opts.CaptureFail) {
			a.attachShot(ctx, &res, id+"_"+res.Status, true)
		}
		res.FinishedAt = now()
		a.addResult(res)
		resetUI(ctx)
	}

	if opts.CaptureFinal {
		if shot, e := a.captureScreenshot(ctx, "FINAL_UPLOADER_MAIN_WINDOW"); e == nil {
			a.mu.Lock()
			a.finalScreenshot = shot
			a.mu.Unlock()
		} else {
			log.Printf("final uploader screenshot failed: %v", e)
		}
	}
}

func (a *AutoUploaderApp) attachShot(ctx context.Context, res *TestResult, name string, want bool) {
	if !want {
		return
	}
	shot, err := a.captureScreenshot(ctx, name)
	if err != nil {
		if res.ErrorText != "" {
			res.ErrorText += " | "
		}
		res.ErrorText += "Screenshot capture failed: " + err.Error()
		return
	}
	res.ScreenshotPath = shot
}

func validateCase(tc TestCase) string {
	if strings.TrimSpace(tc.Segment) == "" || strings.TrimSpace(tc.FileType) == "" {
		return "Segment and File type are required."
	}
	info, err := os.Stat(tc.FilePath)
	if err != nil {
		return "File not found or not accessible: " + tc.FilePath
	}
	if info.IsDir() {
		return "The test path points to a folder, not a file. Enter the complete path including filename and extension: " + tc.FilePath
	}
	if info.Size() == 0 {
		return "The test file is empty (0 bytes): " + tc.FilePath
	}
	return ""
}

// evaluate compares the uploader's outcome with the test's expectation and
// returns a human-readable reason when it does not match.
func evaluate(tc TestCase, actualStatus string, success, reject int, reason string) (bool, string) {
	var problems []string
	if strings.Contains(strings.ToLower(actualStatus), "fail") {
		problems = append(problems, fmt.Sprintf("Uploader reported status %q instead of completing.", actualStatus))
	}
	if normExpected(tc.Expected) == "reject" {
		if reject < 1 {
			problems = append(problems, fmt.Sprintf("Expected the file to be rejected, but the reject count is %d (success %d).", reject, success))
		}
	} else {
		if success < 1 {
			problems = append(problems, fmt.Sprintf("Expected a successful upload, but the success count is %d.", success))
		}
		if reject != 0 && tc.ExpectedReject == nil {
			problems = append(problems, fmt.Sprintf("Expected no rejections, but the reject count is %d.", reject))
		}
	}
	if tc.ExpectedSuccess != nil && success != *tc.ExpectedSuccess {
		problems = append(problems, fmt.Sprintf("Success count %d, expected %d.", success, *tc.ExpectedSuccess))
	}
	if tc.ExpectedReject != nil && reject != *tc.ExpectedReject {
		problems = append(problems, fmt.Sprintf("Reject count %d, expected %d.", reject, *tc.ExpectedReject))
	}
	if want := strings.TrimSpace(tc.ReasonContains); want != "" && !strings.Contains(strings.ToLower(reason), strings.ToLower(want)) {
		problems = append(problems, fmt.Sprintf("Rejection reason does not contain %q (got: %q).", want, reason))
	}
	if len(problems) > 0 {
		return false, strings.Join(problems, " ")
	}
	return true, ""
}

// ---------------------------------------------------------------------------
// Login and page readiness
// ---------------------------------------------------------------------------

// loginWait is how long to wait for the login form to disappear after submit.
var loginWait = 20 * time.Second

const jsVisible = `const visible=e=>{if(!e)return false;const r=e.getBoundingClientRect(),s=getComputedStyle(e);return r.width>0&&r.height>0&&s.display!=='none'&&s.visibility!=='hidden';};`

func jsStr(s string) string { b, _ := json.Marshal(s); return string(b) }

// ensureUploaderLogin skips login when the session is already authenticated.
// Credentials are used only for this run and are never written to disk or reports.
func ensureUploaderLogin(ctx context.Context, username, password string) error {
	var passwordVisible bool
	inspect := `(() => {` + jsVisible + ` return Array.from(document.querySelectorAll('input[type="password"]')).some(visible);})()`
	if err := chromedp.Run(ctx, chromedp.Evaluate(inspect, &passwordVisible)); err != nil {
		return fmt.Errorf("could not inspect uploader login page: %w", err)
	}
	if !passwordVisible {
		return nil // already logged in
	}
	if strings.TrimSpace(username) == "" || password == "" {
		return errors.New("uploader login is required, but username/password were not entered in the QA dashboard")
	}
	js := `(() => {
	  const username=` + jsStr(username) + `, password=` + jsStr(password) + `;` + jsVisible + `
	  const pw=Array.from(document.querySelectorAll('input[type="password"]')).find(visible);
	  if(!pw)return {ok:false,reason:'Visible password field not found'};
	  const fields=Array.from(document.querySelectorAll('input')).filter(e=>visible(e)&&e.type!=='password'&&!['hidden','submit','button','checkbox','radio','file'].includes((e.type||'').toLowerCase()));
	  const user=fields.find(e=>/user|login|email|account/i.test([e.name,e.id,e.placeholder,e.getAttribute('aria-label')].join(' ')))||fields[0];
	  if(!user)return {ok:false,reason:'Username field not found'};
	  const set=(e,v)=>{e.focus();const setter=Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value')?.set;if(setter)setter.call(e,v);else e.value=v;e.dispatchEvent(new Event('input',{bubbles:true}));e.dispatchEvent(new Event('change',{bubbles:true}));e.dispatchEvent(new Event('blur',{bubbles:true}));};
	  set(user,username);set(pw,password);
	  const buttons=Array.from(document.querySelectorAll('button,[role="button"],input[type="submit"],input[type="button"]')).filter(visible);
	  const btn=buttons.find(e=>/^(log ?in|sign ?in|submit|continue)$/i.test((e.innerText||e.value||'').trim()))||buttons.find(e=>/log ?in|sign ?in/i.test((e.innerText||e.value||'').trim()));
	  if(btn){btn.click();return {ok:true};}
	  const form=pw.closest('form');if(form){form.requestSubmit?form.requestSubmit():form.submit();return {ok:true};}
	  return {ok:false,reason:'Login/Sign in button or login form not found'};
	})()`
	var result struct {
		OK     bool   `json:"ok"`
		Reason string `json:"reason"`
	}
	if err := chromedp.Run(ctx, chromedp.Evaluate(js, &result)); err != nil {
		return fmt.Errorf("could not submit uploader login: %w", err)
	}
	if !result.OK {
		return errors.New(result.Reason)
	}
	deadline := time.Now().Add(loginWait)
	for time.Now().Before(deadline) {
		time.Sleep(500 * time.Millisecond)
		var still bool
		if err := chromedp.Run(ctx, chromedp.Evaluate(inspect, &still)); err == nil && !still {
			return nil
		}
	}
	return fmt.Errorf("login was submitted, but the password field is still visible after %s; check the credentials, CAPTCHA or additional login steps", loginWait)
}

// waitForUploader waits until the uploader toolbar (Choose File) is visible,
// switching to the "File Upload" tab if the app opened on another tab.
func waitForUploader(ctx context.Context, stop <-chan struct{}, timeout time.Duration) error {
	js := `(() => {` + jsVisible + `
	  const norm=s=>(s||'').replace(/\s+/g,' ').trim().toLowerCase();
	  if(Array.from(document.querySelectorAll('button,[role=button],label,a')).some(e=>visible(e)&&norm(e.innerText||e.textContent).includes('choose file')))return 'ready';
	  const tab=Array.from(document.querySelectorAll('body *')).find(e=>e.children.length===0&&visible(e)&&norm(e.innerText||e.textContent)==='file upload');
	  if(tab){tab.click();return 'clicked-tab';}
	  return 'missing';})()`
	deadline := time.Now().Add(timeout)
	last := ""
	for time.Now().Before(deadline) {
		if err := chromedp.Run(ctx, chromedp.Evaluate(js, &last)); err == nil && last == "ready" {
			return nil
		}
		if !sleepOrStop(stop, 500*time.Millisecond) {
			return errors.New("stopped by user")
		}
	}
	return fmt.Errorf("the uploader toolbar (Choose File button) did not appear within %s after login (last state: %s)", timeout, last)
}

// resetUI closes any popup left open by a previous step (confirmation popup,
// Error View, upload dialog) so each test starts from a clean page.
func resetUI(ctx context.Context) {
	js := `(() => {` + jsVisible + `
	  let n=0;
	  for(const b of Array.from(document.querySelectorAll('button,[role=button],a,input[type=button]')).filter(visible)){
	    const t=(b.innerText||b.value||'').trim().toLowerCase(), al=(b.getAttribute('aria-label')||'').toLowerCase();
	    if(t==='close'||t==='×'||al.includes('close')){b.click();n++;}
	  }
	  return n;})()`
	var n int
	_ = chromedp.Run(ctx, chromedp.Evaluate(js, &n))
	if n > 0 {
		time.Sleep(300 * time.Millisecond)
	}
}

// ---------------------------------------------------------------------------
// One test
// ---------------------------------------------------------------------------

type outcome struct {
	Actual    string
	Success   int
	Reject    int
	RowKey    string // normalized text of this run's history row
	ErrorText string
	Err       error
}

func fail(actual, text string, err error) outcome {
	return outcome{Actual: actual, ErrorText: text, Err: err}
}

type historyRow struct {
	Key     string `json:"key"`
	Text    string `json:"text"`
	Date    string `json:"date"`
	Status  string `json:"status"`
	Success int    `json:"success"`
	Reject  int    `json:"reject"`
}

const jsAllRowKeys = `Array.from(document.querySelectorAll('table tbody tr')).map(r=>(r.innerText||'').replace(/\s+/g,' ').trim().toLowerCase())`

func jsMatchingRows(fileName, process string) string {
	return `(() => {
	  const norm=s=>(s||'').replace(/\s+/g,' ').trim().toLowerCase();
	  const wantedFile=norm(` + jsStr(fileName) + `), wantedProcess=norm(` + jsStr(process) + `);
	  const out=[];
	  for(const table of document.querySelectorAll('table')){
	    const hs=Array.from(table.querySelectorAll('thead th, thead td, [role=columnheader]')).map(e=>norm(e.innerText));
	    const idx=(q,fb)=>{const i=hs.findIndex(h=>h.includes(q));return i>=0?i:fb;};
	    for(const tr of table.querySelectorAll('tbody tr')){
	      const cells=Array.from(tr.querySelectorAll('td')).map(e=>(e.innerText||'').trim());
	      const lc=cells.map(norm);
	      // Exact cell match: "VALID_HOLDING.txt" must not match "INVALID_HOLDING.txt".
	      if(!lc.includes(wantedFile)||!lc.includes(wantedProcess))continue;
	      const status=cells[idx('status',2)]||'';
	      if(!/complete|fail|error/i.test(status))continue;   // still processing: not a final row yet
	      const num=i=>{const n=parseInt((cells[i]||'0').replace(/[^0-9-]/g,''),10);return Number.isFinite(n)?n:0;};
	      out.push({key:norm(tr.innerText),text:(tr.innerText||'').trim(),date:cells[0]||'',status:status,success:num(idx('success count',3)),reject:num(idx('reject count',4))});
	    }
	  }
	  return out;})()`
}

// uploadButtonJS returns script that finds the dialog's "Upload Files" button
// and either reports its state or clicks it.
func uploadButtonJS(click bool) string {
	tail := `return {found:!!e,disabled:!!(e&&e.disabled)};`
	if click {
		tail = `if(e&&!e.disabled){e.click();return true;}return false;`
	}
	return `(() => {` + jsVisible + `
	  const dialogs=Array.from(document.querySelectorAll('[role="dialog"],.modal,.dialog,.ov')).filter(visible);
	  const scope=dialogs[dialogs.length-1]||document;
	  const e=Array.from(scope.querySelectorAll('button,[role=button],input[type=button],input[type=submit]')).find(x=>(x.innerText||x.value||'').trim().toLowerCase()==='upload files');
	  ` + tail + `})()`
}

func (a *AutoUploaderApp) executeOne(ctx context.Context, tc TestCase, opts RunOptions, stop <-chan struct{}) outcome {
	var pageURL string
	if err := chromedp.Run(ctx, chromedp.Location(&pageURL)); err != nil {
		return fail("", "Could not inspect the uploader tab: "+err.Error(), err)
	}
	if !strings.Contains(strings.ToLower(pageURL), "rupeefileupload") {
		return fail("", "The attached tab ("+pageURL+") does not look like the uploader.", errors.New("wrong active tab"))
	}
	if err := waitForUploader(ctx, stop, 20*time.Second); err != nil {
		return fail("", err.Error(), err)
	}

	// 1) Dropdowns, located by position on the same row as "Choose File".
	exchange := strings.TrimSpace(tc.Exchange)
	if exchange == "" {
		exchange = "BO"
	}
	for i, wanted := range []string{exchange, tc.Segment, tc.FileType} {
		if e := selectUploaderDropdown(ctx, i, wanted); e != nil {
			return fail("", e.Error(), e)
		}
		if !sleepOrStop(stop, opts.SelectSettleDelay) {
			return fail("", "Stopped by user.", errors.New("stopped"))
		}
	}

	// 2) Open the upload dialog. The page-level "Auto Upload" button is never
	//    used as a fallback because it can submit stale/multiple selections.
	var clicked bool
	if err := chromedp.Run(ctx, chromedp.Evaluate(`(() => { const es=Array.from(document.querySelectorAll('button,[role=button],label')); const e=es.find(x=>(x.innerText||'').trim().toLowerCase().includes('choose file')); if(e){e.click();return true;} return false; })()`, &clicked)); err != nil {
		return fail("", "Could not open file picker: "+err.Error(), err)
	}
	if !clicked {
		return fail("", "Could not find the Choose File control.", errors.New("choose file missing"))
	}
	time.Sleep(400 * time.Millisecond)

	// 3) Mark exactly one file input (CDP can fill hidden inputs) and assign the file.
	var marked bool
	markJS := `(() => {
	  const all=Array.from(document.querySelectorAll('input[type="file"]'));
	  for(const x of all)x.removeAttribute('data-qa-upload-target');
	  const inDialog=x=>{let p=x;while(p&&p!==document.body){const role=(p.getAttribute('role')||'').toLowerCase(),cl=(p.className||'').toString().toLowerCase();if(role==='dialog'||cl.includes('modal')||cl.includes('dialog')||cl.includes('dropzone')||cl.includes('ov'))return true;p=p.parentElement;}return false;};
	  if(!all.length)return false;
	  const chosen=all.filter(inDialog).pop()||all[all.length-1];
	  chosen.setAttribute('data-qa-upload-target','true');
	  return true;})()`
	if err := chromedp.Run(ctx, chromedp.Evaluate(markJS, &marked)); err != nil || !marked {
		if err == nil {
			err = errors.New("no input[type=file] element exists in the page")
		}
		return fail("", "Could not identify the uploader file input (the app may use a native OS picker instead of a DOM file input): "+err.Error(), err)
	}
	if err := chromedp.Run(ctx, chromedp.SetUploadFiles(`input[type="file"][data-qa-upload-target="true"]`, []string{tc.FilePath})); err != nil {
		return fail("", "Could not assign the test file: "+err.Error(), err)
	}
	var chosenFile struct {
		Count int      `json:"count"`
		Names []string `json:"names"`
	}
	verifyJS := `(() => {const e=document.querySelector('input[type="file"][data-qa-upload-target="true"]'); if(!e) return {count:0,names:[]}; e.dispatchEvent(new Event('input',{bubbles:true})); e.dispatchEvent(new Event('change',{bubbles:true})); return {count:e.files?e.files.length:0,names:e.files?Array.from(e.files).map(f=>f.name):[]};})()`
	if err := chromedp.Run(ctx, chromedp.Evaluate(verifyJS, &chosenFile)); err != nil {
		return fail("", "Could not verify selected file: "+err.Error(), err)
	}
	if chosenFile.Count != 1 {
		return fail("", fmt.Sprintf("Safety check stopped the upload: expected exactly 1 selected file but the input reports %d (%s). No upload button was clicked.", chosenFile.Count, strings.Join(chosenFile.Names, ", ")), errors.New("single-file selection check failed"))
	}
	if !strings.EqualFold(chosenFile.Names[0], filepath.Base(tc.FilePath)) {
		return fail("", fmt.Sprintf("Safety check stopped the upload: selected file %q does not match %q. No upload button was clicked.", chosenFile.Names[0], filepath.Base(tc.FilePath)), errors.New("selected filename mismatch"))
	}

	// 4) Remember existing history rows so an old row for the same file is never
	//    mistaken for this run's result.
	var baselineKeys []string
	_ = chromedp.Run(ctx, chromedp.Evaluate(jsAllRowKeys, &baselineKeys))
	baseline := map[string]int{}
	for _, k := range baselineKeys {
		baseline[k]++
	}

	// 5) Click "Upload Files" in the dialog once it is enabled.
	stateJS := uploadButtonJS(false)
	var btn struct {
		Found    bool `json:"found"`
		Disabled bool `json:"disabled"`
	}
	ready := time.Now().Add(8 * time.Second)
	for time.Now().Before(ready) {
		_ = chromedp.Run(ctx, chromedp.Evaluate(stateJS, &btn))
		if btn.Found && !btn.Disabled {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	if !btn.Found || btn.Disabled {
		return fail("File Selected - Upload Not Started", "The file is selected, but the dialog's Upload Files button stayed disabled. No upload was submitted; check the uploader's file-selection validation.", errors.New("upload files button disabled"))
	}
	clickJS := uploadButtonJS(true)
	var uploadClicked bool
	if err := chromedp.Run(ctx, chromedp.Evaluate(clickJS, &uploadClicked)); err != nil {
		return fail("", "Could not click the dialog's Upload Files button: "+err.Error(), err)
	}
	if !uploadClicked {
		return fail("File Selected - Upload Not Started", "The dialog's Upload Files button was not found or not enabled.", errors.New("modal upload button missing"))
	}

	// 6) The dialog must close; otherwise nothing was submitted.
	closed := false
	for deadline := time.Now().Add(opts.UploadTimeout); time.Now().Before(deadline); {
		var open bool
		_ = chromedp.Run(ctx, chromedp.Evaluate(`(() => {`+jsVisible+` return Array.from(document.querySelectorAll('body *')).some(e=>{if(e.children.length>0)return false;const t=(e.innerText||'').trim().toLowerCase();if(!t.includes('drag files here')&&t!=='upload files')return false;return visible(e);});})()`, &open))
		if !open {
			closed = true
			break
		}
		if !sleepOrStop(stop, 300*time.Millisecond) {
			return fail("", "Stopped by user while the upload dialog was open.", errors.New("stopped"))
		}
	}
	if !closed {
		return fail("Upload Dialog Still Open", fmt.Sprintf("The upload dialog did not close within %s. The test was not marked successful.", opts.UploadTimeout), errors.New("upload dialog remained open"))
	}

	// 7) Dismiss the "Request accepted successfully kindly refresh..." popup.
	confirmJS := `(() => {` + jsVisible + `
	  const all=Array.from(document.querySelectorAll('body *')).filter(visible);
	  const msg=all.find(e=>e.children.length===0&&/request accepted successfully|kindly refresh for further status/i.test(e.innerText||''));
	  if(!msg)return false;
	  let p=msg;for(let i=0;i<6&&p;i++,p=p.parentElement){const c=Array.from(p.querySelectorAll('button,[role=button],input[type=button]')).filter(visible).find(b=>(b.innerText||b.value||'').trim().toLowerCase()==='close');if(c){c.click();return true;}}
	  const c=Array.from(document.querySelectorAll('button,[role=button],input[type=button]')).find(b=>visible(b)&&(b.innerText||b.value||'').trim().toLowerCase()==='close');
	  if(c){c.click();}return true;})()`
	for deadline := time.Now().Add(8 * time.Second); time.Now().Before(deadline); {
		var found bool
		_ = chromedp.Run(ctx, chromedp.Evaluate(confirmJS, &found))
		if found {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	time.Sleep(300 * time.Millisecond)

	// 8) Poll for a NEW completed history row. The app only updates after the
	//    Refresh button is pressed, so Refresh is re-clicked on every interval
	//    until the row is final or the result timeout expires.
	refreshJS := `(() => {` + jsVisible + `
	  const b=Array.from(document.querySelectorAll('button,[role=button],input[type=button]')).find(e=>visible(e)&&(e.innerText||e.value||'').trim().toLowerCase()==='refresh');
	  if(b){b.click();return true;}return false;})()`
	matchJS := jsMatchingRows(filepath.Base(tc.FilePath), tc.FileType)
	var lastRefresh time.Time
	for deadline := time.Now().Add(opts.ResultTimeout); time.Now().Before(deadline); {
		if time.Since(lastRefresh) >= opts.RefreshEvery {
			var ok bool
			_ = chromedp.Run(ctx, chromedp.Evaluate(refreshJS, &ok))
			lastRefresh = time.Now()
			if !sleepOrStop(stop, 800*time.Millisecond) {
				return fail("", "Stopped by user while waiting for the result.", errors.New("stopped"))
			}
			// The dashboard may re-show the popup after a refresh; close it again.
			var f bool
			_ = chromedp.Run(ctx, chromedp.Evaluate(confirmJS, &f))
		}
		var rows []historyRow
		_ = chromedp.Run(ctx, chromedp.Evaluate(matchJS, &rows))
		seen := map[string]int{}
		var best *historyRow
		for i := range rows {
			r := rows[i]
			seen[r.Key]++
			if seen[r.Key] <= baseline[r.Key] {
				continue // existed before this upload
			}
			if best == nil || r.Date > best.Date {
				best = &rows[i]
			}
		}
		if best != nil {
			actual := strings.TrimSpace(best.Status)
			if actual == "" {
				actual = "Upload Completed"
			}
			oc := outcome{Actual: actual, Success: best.Success, Reject: best.Reject, RowKey: best.Key}
			if strings.Contains(strings.ToLower(actual), "fail") || best.Reject > 0 {
				oc.ErrorText = best.Text
			}
			return oc
		}
		if !sleepOrStop(stop, time.Second) {
			return fail("", "Stopped by user while waiting for the result.", errors.New("stopped"))
		}
	}
	return fail("Result Not Found", fmt.Sprintf("The upload was accepted, but no new completed row for file %s / process %s appeared within %s (Refresh was clicked every %s). Existing rows were ignored.", filepath.Base(tc.FilePath), tc.FileType, opts.ResultTimeout, opts.RefreshEvery), errors.New("new upload result row timeout"))
}

// ---------------------------------------------------------------------------
// Rejection dialog
// ---------------------------------------------------------------------------

// readRejectionReasons clicks Rejection on this run's history row (matched by
// its exact text), reads the ERROR_REASON cells, optionally screenshots the
// dialog, and closes it.
func (a *AutoUploaderApp) readRejectionReasons(ctx context.Context, rowKey string, capture bool, testID string) (string, string) {
	clickJS := `(() => {
	  const norm=s=>(s||'').replace(/\s+/g,' ').trim().toLowerCase();
	  const want=` + jsStr(rowKey) + `;
	  for(const tr of Array.from(document.querySelectorAll('table tbody tr'))){
	    if(norm(tr.innerText)!==want)continue;
	    const b=Array.from(tr.querySelectorAll('button,[role=button],a,input[type=button]')).find(x=>norm(x.innerText||x.value)==='rejection');
	    if(b){b.click();return true;}
	  }
	  return false;})()`
	var clicked bool
	if err := chromedp.Run(ctx, chromedp.Evaluate(clickJS, &clicked)); err != nil || !clicked {
		return "", ""
	}
	readJS := `(() => {` + jsVisible + `
	  const norm=s=>(s||'').trim();
	  const roots=Array.from(document.querySelectorAll('[role=dialog],.modal,.modal-dialog,.ov,body>div')).filter(visible).reverse();
	  for(const root of roots){
	    if(!/error view/i.test(root.innerText||''))continue;
	    for(const table of root.querySelectorAll('table')){
	      const hs=Array.from(table.querySelectorAll('thead th,thead td,[role=columnheader]')).map(e=>norm(e.innerText).toLowerCase());
	      const ix=hs.findIndex(h=>h.includes('error_reason')||h.includes('error reason'));
	      const out=[];
	      for(const tr of table.querySelectorAll('tbody tr')){
	        const cells=Array.from(tr.querySelectorAll('td')).map(e=>norm(e.innerText));
	        if(cells.length<1)continue;
	        const v=ix>=0?cells[ix]:cells[cells.length-1];
	        if(v)out.push(v);
	      }
	      if(out.length)return out;
	    }
	  }
	  return [];})()`
	var reasons []string
	for deadline := time.Now().Add(6 * time.Second); time.Now().Before(deadline); {
		_ = chromedp.Run(ctx, chromedp.Evaluate(readJS, &reasons))
		if len(reasons) > 0 {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	shot := ""
	if capture {
		if p, err := a.captureScreenshot(ctx, testID+"_REJECTION_DIALOG"); err == nil {
			shot = p
		}
	}
	closeJS := `(() => {` + jsVisible + `
	  const title=Array.from(document.querySelectorAll('body *')).find(e=>e.children.length===0&&(e.innerText||'').trim().toLowerCase()==='error view'&&visible(e));
	  const isClose=b=>visible(b)&&((b.getAttribute('aria-label')||'').toLowerCase().includes('close')||['×','x','close'].includes((b.innerText||'').trim().toLowerCase()));
	  if(title){let p=title;for(let i=0;i<6&&p;i++,p=p.parentElement){const b=Array.from(p.querySelectorAll('button,[role=button],a')).find(isClose);if(b){b.click();return true;}}}
	  const x=Array.from(document.querySelectorAll('button,[role=button],a')).find(isClose);if(x){x.click();return true;}return false;})()`
	var closed bool
	_ = chromedp.Run(ctx, chromedp.Evaluate(closeJS, &closed))
	time.Sleep(250 * time.Millisecond)

	parts := make([]string, 0, len(reasons))
	for _, r := range reasons {
		r = strings.TrimSpace(r)
		if lr := strings.ToLower(r); r != "" && lr != "error_reason" && lr != "id" {
			parts = append(parts, r)
		}
	}
	return strings.Join(parts, " | "), shot
}

// ---------------------------------------------------------------------------
// Toolbar dropdowns
// ---------------------------------------------------------------------------

// selectUploaderDropdown selects an option in one of the three toolbar
// dropdowns (0=exchange, 1=segment, 2=file type), located relative to the
// visible Choose File button so unrelated dropdowns (Select View, Setup options,
// Filter by Status) are never touched.
func selectUploaderDropdown(ctx context.Context, index int, wanted string) error {
	wanted = strings.TrimSpace(wanted)
	wantedJSON := jsStr(wanted)
	locateJS := fmt.Sprintf(`(() => {
	  const wantedIndex=%d;
	  const norm=s=>(s||'').replace(/\s+/g,' ').trim().toLowerCase();
	  %s
	  const choose=Array.from(document.querySelectorAll('button,[role=button],label,a')).find(e=>visible(e)&&norm(e.innerText||e.textContent).includes('choose file'));
	  if(!choose)return {ok:false,reason:'Cannot locate the visible Choose File button to identify the uploader dropdown row.'};
	  const ar=choose.getBoundingClientRect();
	  const inRow=e=>{if(!visible(e)||e.disabled)return false;const r=e.getBoundingClientRect();return Math.abs((r.top+r.height/2)-(ar.top+ar.height/2))<32&&r.left<ar.left&&r.right>0;};
	  const selects=Array.from(document.querySelectorAll('select')).filter(inRow).sort((a,b)=>a.getBoundingClientRect().left-b.getBoundingClientRect().left);
	  if(selects.length<=wantedIndex)return {ok:false,reason:'Toolbar dropdown #'+(wantedIndex+1)+' not found; detected '+selects.length+' dropdown(s) on the Choose File row.'};
	  document.querySelectorAll('[data-boqa-select-target]').forEach(e=>e.removeAttribute('data-boqa-select-target'));
	  const target=selects[wantedIndex];
	  const opt=Array.from(target.options).find(o=>norm(o.textContent)===norm(%s)||norm(o.value)===norm(%s));
	  target.setAttribute('data-boqa-select-target','yes');
	  return {ok:true,selector:'select[data-boqa-select-target="yes"]',current:norm(target.options[target.selectedIndex]?.textContent),hasOption:!!opt,options:Array.from(target.options).map(o=>norm(o.textContent)).slice(0,80).join('|')};
	})()`, index, jsVisible, wantedJSON, wantedJSON)
	var target struct {
		OK        bool   `json:"ok"`
		Selector  string `json:"selector"`
		Current   string `json:"current"`
		HasOption bool   `json:"hasOption"`
		Options   string `json:"options"`
		Reason    string `json:"reason"`
	}
	if err := chromedp.Run(ctx, chromedp.Evaluate(locateJS, &target)); err != nil {
		return fmt.Errorf("inspect toolbar dropdown %d: %w", index+1, err)
	}
	if !target.OK {
		return errors.New(target.Reason)
	}
	// Options are loaded asynchronously by the app; wait for the wanted one.
	for attempt := 0; attempt < 16 && !target.HasOption; attempt++ {
		time.Sleep(500 * time.Millisecond)
		if err := chromedp.Run(ctx, chromedp.Evaluate(locateJS, &target)); err != nil {
			return fmt.Errorf("reinspect toolbar dropdown %d: %w", index+1, err)
		}
		if !target.OK {
			return errors.New(target.Reason)
		}
	}
	if !target.HasOption {
		return fmt.Errorf("option %q is not available in toolbar dropdown %d after waiting for options to load; options currently shown: %s", wanted, index+1, target.Options)
	}
	if strings.EqualFold(target.Current, wanted) {
		if stable, _ := toolbarSelectionStable(ctx, index, wanted); stable {
			return nil
		}
	}
	// Real keyboard input first, so the framework's change handlers fire.
	keyboardErr := chromedp.Run(ctx,
		chromedp.Focus(target.Selector),
		chromedp.Click(target.Selector),
		chromedp.SendKeys(target.Selector, wanted),
		chromedp.KeyEvent(kb.Enter),
	)
	if keyboardErr == nil {
		if stable, err := toolbarSelectionStable(ctx, index, wanted); err == nil && stable {
			return nil
		}
	}
	// Fallback: set the option and fire input+change, then require it to persist.
	fallbackJS := fmt.Sprintf(`(() => {
	  const norm=s=>(s||'').replace(/\s+/g,' ').trim().toLowerCase();
	  const wanted=norm(%s);
	  const t=document.querySelector('[data-boqa-select-target="yes"]');
	  if(!t)return {ok:false,reason:'Toolbar dropdown disappeared while selecting'};
	  const opt=Array.from(t.options).find(o=>norm(o.textContent)===wanted||norm(o.value)===wanted);
	  if(!opt)return {ok:false,reason:'Option '+wanted+' not available; options='+Array.from(t.options).map(o=>norm(o.textContent)).join('|')};
	  t.focus();t.selectedIndex=Array.from(t.options).indexOf(opt);
	  t.dispatchEvent(new Event('input',{bubbles:true,composed:true}));
	  t.dispatchEvent(new Event('change',{bubbles:true,composed:true}));
	  t.blur();return {ok:true};})()`, wantedJSON)
	var fb struct {
		OK     bool   `json:"ok"`
		Reason string `json:"reason"`
	}
	if err := chromedp.Run(ctx, chromedp.Evaluate(fallbackJS, &fb)); err != nil {
		return fmt.Errorf("select %q in toolbar dropdown %d: keyboard interaction failed (%v); fallback failed: %w", wanted, index+1, keyboardErr, err)
	}
	if !fb.OK {
		return fmt.Errorf("select %q in toolbar dropdown %d: %s", wanted, index+1, fb.Reason)
	}
	stable, err := toolbarSelectionStable(ctx, index, wanted)
	if err != nil {
		return err
	}
	if !stable {
		return fmt.Errorf("selection of %q in toolbar dropdown %d did not persist after the uploader updated (framework-state mismatch, not a missing option)", wanted, index+1)
	}
	return nil
}

func toolbarSelectionStable(ctx context.Context, index int, wanted string) (bool, error) {
	js := fmt.Sprintf(`(() => {
	  const norm=s=>(s||'').replace(/\s+/g,' ').trim().toLowerCase();
	  %s
	  const wanted=norm(%s);
	  const choose=Array.from(document.querySelectorAll('button,[role=button],label,a')).find(e=>visible(e)&&norm(e.innerText||e.textContent).includes('choose file'));
	  if(!choose)return false;
	  const ar=choose.getBoundingClientRect();
	  const ss=Array.from(document.querySelectorAll('select')).filter(e=>visible(e)&&!e.disabled&&Math.abs((e.getBoundingClientRect().top+e.getBoundingClientRect().height/2)-(ar.top+ar.height/2))<32&&e.getBoundingClientRect().left<ar.left).sort((a,b)=>a.getBoundingClientRect().left-b.getBoundingClientRect().left);
	  const t=ss[%d];return !!t&&norm(t.options[t.selectedIndex]?.textContent)===wanted;})()`, jsVisible, jsStr(strings.TrimSpace(wanted)), index)
	for i := 0; i < 8; i++ {
		var stable bool
		if err := chromedp.Run(ctx, chromedp.Evaluate(js, &stable)); err != nil {
			return false, fmt.Errorf("verify toolbar dropdown %d selection: %w", index+1, err)
		}
		if stable {
			return true, nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return false, nil
}

// ---------------------------------------------------------------------------
// Screenshots
// ---------------------------------------------------------------------------

func (a *AutoUploaderApp) captureScreenshot(ctx context.Context, testID string) (string, error) {
	dir := filepath.Join(a.reportsDir, "screenshots")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	var png []byte
	if err := chromedp.Run(ctx, chromedp.FullScreenshot(&png, 90)); err != nil {
		return "", err
	}
	if len(png) == 0 {
		return "", errors.New("browser returned an empty screenshot")
	}
	clean := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, testID)
	if clean == "" {
		clean = "TEST"
	}
	name := fmt.Sprintf("%s_%s.png", clean, time.Now().Format("20060102_150405_000"))
	if err := os.WriteFile(filepath.Join(dir, name), png, 0o644); err != nil {
		return "", err
	}
	return "screenshots/" + name, nil
}

var csvHeader = []string{"Test ID", "Segment", "File Type", "File Name", "Expected", "Actual", "Success Count", "Reject Count", "Rejection Reason", "Status", "Error Details", "Screenshot", "Rejection Dialog Screenshot", "Final Uploader Screenshot", "Started At", "Finished At"}

func (a *AutoUploaderApp) snapshot() ([]TestResult, string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]TestResult(nil), a.results...), a.finalScreenshot
}

// csvCell neutralises spreadsheet formula injection: values read from the app
// (file names, rejection reasons) are untrusted and must not run as formulas
// when the report is opened in Excel.
func csvCell(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

func (a *AutoUploaderApp) csvBytes() ([]byte, error) {
	rows, finalShot := a.snapshot()
	var out bytes.Buffer
	out.WriteString("\xef\xbb\xbf") // UTF-8 BOM so Excel opens it correctly
	cw := csv.NewWriter(&out)
	_ = cw.Write(csvHeader)
	for _, r := range rows {
		_ = cw.Write([]string{csvCell(r.ID), csvCell(r.Segment), csvCell(r.FileType), csvCell(r.FileName), r.Expected, csvCell(r.Actual), itoa(r.Success), itoa(r.Reject), csvCell(r.RejectionReason), r.Status, csvCell(r.ErrorText), r.ScreenshotPath, r.RejectionScreenshotPath, finalShot, r.StartedAt, r.FinishedAt})
	}
	cw.Flush()
	return out.Bytes(), cw.Error()
}

var reportTmpl = template.Must(template.New("r").Funcs(template.FuncMap{
	"dur": func(s, e string) string {
		a, err1 := time.Parse(time.RFC3339, s)
		b, err2 := time.Parse(time.RFC3339, e)
		if err1 != nil || err2 != nil {
			return "—"
		}
		d := b.Sub(a).Round(time.Second)
		return d.String()
	},
}).Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><title>BO Uploader QA Report</title>
<style>body{font:14px/1.45 "Segoe UI",Arial,sans-serif;margin:28px;color:#17352b}h1{margin:0 0 4px}table{border-collapse:collapse;width:100%;margin-top:16px}th,td{border:1px solid #dce9e0;padding:7px 10px;text-align:left;vertical-align:top;font-size:13px}th{background:#f4faf6}
.b{display:inline-block;padding:2px 9px;border-radius:99px;font-weight:700;font-size:11px}.PASS{background:#e3f7ea;color:#147749}.FAIL{background:#fff0ef;color:#b82d35}.ERROR{background:#fff3df;color:#9a5d00}.SKIPPED{background:#eee;color:#555}
.sum span{display:inline-block;margin-right:18px;font-weight:700}small{color:#71847b}</style>
<h1>BO Uploader QA Report</h1><small>Generated {{.At}}</small>
<p class="sum"><span>Total: {{.Total}}</span><span style="color:#147749">Passed: {{.Pass}}</span><span style="color:#b82d35">Failed: {{.Fail}}</span><span style="color:#9a5d00">Errors: {{.Err}}</span><span>Skipped: {{.Skip}}</span></p>
<table><thead><tr><th>Test</th><th>Segment / File type</th><th>File</th><th>Expected</th><th>Actual</th><th>Success</th><th>Reject</th><th>Rejection reason</th><th>Result</th><th>Time</th><th>Details</th><th>Evidence</th></tr></thead><tbody>
{{range .Rows}}<tr><td><b>{{.ID}}</b></td><td>{{.Segment}}<br><small>{{.FileType}}</small></td><td>{{.FileName}}</td><td>{{.Expected}}</td><td>{{.Actual}}</td><td>{{.Success}}</td><td>{{.Reject}}</td><td>{{.RejectionReason}}</td><td><span class="b {{.Status}}">{{.Status}}</span></td><td>{{dur .StartedAt .FinishedAt}}</td><td>{{.ErrorText}}</td>
<td>{{if .ScreenshotPath}}<a href="{{.ScreenshotPath}}"><img src="{{.ScreenshotPath}}" width="150" alt="screenshot"></a>{{end}}{{if .RejectionScreenshotPath}}<br><a href="{{.RejectionScreenshotPath}}">Rejection dialog</a>{{end}}</td></tr>
{{else}}<tr><td colspan="12">No results.</td></tr>{{end}}</tbody></table>
{{if .Final}}<p><b>Final uploader screenshot:</b> <a href="{{.Final}}">open</a></p>{{end}}</html>`))

func (a *AutoUploaderApp) htmlBytes() ([]byte, error) {
	rows, final := a.snapshot()
	d := struct {
		At                           string
		Rows                         []TestResult
		Total, Pass, Fail, Err, Skip int
		Final                        string
	}{At: time.Now().Format("2006-01-02 15:04:05"), Rows: rows, Total: len(rows), Final: final}
	for _, r := range rows {
		switch r.Status {
		case "PASS":
			d.Pass++
		case "FAIL":
			d.Fail++
		case "ERROR":
			d.Err++
		case "SKIPPED":
			d.Skip++
		}
	}
	var buf bytes.Buffer
	if err := reportTmpl.Execute(&buf, d); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// writeReports persists the CSV and HTML report next to the screenshots.
func (a *AutoUploaderApp) writeReports() error {
	if err := os.MkdirAll(a.reportsDir, 0o755); err != nil {
		return err
	}
	b, err := a.csvBytes()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(a.reportsDir, "bo_uploader_qa_report.csv"), b, 0o644); err != nil {
		return err
	}
	h, err := a.htmlBytes()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(a.reportsDir, "bo_uploader_qa_report.html"), h, 0o644)
}

func (a *AutoUploaderApp) reportCSV(w http.ResponseWriter, r *http.Request) {
	b, err := a.csvBytes()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=bo_uploader_qa_report.csv")
	_, _ = w.Write(b)
}

func (a *AutoUploaderApp) reportHTML(w http.ResponseWriter, r *http.Request) {
	b, err := a.htmlBytes()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	// Rewrite relative evidence links so they resolve under /reports/.
	b = bytes.ReplaceAll(b, []byte(`"screenshots/`), []byte(`"/reports/screenshots/`))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(b)
}

func (a *AutoUploaderApp) testCasesCSV(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=bo_uploader_testcases_template.csv")
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"TestID", "Segment", "FileType", "FilePath", "Expected", "ExpectedSuccess", "ExpectedReject", "ReasonContains", "Exchange"})
	_ = cw.Write([]string{"TC001", "HOLDING", "BO_INC_DP_HOLDING", `C:\BO-TestFiles\VALID_HOLDING.txt`, "Success", "3", "0", "", "BO"})
	_ = cw.Write([]string{"TC002", "HOLDING", "BO_INC_DP_HOLDING", `C:\BO-TestFiles\INVALID_HOLDING.txt`, "Reject", "", "2", "invalid", "BO"})
	cw.Flush()
}

// ============================================================
// MAIN
// ============================================================

func main() {
	// Auto File Uploader QA — additive routes; existing RMS handlers are retained.
	qaBrowser := strings.TrimRight(strings.TrimSpace(os.Getenv("BOQA_CHROME_DEBUG_URL")), "/")
	if qaBrowser == "" {
		qaBrowser = "http://127.0.0.1:9222"
	}
	qaReports := strings.TrimSpace(os.Getenv("BOQA_REPORTS_DIR"))
	if qaReports == "" {
		qaReports = "/home/rsapp/Application/Web_host/KSH/auto_uploader_reports"
	}
	qaApp := &AutoUploaderApp{browser: qaBrowser, reportsDir: qaReports, defaultURL: ""}
	if err := os.MkdirAll(filepath.Join(qaReports, "screenshots"), 0755); err != nil {
		log.Printf("Auto uploader: cannot create reports folder: %v", err)
	}
	http.HandleFunc("/api/auto-uploader/status", qaApp.status)
	http.HandleFunc("/api/auto-uploader/version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"version": "rms-api-auto-uploader-2026.10.27"})
	})
	http.HandleFunc("/api/auto-uploader/inspect", qaApp.inspect)
	http.HandleFunc("/api/auto-uploader/testcases.csv", qaApp.testCasesCSV)
	http.HandleFunc("/api/auto-uploader/run", qaApp.run)
	http.HandleFunc("/api/auto-uploader/stop", qaApp.stopRun)
	http.HandleFunc("/api/auto-uploader/results", qaApp.getResults)
	http.HandleFunc("/api/auto-uploader/clear-results", qaApp.clearResults)
	http.HandleFunc("/api/auto-uploader/report.csv", qaApp.reportCSV)
	http.HandleFunc("/api/auto-uploader/report.html", qaApp.reportHTML)
	http.HandleFunc("/api/auto-uploader/video", qaApp.uploadVideo)
	http.Handle("/api/auto-uploader/reports/", http.StripPrefix("/api/auto-uploader/reports/", http.FileServer(http.Dir(qaReports))))

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
