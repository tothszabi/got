package got_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/bitrise-io/got"
)

var (
	httpt      = NewHttptestServer()
	okFileStat os.FileInfo
)

func init() {

	var err error

	okFileStat, err = os.Stat("go.mod")

	if err != nil {
		panic(err)
	}
}

func TestGetInfoAndInit(t *testing.T) {

	t.Run("getInfoTest", getInfoTest)
	t.Run("okInitTest", okInitTest)
	t.Run("errInitTest", errInitTest)
	t.Run("sendHeadersTest", sendHeadersTest)
}

func TestDownloading(t *testing.T) {

	t.Run("downloadOkFileTest", downloadOkFileTest)
	t.Run("downloadOneByteFile", downloadOneByteFile)
	t.Run("downloadTwoByteFile", downloadTwoByteFile)
	t.Run("downloadThreeByteFile", downloadThreeByteFile)
	t.Run("downloadNotFoundTest", downloadNotFoundTest)
	t.Run("downloadOkFileContentTest", downloadOkFileContentTest)
	t.Run("downloadTimeoutContextTest", downloadTimeoutContextTest)
	t.Run("downloadHeadNotSupported", downloadHeadNotSupported)
	t.Run("downloadPartialContentNotSupportedTest", downloadPartialContentNotSupportedTest)
	t.Run("getFilenameTest", getFilenameTest)
	t.Run("coverTests", coverTests)
}

func getInfoTest(t *testing.T) {

	tmpFile := createTemp()
	defer clean(tmpFile)

	dl := got.NewDownload(context.Background(), httpt.URL+"/ok_file", tmpFile)

	info, err := dl.GetInfoOrDownload()

	if err != nil {
		t.Error(err)
		return
	}

	if info.Rangeable == false {
		t.Error("rangeable should be true")
	}

	if info.Size != uint64(okFileStat.Size()) {
		t.Errorf("Invalid file size, wants %d but got %d", okFileStat.Size(), info.Size)
	}
}

func sendHeadersTest(t *testing.T) {

	tmpFile := createTemp()
	defer clean(tmpFile)

	dl := got.NewDownload(context.Background(), httpt.URL+"/header_values", tmpFile)
	dl.Header = []got.GotHeader{
		{
			Key:   "x-test-header",
			Value: "foobar",
		},
	}

	info, err := dl.GetInfoOrDownload()

	if err != nil {
		t.Error(err)
		return
	}

	if info.Rangeable == false {
		t.Error("rangeable should be true")
	}

	if info.Size != uint64(okFileStat.Size()) {
		t.Errorf("Invalid file size, wants %d but got %d", okFileStat.Size(), info.Size)
	}
}

func getFilenameTest(t *testing.T) {

	tmpDir := os.TempDir()
	defer os.RemoveAll(tmpDir)

	dl := got.NewDownload(context.Background(), httpt.URL+"/file_name", "")
	dl.Dir = tmpDir

	_, err := dl.GetInfoOrDownload()

	if err != nil {

		t.Errorf("Unexpected error: " + err.Error())
	}

	if dl.Path() != filepath.Join(tmpDir, "go.mod") {
		t.Errorf("Expecting file name to be: go.mod but got: " + filepath.Join(tmpDir, "go.mod"))
	}

}

func okInitTest(t *testing.T) {

	tmpFile := createTemp()
	defer clean(tmpFile)

	dl := &got.Download{
		URL:  httpt.URL + "/ok_file",
		Dest: tmpFile,
	}

	if err := dl.Init(); err != nil {
		t.Error(err)
	}
}

func errInitTest(t *testing.T) {

	tmpFile := createTemp()
	defer clean(tmpFile)

	dl := &got.Download{
		URL:  httpt.URL + "/not_found",
		Dest: tmpFile,
	}

	if err := dl.Init(); err == nil {
		t.Error("Expecting error but got nil")
	}
}

func downloadOkFileTest(t *testing.T) {

	tmpFile := createTemp()
	defer clean(tmpFile)

	dl := &got.Download{
		URL:  httpt.URL + "/ok_file",
		Dest: tmpFile,
	}

	// Init
	if err := dl.Init(); err != nil {
		t.Error(err)
		return
	}

	// Check size
	if dl.TotalSize() != uint64(okFileStat.Size()) {
		t.Errorf("Invalid file size, wants %d but got %d", okFileStat.Size(), dl.TotalSize())
	}

	// Start download
	if err := dl.Start(); err != nil {
		t.Error(err)
	}

	stat, err := os.Stat(tmpFile)

	if err != nil {
		t.Error(err)
	}

	if okFileStat.Size() != stat.Size() {
		t.Errorf("Expecting size: %d, but got %d", okFileStat.Size(), stat.Size())
	}
}

func downloadNotFoundTest(t *testing.T) {

	tmpFile := createTemp()
	defer clean(tmpFile)

	dl := &got.Download{
		URL:  httpt.URL + "/not_found",
		Dest: tmpFile,
	}

	err := dl.Init()

	if err == nil {
		t.Error("It should have an error")
		return
	}
}

func downloadOkFileContentTest(t *testing.T) {

	tmpFile := createTemp()
	defer clean(tmpFile)

	d := &got.Download{
		URL:       httpt.URL + "/ok_file_with_range_delay",
		Dest:      tmpFile,
		ChunkSize: 10,
	}

	if err := d.Init(); err != nil {
		t.Error(err)
		return
	}

	if err := d.Start(); err != nil {
		t.Error(err)
		return
	}

	mod, err := ioutil.ReadFile("go.mod")

	if err != nil {
		t.Error(err)
		return
	}

	dlFile, err := ioutil.ReadFile(tmpFile)

	if err != nil {
		t.Error(err)
		return
	}

	if string(mod) != string(dlFile) {

		fmt.Println("a", string(mod))
		fmt.Println("b", string(dlFile))
		t.Error("Corrupted file")
	}

}

