package fileutils

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	biutils "github.com/jfrog/build-info-go/utils"
	"io"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/jfrog/build-info-go/entities"
	"github.com/jfrog/gofrog/crypto"
	gofrog "github.com/jfrog/gofrog/io"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
)

const (
	SymlinkFileContent          = ""
	File               ItemType = "file"
	Dir                ItemType = "dir"
	Any                ItemType = "any"
)

func GetFileSeparator() string {
	return string(os.PathSeparator)
}

// Check if path exists.
// If path points at a symlink and `preserveSymLink == true`,
// function will return `true` regardless of the symlink target
func IsPathExists(path string, preserveSymLink bool) bool {
	_, err := GetFileInfo(path, preserveSymLink)
	return !os.IsNotExist(err)
}

// Check if path points at a file.
// If path points at a symlink and `preserveSymLink == true`,
// function will return `true` regardless of the symlink target
func IsFileExists(path string, preserveSymLink bool) (bool, error) {
	fileInfo, err := GetFileInfo(path, preserveSymLink)
	if err != nil {
		if os.IsNotExist(err) { // If doesn't exist, don't omit an error
			return false, nil
		}
		return false, errorutils.CheckError(err)
	}
	return !fileInfo.IsDir(), nil
}

// Check if path points at a directory.
// If path points at a symlink and `preserveSymLink == true`,
// function will return `false` regardless of the symlink target
func IsDirExists(path string, preserveSymLink bool) (bool, error) {
	fileInfo, err := GetFileInfo(path, preserveSymLink)
	if err != nil {
		if os.IsNotExist(err) { // If doesn't exist, don't omit an error
			return false, nil
		}
		return false, errorutils.CheckError(err)
	}
	return fileInfo.IsDir(), nil
}

// Get the file info of the file in path.
// If path points at a symlink and `preserveSymLink == true`, return the file info of the symlink instead
func GetFileInfo(path string, preserveSymLink bool) (fileInfo os.FileInfo, err error) {
	if preserveSymLink {
		fileInfo, err = os.Lstat(path)
	} else {
		fileInfo, err = os.Stat(path)
	}
	// We should not do CheckError here, because the error is checked by the calling functions.
	return
}

func IsDirEmpty(path string) (isEmpty bool, err error) {
	dir, err := os.Open(path)
	if errorutils.CheckError(err) != nil {
		return
	}
	defer func() {
		err = errors.Join(err, errorutils.CheckError(dir.Close()))
	}()

	_, err = dir.Readdirnames(1)
	if err == io.EOF {
		isEmpty = true
		err = nil
		return
	}
	err = errorutils.CheckError(err)
	return
}

func IsPathSymlink(path string) bool {
	f, _ := os.Lstat(path)
	return f != nil && IsFileSymlink(f)
}

func IsFileSymlink(file os.FileInfo) bool {
	return file.Mode()&os.ModeSymlink != 0
}

// Return the file's name and dir of a given path by finding the index of the last separator in the path.
// Support separators : "/" , "\\" and "\\\\"
func GetFileAndDirFromPath(path string) (fileName, dir string) {
	index1 := strings.LastIndex(path, "/")
	index2 := strings.LastIndex(path, "\\")
	var index int
	offset := 0
	if index1 >= index2 {
		index = index1
	} else {
		index = index2
		// Check if the last separator is "\\\\" or "\\".
		index3 := strings.LastIndex(path, "\\\\")
		if index3 != -1 && index2-index3 == 1 {
			offset = 1
		}
	}
	if index != -1 {
		fileName = path[index+1:]
		// If the last separator is "\\\\" index will contain the index of the last "\\" ,
		// to get the dir path (without separator suffix) we will use the offset's value.
		dir = path[:index-offset]
		return
	}
	fileName = path
	dir = ""
	return
}

// Get the local path and filename from original file name and path according to targetPath
func GetLocalPathAndFile(originalFileName, relativePath, targetPath string, flat bool, placeholdersUsed bool) (localTargetPath, fileName string) {
	targetFileName, targetDirPath := GetFileAndDirFromPath(targetPath)
	// Remove double slashes and double backslashes that may appear in the path
	localTargetPath = filepath.Clean(targetDirPath)
	// When placeholders are used, the file path shouldn't be taken into account (or in other words, flat = true).
	if !flat && !placeholdersUsed {
		localTargetPath = filepath.Join(targetDirPath, relativePath)
	}

	fileName = originalFileName
	// '.' as a target path is equivalent to an empty target path.
	if targetFileName != "" && targetFileName != "." {
		fileName = targetFileName
	}
	return
}

