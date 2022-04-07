package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"hash"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	ignore "github.com/sabhiram/go-gitignore"
)

func FileExists(path string) bool {
	fileInfo, err := os.Stat(path)
	return err == nil && fileInfo.Mode().IsRegular()
}
func DirExists(path string) bool {
	dirInfo, err := os.Stat(path)
	return err == nil && dirInfo.IsDir()
}
func FileRead(file *os.File, data []byte) (uint64, error) {
	i, err := file.Read(data)
	return uint64(i), err
}
func FileWrite(file *os.File, data []byte) (uint64, error) {
	i, err := file.Write(data)
	return uint64(i), err
}
func HashWrite(hash *hash.Hash, data []byte) (uint64, error) {
	i, err := (*hash).Write(data)
	return uint64(i), err
}
func DualWrite(file *os.File, hash *hash.Hash, data []byte) (uint64, error) {
	i, err := FileWrite(file, data)
	if err != nil {
		if i == 0 {
			return i, err
		} else {
			(*hash).Write(data[:i])
			return i, err
		}
	}

	return HashWrite(hash, data)
}
func DualWrite_Uint16(file *os.File, hash *hash.Hash, data uint16) (uint64, error) {
	var buffer [2]byte
	binary.BigEndian.PutUint16(buffer[:], data)
	return DualWrite(file, hash, buffer[:])
}
func DualWrite_Uint32(file *os.File, hash *hash.Hash, data uint32) (uint64, error) {
	var buffer [4]byte
	binary.BigEndian.PutUint32(buffer[:], data)
	return DualWrite(file, hash, buffer[:])
}
func DualWrite_Uint64(file *os.File, hash *hash.Hash, data uint64) (uint64, error) {
	var buffer [8]byte
	binary.BigEndian.PutUint64(buffer[:], data)
	return DualWrite(file, hash, buffer[:])
}
func DualWrite_String(file *os.File, hash *hash.Hash, str string) (uint64, error) {
	data := []byte(str)
	i, err := DualWrite_Uint16(file, hash, uint16(len(data)))
	total := i
	if err != nil {
		return total, err
	}

	i, err = DualWrite(file, hash, data)
	total += i
	return total, err
}
func DualWrite_File(file *os.File, hash *hash.Hash, path string) (uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	buf := make([]byte, 1024*1024)

	var nread uint64 = 0
	var nwritten uint64 = 0

	for {
		i, err := FileRead(f, buf)
		nread += uint64(i)

		if err != nil {
			if err != io.EOF {
				return nwritten, err
			}
			break
		}

		i, err = DualWrite(file, hash, buf[:i])
		nwritten += uint64(i)
		if err != nil {
			return nwritten, err
		}
	}

	return nwritten, nil
}

func FileHash(path string) ([]byte, uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	buf := make([]byte, 1024*1024)
	hash := sha256.New()

	var size uint64 = 0

	for {
		nread, err := f.Read(buf)
		size += uint64(nread)

		if err != nil {
			if err != io.EOF {
				return nil, 0, err
			}
			break
		}

		hash.Write(buf[:nread])
	}

	return hash.Sum(nil), size, nil
}

var outputFilePath string

func crawl(cwdPath, gitignoreBasePath string, gitignore *ignore.GitIgnore) []string {
	if FileExists(filepath.Join(cwdPath, ".gitignore")) {
		ignore, err := ignore.CompileIgnoreFile(filepath.Join(cwdPath, ".gitignore"))
		if err == nil {
			gitignore = ignore
			gitignoreBasePath = cwdPath
		}
	}

	entries, err := ioutil.ReadDir(cwdPath)
	if err != nil {
		log.Fatalln("[0] ERR:" + err.Error())
	}

	var paths []string
	for _, entry := range entries {
		entryPath, err := filepath.Abs(filepath.Join(cwdPath, entry.Name()))
		if err != nil {
			log.Fatalln("[1] ERR:" + err.Error())
			continue
		}

		if strings.Contains(entryPath, "$RECYCLE.BIN") {
			continue
		}

		if gitignore != nil {
			relToIgnoreBase, err := filepath.Rel(gitignoreBasePath, entryPath)
			if err != nil {
				log.Fatalln("[2] ERR: " + err.Error())
			}
			if gitignore.MatchesPath(".\\"+relToIgnoreBase) || gitignore.MatchesPath(relToIgnoreBase) {
				continue
			}
		}

		if entry.IsDir() {
			if entry.Name() != ".git" {
				paths = append(paths, crawl(entryPath, gitignoreBasePath, gitignore)...)
			}
		} else {
			if entryPath == outputFilePath {
				continue
			}

			paths = append(paths, entryPath)
		}
	}

	return paths
}

