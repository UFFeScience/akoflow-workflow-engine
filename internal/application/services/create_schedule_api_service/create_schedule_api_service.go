package create_schedule_api_service

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"runtime"
	"strings"

	"github.com/UFFeScience/akoflow/internal/api/requests"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/schedule_repository"
)

type CreateScheduleApiService struct {
	scheduleRepository schedule_repository.IScheduleRepository
	versionOutput      func() ([]byte, error)
	statFile           func(string) (os.FileInfo, error)
	writeFile          func(string, []byte, os.FileMode) error
	buildPlugin        func(string, string) error
	pluginOpener       func(string) (PluginLookup, error)
	validateCode       func(string) (bool, string)
}

type PluginLookup interface {
	Lookup(string) (plugin.Symbol, error)
}

func New() *CreateScheduleApiService {
	service := &CreateScheduleApiService{
		scheduleRepository: config.App().Repository.ScheduleRepository,
		versionOutput:      func() ([]byte, error) { return exec.Command("go", "version").Output() },
		statFile:           os.Stat, writeFile: os.WriteFile,
		pluginOpener: func(path string) (PluginLookup, error) { return plugin.Open(path) },
	}
	service.buildPlugin = func(goFile, soFile string) error {
		cmd := exec.Command("go", "build", "-gcflags=all=-N -l", "-buildmode=plugin", "-o", soFile, goFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	service.validateCode = service.ValidateUserCode
	return service
}

func NewWithDependencies(repository schedule_repository.IScheduleRepository, version func() ([]byte, error), stat func(string) (os.FileInfo, error), write func(string, []byte, os.FileMode) error, build func(string, string) error, opener func(string) (PluginLookup, error)) *CreateScheduleApiService {
	service := &CreateScheduleApiService{scheduleRepository: repository, versionOutput: version, statFile: stat, writeFile: write, buildPlugin: build, pluginOpener: opener}
	service.validateCode = service.ValidateUserCode
	return service
}

func (h *CreateScheduleApiService) ValidateUserCode(userCode string) (bool, string) {
	hash := sha256.Sum256([]byte(userCode))
	hashStr := hex.EncodeToString(hash[:])
	baseName := "ako_plugin_" + hashStr

	goFile := baseName + ".go"
	soFile := baseName + ".so"

	runtimeVersion := runtime.Version()

	output, err := h.versionOutput()
	if err != nil {
		return false, ""
	}
	buildVersion := string(output)
	buildVersionTrimmed := strings.TrimSpace(buildVersion)

	if !strings.Contains(buildVersionTrimmed, runtimeVersion) {
		return false, ""
	}

	if _, err := h.statFile(soFile); os.IsNotExist(err) {
		if !h.compilePlugin(goFile, soFile, userCode) {
			return false, soFile
		}
	} else {
		fmt.Println("Usando plugin já compilado:", soFile)
	}

	return h.executePlugin(soFile), soFile
}

func (h *CreateScheduleApiService) compilePlugin(goFile, soFile, userCode string) bool {
	if err := h.writeFile(goFile, []byte(userCode), 0644); err != nil {
		return false
	}

	fmt.Println("Compilando plugin:", soFile)

	if err := h.buildPlugin(goFile, soFile); err != nil {
		fmt.Println("Erro ao compilar plugin:", err)
		return false
	}

	fmt.Println("Plugin compilado com sucesso:", soFile)

	return true
}

func (h *CreateScheduleApiService) executePlugin(soFile string) bool {
	p, err := h.pluginOpener(filepath.Clean(soFile))

	if err != nil {
		fmt.Println("Erro ao abrir plugin:", err)
		return false
	}

	sym, err := p.Lookup("AkoScore")
	if err != nil {
		fmt.Println("Erro ao procurar símbolo 'AkoScore':", err)
		return false
	}

	akoScoreFunc, ok := sym.(func(any) float64)

	if !ok {
		fmt.Println("Símbolo 'AkoScore' não é uma função válida")
		return false
	}

	input := map[string]any{
		"time_estimate":   2.5,
		"memory_required": 1024.0,
		"vcpus_required":  2.0,
		"memory_free":     2048.0,
		"memory_max":      4096.0,
		"vcpus_available": 4.0,
		"alpha":           0.6,
		"activity_name":   "activity1",
		"machine_type":    "c7i.large",
	}

	result := akoScoreFunc(input)

	fmt.Println("Plugin executado com sucesso:", result)
	return true
}

func (h *CreateScheduleApiService) CreateSchedule(name string, scheduleType string, code string) (types_api.ApiScheduleType, error) {

	codeDecoded, err := base64.StdEncoding.DecodeString(code)
	if err != nil {
		return types_api.ApiScheduleType{}, fmt.Errorf("invalid base64 code: %v", err)
	}

	isValid, soFile := h.validateCode(string(codeDecoded))
	if !isValid {
		return types_api.ApiScheduleType{}, fmt.Errorf("invalid user code")
	}

	scheduleEngine, err := h.scheduleRepository.CreateSchedule(name, scheduleType, code, soFile)
	if err != nil {
		return types_api.ApiScheduleType{}, err
	}

	return types_api.ApiScheduleType{
		ID:        scheduleEngine.ID,
		Type:      scheduleEngine.Type,
		Code:      scheduleEngine.Code,
		Name:      scheduleEngine.Name,
		CreatedAt: scheduleEngine.CreatedAt,
		UpdatedAt: scheduleEngine.UpdatedAt,
	}, nil
}