func downloadTimeoutContextTest(t *testing.T) {

	tmpFile, _ := ioutil.TempDir("", "")
	defer clean(tmpFile)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	d := got.NewDownload(ctx, httpt.URL+"/ok_file_with_range_delay", tmpFile)
	d.ChunkSize = 2

	if err := d.Init(); err == nil {
		t.Error("Expecting context deadline")
	}

	if err := d.Start(); err == nil {
		t.Error("Expecting context deadline")
	}

	d = got.NewDownload(ctx, httpt.URL+"/ok_file_with_range_delay", tmpFile)

	// just to cover request error.
	g := got.NewWithContext(ctx)
	err := g.Download("invalid://ok_file_with_range_delay", tmpFile)

	if err == nil {
		t.Errorf("Expecting invalid scheme error")
	}
}

func downloadHeadNotSupported(t *testing.T) {

	tmpFile := createTemp()
	defer clean(tmpFile)

	d := &got.Download{
		URL:  httpt.URL + "/found_and_head_not_allowed",
		Dest: tmpFile,
	}

	// init
	if err := d.Init(); err != nil {
		t.Error(err)
		return
	}

	if d.TotalSize() != 0 {
		t.Error("Size should be 0")
	}

	if d.IsRangeable() != false {
		t.Error("rangeable should be false")
	}

	d = &got.Download{
		URL:  httpt.URL + "/found_and_head_not_allowed",
		Dest: "/invalid/path",
	}

	if err := d.Init(); err == nil {
		t.Error("Expecting invalid path error")
	}
}

func downloadPartialContentNotSupportedTest(t *testing.T) {

	tmpFile := createTemp()
	defer clean(tmpFile)

	d := &got.Download{
		URL:  httpt.URL + "/found_and_head_not_allowed",
		Dest: tmpFile,
	}

	if err := d.Init(); err != nil {
		t.Error(err)
		return
	}

	if d.TotalSize() != 0 {
		t.Errorf("Expect length to be 0, but got %d", d.TotalSize())
	}

	if err := d.Start(); err != nil {
		t.Error(err)
	}

	stat, err := os.Stat(tmpFile)

	if err != nil {
		t.Error(err)
	}

	if stat.Size() != 10 {
		t.Errorf("Invalid size: %d", stat.Size())
	}
}

func coverTests(t *testing.T) {

	// Just for testing
	destPath := createTemp()
	defer clean(destPath)

	// cover default dest path.
	// cover progress func and methods
	d := &got.Download{
		URL: httpt.URL + "/ok_file_with_range_delay",
	}

	// init
	if err := d.Init(); err != nil {
		t.Error(err)
	}

	if d.Path() != got.DefaultFileName {
		t.Errorf("Expecting name to be: %s but got: %s", got.DefaultFileName, d.Path())
	}

	go d.RunProgress(func(d *got.Download) {
		d.Size()
		d.Speed()
		d.AvgSpeed()
		d.TotalCost()
	})
}

func downloadOneByteFile(t *testing.T) {
	runSmallFileDownloadTest(t, "1_byte_file", "testdata/1-byte.txt")
}

func downloadTwoByteFile(t *testing.T) {
	runSmallFileDownloadTest(t, "2_byte_file", "testdata/2-byte.txt")
}

func downloadThreeByteFile(t *testing.T) {
	runSmallFileDownloadTest(t, "3_byte_file", "testdata/3-byte.txt")
}

func runSmallFileDownloadTest(t *testing.T, endpoint, path string) {
	tmpFile := createTemp()
	defer clean(tmpFile)

	dl := &got.Download{
		URL:  httpt.URL + "/" + endpoint,
		Dest: tmpFile,
	}

	// Init
	if err := dl.Init(); err != nil {
		t.Error(err)
		return
	}

	fileStat, err := os.Stat(path)
	if err != nil {
		t.Error(err)
		return
	}

	// Check size
	if dl.TotalSize() != uint64(fileStat.Size()) {
		t.Errorf("Invalid file size, wants %d but got %d", fileStat.Size(), dl.TotalSize())
	}

	// Start download
	if err := dl.Start(); err != nil {
		t.Error(err)
	}

	downloadedStat, err := os.Stat(tmpFile)

	if err != nil {
		t.Error(err)
		return
	}

	if fileStat.Size() != downloadedStat.Size() {
		t.Errorf("Expecting size: %d, but got %d", fileStat.Size(), downloadedStat.Size())
	}
}

func ExampleDownload() {

	// Just for testing
	destPath := createTemp()
	defer clean(destPath)

	ctx := context.Background()

	dl := got.NewDownload(ctx, testUrl, destPath)

	// Init
	if err := dl.Init(); err != nil {
		fmt.Println(err)
	}

	// Start download
	if err := dl.Start(); err != nil {
		fmt.Println(err)
	}

	fmt.Println("Done")

	// Output: Done
}

func createTemp() string {

	tmp, err := ioutil.TempFile("", "")

	if err != nil {
		panic(err)
	}

	defer tmp.Close()

	return tmp.Name()
}

func clean(tmpFile string) {

	os.Remove(tmpFile)
}