type BlobRecord struct {
	Hash      []byte // offs:  0 size: 32
	Size      uint64 // offs: 32 size:  8
	FileCount uint32 // offs: 40 size:  4

	// Will not be written to file
	Path string
}
type FileRecord struct {
	BlobIndex uint32 // offs: 0 size: 4
	Name      string // offs: 4 size: ?
}
type DirectoryRecord struct {
	Name        string
	Files       []FileRecord
	Directories []DirectoryRecord
}
type PackedFile struct {
	FileCount      uint32          // offs:  4 size:  4
	UnpackedSize   uint64          // offs:  8 size:  8
	RootDirectory  DirectoryRecord // offs: 16 size:  ?
	BlobRecords    []BlobRecord    // offs:  ? size:  4 + ?
	PackedFileHash []byte          // offs:  ? size: 32
}

func (br *BlobRecord) Write(file *os.File, hash *hash.Hash) (uint64, error) {
	total, err := DualWrite(file, hash, br.Hash)
	if err != nil {
		return total, err
	}
	i, err := DualWrite_Uint64(file, hash, br.Size)
	total += i
	if err != nil {
		return total, err
	}
	i, err = DualWrite_Uint32(file, hash, br.FileCount)
	total += i
	return total, err
}

func (fr *FileRecord) Write(file *os.File, hash *hash.Hash) (uint64, error) {
	total, err := DualWrite_Uint32(file, hash, fr.BlobIndex)
	if err != nil {
		return total, err
	}

	i, err := DualWrite_String(file, hash, fr.Name)
	total += i
	return total, err
}

func (dr *DirectoryRecord) Write(file *os.File, hash *hash.Hash) (uint64, error) {
	var total uint64

	i, err := DualWrite_String(file, hash, dr.Name)
	total = uint64(i)
	if err != nil {
		return total, err
	}

	i, err = DualWrite_Uint32(file, hash, uint32(len(dr.Files)))
	total += uint64(i)
	if err != nil {
		return total, err
	}
	for _, entry := range dr.Files {
		i, err = entry.Write(file, hash)
		total += uint64(i)
		if err != nil {
			return total, err
		}
	}
	i, err = DualWrite_Uint32(file, hash, uint32(len(dr.Directories)))
	total += uint64(i)
	if err != nil {
		return total, err
	}
	for _, entry := range dr.Directories {
		i64, err := entry.Write(file, hash)
		total += uint64(i64)
		if err != nil {
			break
		}
	}

	return total, err
}

type FileInfo struct {
	Path string
	Hash []byte
	Size uint64
}

/*
   hash, size, err := FileHash(entryPath)
   if err != nil {
       log.Fatalln("[3] ERR: " + err.Error())
   }

   paths = append(paths, FileInfo{
       Path: entryPath,
       Hash: hash,
       Size: size,
   })
*/