// Return the recursive list of files and directories in the specified path
func ListFilesRecursiveWalkIntoDirSymlink(path string, walkIntoDirSymlink bool) (fileList []string, err error) {
	fileList = []string{}
	err = gofrog.Walk(path, func(path string, f os.FileInfo, err error) error {
		fileList = append(fileList, path)
		return nil
	}, walkIntoDirSymlink)
	err = errorutils.CheckError(err)
	return
}

// Return the recursive list of files and directories in the specified path
func ListFilesWithFilterFunc(rootPath string, isRecursive, walkIntoDirSymlink bool, filterFunc func(filePath string) (bool, error)) (fileList []string, err error) {
	fileList = []string{}
	err = gofrog.Walk(rootPath, func(path string, f os.FileInfo, err error) error {
		if err != nil || path == rootPath {
			return err
		}
		include, err := filterFunc(path)
		if err != nil {
			return err
		}
		if include {
			fileList = append(fileList, path)
		}
		if !isRecursive {
			// If the path is not in the root directory, and it's a directory we should skip it and not walk into it.
			isDir, err := IsDirExists(path, false)
			if err != nil {
				return err
			}
			if isDir {
				return gofrog.ErrSkipDir
			}
		}
		return nil
	}, walkIntoDirSymlink)
	err = errorutils.CheckError(err)
	return
}

// Return the list of files and directories in the specified path
func ListFiles(path string, includeDirs bool) ([]string, error) {
	sep := GetFileSeparator()
	if !strings.HasSuffix(path, sep) {
		path += sep
	}
	fileList := []string{}
	files, _ := os.ReadDir(path)
	path = strings.TrimPrefix(path, "."+sep)

	for _, f := range files {
		filePath := path + f.Name()
		exists, err := IsFileExists(filePath, false)
		if err != nil {
			return nil, err
		}
		if exists || IsPathSymlink(filePath) {
			fileList = append(fileList, filePath)
		} else if includeDirs {
			isDir, err := IsDirExists(filePath, false)
			if err != nil {
				return nil, err
			}
			if isDir {
				fileList = append(fileList, filePath)
			}
		}
	}
	return fileList, nil
}

func GetUploadRequestContent(file *os.File) io.Reader {
	if file == nil {
		return bytes.NewBuffer([]byte(SymlinkFileContent))
	}
	return bufio.NewReader(file)
}

func GetFileSize(file *os.File) (int64, error) {
	size := int64(0)
	if file != nil {
		fileInfo, err := file.Stat()
		if errorutils.CheckError(err) != nil {
			return size, err
		}
		size = fileInfo.Size()
	}
	return size, nil
}

func CreateFilePath(localPath, fileName string) (string, error) {
	if localPath != "" {
		err := os.MkdirAll(localPath, 0777)
		if errorutils.CheckError(err) != nil {
			return "", err
		}
		fileName = filepath.Join(localPath, fileName)
	}
	return fileName, nil
}

func CreateDirIfNotExist(path string) error {
	exist, err := IsDirExists(path, false)
	if exist || err != nil {
		return err
	}
	_, err = CreateFilePath(path, "")
	return err
}

// Reads the content of the file in the source path and appends it to
// the file in the destination path.
func AppendFile(srcPath string, destFile *os.File) (err error) {
	srcFile, err := os.Open(srcPath)
	if errorutils.CheckError(err) != nil {
		return
	}

	defer func() {
		err = errors.Join(err, errorutils.CheckError(srcFile.Close()))
	}()

	reader := bufio.NewReader(srcFile)

	writer := bufio.NewWriter(destFile)
	buf := make([]byte, 1024000)
	for {
		var n int
		n, err = reader.Read(buf)
		if err != io.EOF {
			err = errorutils.CheckError(err)
			if err != nil {
				return err
			}
		}
		if n == 0 {
			break
		}
		_, err = writer.Write(buf[:n])
		err = errorutils.CheckError(err)
		if err != nil {
			return err
		}
	}
	err = writer.Flush()
	return errorutils.CheckError(err)
}

func GetHomeDir() string {
	home := os.Getenv("HOME")
	if home != "" {
		return home
	}
	home = os.Getenv("USERPROFILE")
	if home != "" {
		return home
	}
	currentUser, err := user.Current()
	if err == nil {
		return currentUser.HomeDir
	}
	return ""
}

func IsSshUrl(urlPath string) bool {
	u, err := url.Parse(urlPath)
	if err != nil {
		return false
	}
	return strings.ToLower(u.Scheme) == "ssh"
}

func ReadFile(filePath string) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	return content, errorutils.CheckError(err)
}

