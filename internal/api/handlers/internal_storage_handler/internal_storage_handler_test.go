package internal_storage_handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	file_list_parser_service "github.com/UFFeScience/akoflow/internal/application/services/file_disk_parser_service"
	"github.com/UFFeScience/akoflow/internal/application/services/file_spec_parser_service"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/storages_repository"
)

type storageFake struct {
	storages_repository.IStorageRepository
	found                              storages_repository.StorageDatabase
	findErr                            error
	initial, end, initialSpec, endSpec string
	id                                 int
}

func (f *storageFake) UpdateInitialFileListDisk(id int, value string) error {
	f.id, f.initial = id, value
	return nil
}
func (f *storageFake) UpdateEndFileListDisk(id int, value string) error {
	f.id, f.end = id, value
	return nil
}
func (f *storageFake) UpdateInitialDiskSpec(id int, value string) error {
	f.id, f.initialSpec = id, value
	return nil
}
func (f *storageFake) UpdateEndDiskSpec(id int, value string) error {
	f.id, f.endSpec = id, value
	return nil
}
func (f *storageFake) Find(int) (storages_repository.StorageDatabase, error) {
	return f.found, f.findErr
}

func newHandler(fake *storageFake) *InternalStorageHandler {
	return &InternalStorageHandler{fileSpecParserService: file_spec_parser_service.New(), fileListParserService: file_list_parser_service.New(), storageRepository: fake}
}

func TestStorageUpdateHandlers(t *testing.T) {
	fake := &storageFake{}
	handler := newHandler(fake)
	fileList := "./:\n-rw-r--r-- 1 owner group 10 Jan 01 00:00 file.txt"
	disk := "Filesystem Size Used Avail Use% Mounted on\n/dev/sda 10G 1G 9G 10% /data"
	cases := []struct {
		call  func(http.ResponseWriter, *http.Request)
		body  string
		value func() string
	}{
		{handler.InitialFileListHandler, fileList, func() string { return fake.initial }}, {handler.EndFileListHandler, fileList, func() string { return fake.end }},
		{handler.InitialDiskSpecHandler, disk, func() string { return fake.initialSpec }}, {handler.EndDiskSpecHandler, disk, func() string { return fake.endSpec }},
	}
	for _, item := range cases {
		recorder := httptest.NewRecorder()
		item.call(recorder, httptest.NewRequest(http.MethodPost, "/?activityId=12", strings.NewReader(item.body)))
		if recorder.Body.String() != "ok" || fake.id != 12 || item.value() == "" {
			t.Fatal("update handler failed")
		}
	}
}

func TestGetInitialFileList(t *testing.T) {
	fake := &storageFake{found: storages_repository.StorageDatabase{InitialFileList: "i", EndFileList: "e", InitialDiskSpec: "is", EndDiskSpec: "es"}}
	handler := newHandler(fake)
	recorder := httptest.NewRecorder()
	handler.GetInitialFileListHandler(recorder, httptest.NewRequest(http.MethodGet, "/?activityId=1", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"initial_file_list":"i"`) {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
	fake.findErr = errors.New("db")
	recorder = httptest.NewRecorder()
	handler.GetInitialFileListHandler(recorder, httptest.NewRequest(http.MethodGet, "/?activityId=1", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatal("find error must return 500")
	}
}