func (pf *PackedFile) RegisterFile(fileInfo FileInfo, relPath string) {
	blobFound := false
	blobIndex := 0
	for i, blob := range pf.BlobRecords {
		if bytes.Equal(blob.Hash, fileInfo.Hash) {
			pf.BlobRecords[i].FileCount++
			blobFound = true
			blobIndex = i
		}
	}
	if !blobFound {
		blobIndex = len(pf.BlobRecords)
		pf.BlobRecords = append(pf.BlobRecords, BlobRecord{
			Hash:      fileInfo.Hash,
			Size:      fileInfo.Size,
			FileCount: 1,
			Path:      fileInfo.Path,
		})
	}

	cwd := &pf.RootDirectory
	pathParts := strings.Split(relPath, "\\")
	for i, part := range pathParts {
		if i == len(pathParts)-1 {
			cwd.Files = append(cwd.Files, FileRecord{Name: part, BlobIndex: uint32(blobIndex)})
		} else {
			found := false
			for i, dir := range cwd.Directories {
				if dir.Name == part {
					cwd = &cwd.Directories[i]
					found = true
				}
			}
			if !found {
				cwd.Directories = append(cwd.Directories, DirectoryRecord{Name: part})
				cwd = &cwd.Directories[len(cwd.Directories)-1]
			}
		}
	}

	pf.FileCount++
	pf.UnpackedSize += fileInfo.Size
}
func (pf *PackedFile) WriteAll(file *os.File) (uint64, error) {
	var err error

	hash := sha256.New()

	log.Println("Writing header...")
	// Write header
	buffer := make([]byte, 16)
	buffer[0] = 'O' // Opto's
	buffer[1] = 'S' // Smart
	buffer[2] = 'P' // Packing
	buffer[3] = 'F' // Format
	binary.BigEndian.PutUint32(buffer[4:8], pf.FileCount)
	binary.BigEndian.PutUint64(buffer[8:16], pf.UnpackedSize)
	i, err := DualWrite(file, &hash, buffer)
	total := uint64(i)
	if err != nil {
		return total, err
	}

	log.Println("Writing records...")
	// Write FileRecords
	i64, err := pf.RootDirectory.Write(file, &hash)
	total += i64
	if err != nil {
		return total, err
	}

	log.Println("Writing blobs...")
	// Write Blobs
	i, err = DualWrite_Uint32(file, &hash, uint32(len(pf.BlobRecords)))
	total += uint64(i)
	if err != nil {
		return total, err
	}
	for _, entry := range pf.BlobRecords {
		log.Printf("Writing BLOB %s...\n", hex.EncodeToString(entry.Hash))
		i, err = entry.Write(file, &hash)
		total += uint64(i)
		if err != nil {
			return total, err
		}
		i64, err = DualWrite_File(file, &hash, entry.Path)
		total += uint64(i64)
		if err != nil {
			return total, err
		}
	}

	pf.PackedFileHash = hash.Sum(nil)

	// Write Hash
	i, err = DualWrite(file, &hash, pf.PackedFileHash)
	total += uint64(i)
	return total, err
}

func randSlice(slice *[]string) {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(*slice), func(i, j int) { (*slice)[i], (*slice)[j] = (*slice)[j], (*slice)[i] })
}
func chunkSlice(slice []string, chunkSize int) [][]string {
	var chunks [][]string

	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize

		// necessary check to avoid slicing beyond
		// slice capacity
		if end > len(slice) {
			end = len(slice)
		}

		chunks = append(chunks, slice[i:end])
	}

	return chunks
}

func main() {
	if len(os.Args) != 3 {
		println("Usage: Packer.exe InputFolder OutputFile")
		return
	}

	var err error
	//var relPath string

	inputDir := os.Args[1]
	outputFile := os.Args[2]

	if !DirExists(inputDir) {
		println("Input directory doesnt exist!")
		return
	}

	outputFilePath, err = filepath.Abs(outputFile)
	if err != nil {
		println("Failed to resolve output file path: " + err.Error())
		return
	}

	file, err := os.Create(outputFilePath)
	if err != nil {
		println("Failed creating output file: " + err.Error())
		return
	}
	defer file.Close()

	var filePaths []string
	filePathsCache, err := ioutil.ReadFile("cache-paths.json")
	if err == nil {
		log.Println("Loading paths from cache...")
		json.Unmarshal(filePathsCache, &filePaths)
	} else {
		log.Println("Crawling...")
		filePaths = crawl(inputDir, "", nil)
		filePathsCache, _ = json.MarshalIndent(filePaths, "", " ")
		_ = ioutil.WriteFile("cache-paths.json", filePathsCache, 0644)
	}

	var fileRecords []FileRecord
	fileRecordsCache, err := ioutil.ReadFile("cache-records.json")
	if err == nil {
		log.Println("Loading records from cache...")
		json.Unmarshal(fileRecordsCache, &fileRecords)
	} else {
		log.Println("Hashing records...")

		randSlice(&filePaths)
		workLoads := chunkSlice(filePaths, runtime.NumCPU())

        

		fileRecordsCache, _ = json.MarshalIndent(fileRecords, "", " ")
		_ = ioutil.WriteFile("cache-records.json", fileRecordsCache, 0644)
	}

	var document PackedFile
	document.RootDirectory.Name = "."

	log.Println("Indexing files...") /*
		lastPrint := time.Now()
		for i, entry := range crawlerResult {
			relPath, err = filepath.Rel(inputDir, entry.Path)
			if err != nil {
				log.Fatalln("[4] ERR: " + err.Error())
			}

			document.RegisterFile(entry, relPath)

			if time.Since(lastPrint).Seconds() > 0.25 {
				percentComplete := (float32(i) / float32(len(crawlerResult))) * 100.
				log.Printf("Indexing files... %.2f%% done\n", percentComplete)
				lastPrint = time.Now()
			}
		}

		log.Println("Writing to file...")
		document.WriteAll(file)*/
}