func GetFileDetails(filePath string, includeChecksums bool) (details *FileDetails, err error) {
	details = new(FileDetails)
	if includeChecksums {
		details.Checksum, err = calcChecksumDetails(filePath)
		if err != nil {
			return details, err
		}
	} else {
		details.Checksum = entities.Checksum{}
	}

	file, err := os.Open(filePath)
	defer func() {
		if file != nil {
			err = errors.Join(err, errorutils.CheckError(file.Close()))
		}
	}()
	if err != nil {
		return nil, errorutils.CheckError(err)
	}
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, errorutils.CheckError(err)
	}
	details.Size = fileInfo.Size()
	return details, nil
}

func calcChecksumDetails(filePath string) (checksum entities.Checksum, err error) {
	file, err := os.Open(filePath)
	defer func() {
		if file != nil {
			err = errors.Join(err, errorutils.CheckError(file.Close()))
		}
	}()
	if err != nil {
		return entities.Checksum{}, errorutils.CheckError(err)
	}
	return calcChecksumDetailsFromReader(file)
}

func GetFileDetailsFromReader(reader io.Reader, includeChecksums bool) (details *FileDetails, err error) {
	details = new(FileDetails)
	if !includeChecksums {
		// io.Copy copies from the reader to io.Discard and returns the number of bytes copied
		details.Size, err = io.Copy(io.Discard, reader)
		return
	}
	pr, pw := io.Pipe()
	defer func() {
		err = errors.Join(err, errorutils.CheckError(pr.Close()))
	}()

	go func() {
		defer func() {
			err = errors.Join(err, errorutils.CheckError(pw.Close()))
		}()
		details.Size, err = io.Copy(pw, reader)
	}()

	details.Checksum, err = calcChecksumDetailsFromReader(pr)
	return
}

func calcChecksumDetailsFromReader(reader io.Reader) (entities.Checksum, error) {
	checksums, err := crypto.CalcChecksums(reader)
	if err != nil {
		return entities.Checksum{}, errorutils.CheckError(err)
	}
	return entities.Checksum{Md5: checksums[crypto.MD5], Sha1: checksums[crypto.SHA1], Sha256: checksums[crypto.SHA256]}, nil
}

type FileDetails struct {
	Checksum entities.Checksum
	Size     int64
}

// Removing the provided path from the filesystem
func RemovePath(testPath string) error {
	if file, err := os.Stat(testPath); err == nil {
		if file.IsDir() {
			err = RemoveTempDir(testPath)
		} else {
			err = errorutils.CheckError(os.Remove(testPath))
		}
		// Delete the path
		if err != nil {
			return errors.New("Cannot remove path: " + testPath + " due to: " + err.Error())
		}
	}
	return nil
}

// Renaming from old path to new path.
func RenamePath(oldPath, newPath string) error {
	err := biutils.CopyDir(oldPath, newPath, true, nil)
	if err != nil {
		return errors.New("Error copying directory: " + oldPath + "to" + newPath + err.Error())
	}
	return RemovePath(oldPath)
}

// Returns the path to the directory in which itemToFind is located.
// Traversing through directories from current work-dir to root.
// itemType determines whether looking for a file or dir.
func FindUpstream(itemToFInd string, itemType ItemType) (wd string, exists bool, err error) {
	// Create a map to store all paths visited, to avoid running in circles.
	visitedPaths := make(map[string]bool)
	// Get the current directory.
	wd, err = os.Getwd()
	if err != nil {
		return
	}
	origWd := wd
	defer func() {
		err = errors.Join(err, errorutils.CheckError(os.Chdir(origWd)))
	}()
	// Get the OS root.
	osRoot := os.Getenv("SYSTEMDRIVE")
	if osRoot != "" {
		// If this is a Windows machine:
		osRoot += "\\"
	} else {
		// Unix:
		osRoot = "/"
	}

	// Check if the current directory includes itemToFind. If not, check the parent directory
	// and so on.
	exists = false
	for {
		// If itemToFind is found in the current directory, return the path.
		switch itemType {
		case Any:
			exists = IsPathExists(filepath.Join(wd, itemToFInd), false)
		case File:
			exists, err = IsFileExists(filepath.Join(wd, itemToFInd), false)
		case Dir:
			exists, err = IsDirExists(filepath.Join(wd, itemToFInd), false)
		}
		if err != nil || exists {
			return
		}

		// If this the OS root, we can stop.
		if wd == osRoot {
			break
		}

		// Save this path.
		visitedPaths[wd] = true
		// CD to the parent directory.
		wd = filepath.Dir(wd)
		err = os.Chdir(wd)
		if err != nil {
			return "", false, err
		}

		// If we already visited this directory, it means that there's a loop and we can stop.
		if visitedPaths[wd] {
			return "", false, nil
		}
	}

	return "", false, nil
}

type ItemType string

// Returns true if the two files have the same MD5 checksum.
func FilesIdentical(file1 string, file2 string) (bool, error) {
	srcDetails, err := GetFileDetails(file1, true)
	if err != nil {
		return false, err
	}
	toCompareDetails, err := GetFileDetails(file2, true)
	if err != nil {
		return false, err
	}
	return srcDetails.Checksum.Md5 == toCompareDetails.Checksum.Md5, nil
}

// JSONEqual compares the JSON from two files.
func JsonEqual(filePath1, filePath2 string) (isEqual bool, err error) {
	reader1, err := os.Open(filePath1)
	if err != nil {
		return false, err
	}
	defer func() {
		err = errors.Join(err, errorutils.CheckError(reader1.Close()))
	}()
	reader2, err := os.Open(filePath2)
	if err != nil {
		return false, err
	}
	defer func() {
		err = errors.Join(err, errorutils.CheckError(reader2.Close()))
	}()
	var j, j2 interface{}
	d := json.NewDecoder(reader1)
	if err := d.Decode(&j); err != nil {
		return false, err
	}
	d = json.NewDecoder(reader2)
	if err := d.Decode(&j2); err != nil {
		return false, err
	}
	return reflect.DeepEqual(j2, j), nil
}

// Compares provided Md5 and Sha1 to those of a local file.
func IsEqualToLocalFile(localFilePath, md5, sha1 string) (bool, error) {
	if md5 == "" || sha1 == "" {
		// If not received checksums from downloaded file, no need to calculate local ones
		return false, nil
	}
	exists, err := IsFileExists(localFilePath, false)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}
	localFileDetails, err := GetFileDetails(localFilePath, true)
	if err != nil {
		return false, err
	}
	return localFileDetails.Checksum.Md5 == md5 && localFileDetails.Checksum.Sha1 == sha1, nil
}

// Move directory content from one path to another.
func MoveDir(fromPath, toPath string) error {
	err := CreateDirIfNotExist(toPath)
	if err != nil {
		return err
	}

	files, err := ListFiles(fromPath, true)
	if err != nil {
		return err
	}

	for _, v := range files {
		dir, err := IsDirExists(v, true)
		if err != nil {
			return err
		}

		if dir {
			toPath := toPath + GetFileSeparator() + filepath.Base(v)
			err := MoveDir(v, toPath)
			if err != nil {
				return err
			}
			continue
		}
		err = MoveFile(v, filepath.Join(toPath, filepath.Base(v)))
		if err != nil {
			return err
		}
	}
	return err
}

// GoLang: os.Rename() give error "invalid cross-device link" for Docker container with Volumes.
// MoveFile(source, destination) will work moving file between folders
// Therefore, we are using our own implementation (MoveFile) in order to rename files.
func MoveFile(sourcePath, destPath string) (err error) {
	inputFileOpen := true
	var inputFile *os.File
	inputFile, err = os.Open(sourcePath)
	if err != nil {
		return errorutils.CheckError(err)
	}
	defer func() {
		if inputFileOpen {
			err = errors.Join(err, errorutils.CheckError(inputFile.Close()))
		}
	}()
	inputFileInfo, err := inputFile.Stat()
	if err != nil {
		return errorutils.CheckError(err)
	}

	var outputFile *os.File
	outputFile, err = os.Create(destPath)
	if err != nil {
		return errorutils.CheckError(err)
	}
	defer func() {
		err = errors.Join(err, errorutils.CheckError(outputFile.Close()))
	}()

	_, err = io.Copy(outputFile, inputFile)
	if err != nil {
		return errorutils.CheckError(err)
	}
	err = os.Chmod(destPath, inputFileInfo.Mode())
	if err != nil {
		return errorutils.CheckError(err)
	}

	// The copy was successful, so now delete the original file
	err = inputFile.Close()
	if err != nil {
		return errorutils.CheckError(err)
	}
	inputFileOpen = false
	err = os.Remove(sourcePath)
	return errorutils.CheckError(err)
}

// RemoveDirContents removes the contents of the directory, without removing the directory itself.
// If it encounters an error before removing all the files, it stops and returns that error.
func RemoveDirContents(dirPath string) (err error) {
	d, err := os.Open(dirPath)
	if err != nil {
		return errorutils.CheckError(err)
	}
	defer func() {
		err = errors.Join(err, errorutils.CheckError(d.Close()))
	}()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return errorutils.CheckError(err)
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dirPath, name))
		if err != nil {
			return errorutils.CheckError(err)
		}
	}
	return nil
}
