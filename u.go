package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Global configurations
const (
	// Storage directories
	UploadDirBase = "data"
	OwnersDir     = "owners"
	MetadataDir   = "metadata"
	
	// HTTP server settings
	HTTPPort = "8081"
	
	// P2P server settings
	P2PPort = "8080"
	
	// Buffer size for file transfers
	BufferSize = 32 * 1024 // 32KB per buffer
	
	// P2P protocol commands
	CmdList    = byte(1)
	CmdGetFile = byte(2)
	CmdPutFile = byte(3)
	CmdError   = byte(4)
	CmdSuccess = byte(5)
)

// Metadata structure
type Metadata struct {
	User        string `json:"user"`
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

// P2P sync result structure
type SyncResult struct {
	Server      string   `json:"server"`
	Status      string   `json:"status"`
	Downloaded  []string `json:"downloaded"`
	Uploaded    []string `json:"uploaded"`
	Errors      []string `json:"errors"`
	ElapsedTime string   `json:"elapsed_time"`
}

// Calculate SHA-256 hash
func calculateSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// Check if a string is a valid SHA-256 hash
func isValidSHA256(input string) bool {
	matched, _ := regexp.MatchString(`^[a-fA-F0-9]{64}$`, input)
	return matched
}

// Check and process SHA-256 hash
func checkSHA256(input string) string {
	if isValidSHA256(input) {
		return input
	}
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// Create directories if they don't exist
func ensureDirectoriesExist() {
	dirs := []string{UploadDirBase, OwnersDir, MetadataDir}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			os.MkdirAll(dir, 0777)
		}
	}
}

// Save file using hash pattern
func saveFileWithHashPattern(fileContent []byte, fileExtension string, originalFileName string, category string, btcInfo string, metadata *Metadata) (string, string, error) {
	// Calculate hashes
	fileHash := calculateSHA256(fileContent)
	categoryHash := checkSHA256(category)
	
	// Build directory paths
	fileNameWithExtension := fileHash + "." + fileExtension
	fileUploadDir := filepath.Join(UploadDirBase, fileHash)
	categoryDir := filepath.Join(UploadDirBase, categoryHash)
	
	// Create directories if they don't exist
	os.MkdirAll(fileUploadDir, 0777)
	os.MkdirAll(categoryDir, 0777)
	
	// Save the content
	destinationFilePath := filepath.Join(fileUploadDir, fileNameWithExtension)
	
	// Check if file already exists
	//if _, err := os.Stat(destinationFilePath); err == nil {
		//return "", "", fmt.Errorf("File already exists")
	//}
	
	// Save the file
	err := ioutil.WriteFile(destinationFilePath, fileContent, 0666)
	if err != nil {
		return "", "", fmt.Errorf("Error saving content: %v", err)
	}
	
	// Save BTC info if provided
	if btcInfo != "" {
		btcFilePath := filepath.Join(OwnersDir, fileHash)
		if _, err := os.Stat(btcFilePath); os.IsNotExist(err) {
			ioutil.WriteFile(btcFilePath, []byte(btcInfo), 0666)
		}
	}
	
	// Save metadata if provided
	if metadata != nil && metadata.User != "" && metadata.Title != "" && metadata.Description != "" && metadata.URL != "" {
		metadataBytes, _ := json.MarshalIndent(metadata, "", "  ")
		metadataFilePath := filepath.Join(MetadataDir, fileHash+".json")
		
		if _, err := os.Stat(metadataFilePath); os.IsNotExist(err) {
			ioutil.WriteFile(metadataFilePath, metadataBytes, 0666)
		}
	}
	
	// Create empty file in category folder with hash + extension name
	categoryFilePath := filepath.Join(categoryDir, fileNameWithExtension)
	emptyFile, err := os.Create(categoryFilePath)
	if err != nil {
		return "", "", fmt.Errorf("Error creating empty file in category folder: %v", err)
	}
	emptyFile.Close()
	
	// Default content for index.html header
	contentHead := "<link rel='stylesheet' href='../../default.css'><script src='../../default.js'></script><script src='../../ads.js'></script><div id='ads' name='ads' class='ads'></div><div id='default' name='default' class='default'></div>"
	
	// Handle index.html inside file hash folder (for content links)
	indexPathFileFolder := filepath.Join(fileUploadDir, "index.html")
	var indexContentFileFolder string
	
	if _, err := os.Stat(indexPathFileFolder); os.IsNotExist(err) {
		indexContentFileFolder = contentHead
	} else {
		indexContentFileBytes, _ := ioutil.ReadFile(indexPathFileFolder)
		indexContentFileFolder = string(indexContentFileBytes)
	}
	
	linkReply := fmt.Sprintf("<a href=\"../../?reply=%s\">[ Reply ]</a> ", fileHash)
	linkToHash := linkReply + fmt.Sprintf("<a href=\"../%s/index.html\">[ Open ]</a> ", fileHash)
	linkToFileFolderIndex := linkToHash + fmt.Sprintf("<a href=\"%s\">%s</a><br>", fileNameWithExtension, originalFileName)
	
	if !strings.Contains(indexContentFileFolder, linkToFileFolderIndex) {
		indexContentFileFolder += linkToFileFolderIndex
		ioutil.WriteFile(indexPathFileFolder, []byte(indexContentFileFolder), 0666)
	}
	
	// Handle index.html inside category folder (for link to original content)
	indexPathCategoryFolder := filepath.Join(categoryDir, "index.html")
	var indexContentCategoryFolder string
	
	if _, err := os.Stat(indexPathCategoryFolder); os.IsNotExist(err) {
		indexContentCategoryFolder = contentHead
	} else {
		indexContentCategoryBytes, _ := ioutil.ReadFile(indexPathCategoryFolder)
		indexContentCategoryFolder = string(indexContentCategoryBytes)
	}
	
	// Build relative path to content in content hash folder
	relativePathToFile := fmt.Sprintf("../%s/%s", fileHash, fileNameWithExtension)
	
	categoryReply := fmt.Sprintf("<a href=\"../../?reply=%s\">[ Reply ]</a> ", fileHash)
	linkToHashCategory := categoryReply + fmt.Sprintf("<a href=\"../%s/index.html\">[ Open ]</a> ", fileHash)
	linkToCategoryFolderIndex := linkToHashCategory + fmt.Sprintf("<a href=\"%s\">%s</a><br>", relativePathToFile, originalFileName)
	
	if !strings.Contains(indexContentCategoryFolder, linkToCategoryFolderIndex) {
		indexContentCategoryFolder += linkToCategoryFolderIndex
		ioutil.WriteFile(indexPathCategoryFolder, []byte(indexContentCategoryFolder), 0666)
	}
	
	return fileHash, indexPathCategoryFolder, nil
}

// sha256Hash generates SHA-256 hash for a message
func sha256Hash(message string) string {
	hash := sha256.Sum256([]byte(message))
	return hex.EncodeToString(hash[:])
}

// Handler for search
func searchHandler(w http.ResponseWriter, r *http.Request) {
        uploadDirBase := "data_tmp"
	searchInput := strings.TrimSpace(r.URL.Query().Get("search-input"))
	if searchInput == "" {
		return
	}

	// Check if input is already a valid SHA-256 hash (64 hex characters)
	isValidHash, _ := regexp.MatchString(`^[a-fA-F0-9]{64}$`, searchInput)

	var hash string
	if isValidHash {
		// If input is already a valid hash, use it directly
		hash = searchInput
	} else {
		// Otherwise generate SHA-256 hash of the input
		hash = sha256Hash(searchInput)
	}

	// Check if the file exists
	if _, err := os.Stat(filepath.Join(uploadDirBase, hash, "index.html")); err == nil {
		// Redirect to the page
		http.Redirect(w, r, filepath.Join(uploadDirBase, hash, "index.html"), http.StatusFound)
		return
	}

	// File doesn't exist
	fmt.Fprintf(w, "File don't exists!")
}

// Handler for upload and P2P
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Ensure directories exist
		ensureDirectoriesExist()
		
		// Check if it's a P2P sync
		if r.FormValue("p2p_sync") == "true" {
			handleP2PSync(w, r)
			return
		}
		
		// Check if category was provided
		category := r.FormValue("category")
		if category == "" {
			fmt.Fprint(w, "<p class='error'>Please select a file or enter text content and provide a category.</p>")
			renderMainPage(w, r, "", nil)
			return
		}
		
		var fileContent []byte
		var originalFileName string
		var fileExtension string
		var isTextContent bool

                if isTextContent{}
		
		// Check if a file was uploaded
		file, header, err := r.FormFile("uploaded_file")
		if err == nil {
			defer file.Close()
			fileContent, err = ioutil.ReadAll(file)
			if err != nil {
				fmt.Fprint(w, "<p class='error'>Error reading uploaded file.</p>")
				renderMainPage(w, r, "", nil)
				return
			}
			originalFileName = header.Filename
			fileExtension = filepath.Ext(originalFileName)
			if fileExtension != "" {
				fileExtension = fileExtension[1:] // Remove leading dot
			}
			isTextContent = false
		} else {
			// If no file, check for text content
			textContent := r.FormValue("text_content")
			if textContent != "" {
				fileContent = []byte(textContent)
				date := time.Now().Format("2006.01.02 15:04:05")
				
				if len(textContent) > 50 {
					originalFileName = textContent[:50] + " (" + date + ")"
				} else {
					originalFileName = textContent + " (" + date + ")"
				}
				
				fileExtension = "txt"
				isTextContent = true
			} else {
				fmt.Fprint(w, "<p class='error'>Please select a file or enter text content.</p>")
				renderMainPage(w, r, "", nil)
				return
			}
		}
		
		// Check if there's content to process
		if len(fileContent) == 0 {
			fmt.Fprint(w, "<p class='error'>No content to process.</p>")
			renderMainPage(w, r, "", nil)
			return
		}
		
		// Check if it's a PHP file (not allowed)
		if strings.ToLower(fileExtension) == "php" {
			fmt.Fprint(w, "<p class='error'>Error: PHP files are not allowed!</p>")
			renderMainPage(w, r, "", nil)
			return
		}
		
		// Check if category is the same as text content
		if category == r.FormValue("text_content") {
			fmt.Fprint(w, "<p class='error'>Error: Category can't be the same of text contents.</p>")
			renderMainPage(w, r, "", nil)
			return
		}
		
		// Prepare metadata
		metadata := &Metadata{
			User:        r.FormValue("user"),
			Title:       r.FormValue("title"),
			Description: r.FormValue("description"),
			URL:         r.FormValue("url"),
		}
		
		// Save the file with hash pattern
		_, indexPathCategoryFolder, err := saveFileWithHashPattern(
			fileContent,
			fileExtension,
			originalFileName,
			category,
			r.FormValue("btc"),
			metadata,
		)
		
		if err != nil {
			fmt.Fprintf(w, "<p class='error'>%s</p>", err.Error())
			renderMainPage(w, r, "", nil)
			return
		}
		
		// Display success message
		fmt.Fprintf(w, "<p class='success'>Content processed successfully!</p>")
		fmt.Fprintf(w, "<p>Content saved in: <pre><a href='/%s'>%s</a></pre></p>", indexPathCategoryFolder, indexPathCategoryFolder)
		
		renderMainPage(w, r, "", nil)
	} else {
		// If not POST, render main page
		reply := r.URL.Query().Get("reply")
		renderMainPage(w, r, reply, nil)
	}
}

// Handler for P2P sync
func handleP2PSync(w http.ResponseWriter, r *http.Request) {
	serversText := r.FormValue("servers")
	// Removida a verificação do checkbox "allow_send_files"
	// Agora sempre fará upload e download
	servers := strings.Split(serversText, "\n")
	
	var results []SyncResult
	var wg sync.WaitGroup
	var mutex sync.Mutex
	
	for _, server := range servers {
		server = strings.TrimSpace(server)
		if server == "" {
			continue
		}
		
		wg.Add(1)
		go func(srv string) {
			defer wg.Done()
			// Modificado para sempre permitir envio e download
			result := p2pSyncWithServer(srv, true)
			
			mutex.Lock()
			results = append(results, result)
			mutex.Unlock()
		}(server)
	}
	
	wg.Wait()
	
	// Render main page with results
	renderMainPage(w, r, "", results)
}

// Render main page
func renderMainPage(w http.ResponseWriter, r *http.Request, reply string, p2pResults []SyncResult) {
	htmlTemplate := `<!DOCTYPE html>
<html>
<head>
    <title>File/Text Upload with Category and P2P</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
        }

        body {
            max-width: 800px;
            margin: 40px auto;
            padding: 0 20px;
            color: #333;
            line-height: 1.6;
        }

        h2, h3 {
            font-size: 1.5em;
            margin-bottom: 20px;
            color: #222;
        }

        form {
            display: flex;
            flex-direction: column;
            gap: 15px;
            margin-bottom: 30px;
        }

        label {
            font-size: 0.9em;
            color: #444;
        }

        input[type="text"],
        input[type="url"],
        textarea,
        input[type="file"] {
            width: 100%;
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 5px;
            font-size: 1em;
            transition: border-color 0.2s;
        }

        input[type="text"]:focus,
        input[type="url"]:focus,
        textarea:focus {
            outline: none;
            border-color: #4a90e2;
        }

        textarea {
            resize: vertical;
            min-height: 100px;
        }

        button, input[type="submit"] {
            padding: 10px 20px;
            background: #4a90e2;
            color: white;
            border: none;
            border-radius: 5px;
            font-size: 1em;
            cursor: pointer;
            transition: background 0.2s;
        }

        button:hover, input[type="submit"]:hover {
            background: #357abd;
        }

        .search-form {
            flex-direction: row;
            gap: 10px;
            margin-bottom: 30px;
        }

        .search-form input {
            flex: 1;
        }

        .more-options-link {
            color: #4a90e2;
            font-size: 0.9em;
            cursor: pointer;
            text-decoration: none;
            margin: 10px 0;
            display: inline-block;
        }

        .more-options-link:hover {
            text-decoration: underline;
        }

        .optional-fields {
            display: none;
            padding: 15px;
            background: #f8f9fa;
            border-radius: 5px;
        }

        .success {
            color: #2ecc71;
            font-size: 0.9em;
        }

        .error {
            color: #e74c3c;
            font-size: 0.9em;
        }

        .section-divider {
            border-top: 1px solid #ddd;
            margin: 30px 0;
            padding-top: 20px;
        }

        .checkbox-container {
            display: flex;
            align-items: center;
            gap: 10px;
            margin: 10px 0;
        }

        .checkbox-container input[type="checkbox"] {
            width: 18px;
            height: 18px;
        }

        .mode-description {
            font-size: 0.85em;
            color: #666;
            margin-top: 5px;
            margin-left: 28px;
        }

        .results {
            margin-top: 30px;
        }

        .result-card {
            background: #f8f9fa;
            border-radius: 5px;
            padding: 15px;
            margin-bottom: 20px;
            border-left: 4px solid #4a90e2;
        }

        .result-card.error {
            border-left-color: #e74c3c;
        }

        .result-card h3 {
            margin-bottom: 10px;
            display: flex;
            justify-content: space-between;
        }

        .result-card .status {
            font-size: 0.8em;
            padding: 3px 8px;
            border-radius: 3px;
            color: white;
        }

        .result-card .status.success {
            background: #2ecc71;
        }

        .result-card .status.error {
            background: #e74c3c;
        }

        .file-list {
            margin: 10px 0;
            padding-left: 20px;
        }

        .file-list li {
            font-size: 0.9em;
            margin-bottom: 5px;
            word-break: break-all;
        }

        .error-list {
            color: #e74c3c;
        }

        .time {
            font-size: 0.8em;
            color: #666;
            margin-top: 10px;
            text-align: right;
        }

        .empty-message {
            color: #666;
            font-style: italic;
        }

        .section-title {
            font-size: 1em;
            margin: 15px 0 5px 0;
            color: #555;
        }
    </style>
    <script>
        document.addEventListener('DOMContentLoaded', () => {
            const moreOptionsLink = document.getElementById('more-options-link');
            const optionalFields = document.getElementById('optional-fields');

            moreOptionsLink.addEventListener('click', () => {
                const isHidden = optionalFields.style.display === 'none' || optionalFields.style.display === '';
                optionalFields.style.display = isHidden ? 'block' : 'none';
                moreOptionsLink.textContent = isHidden ? 'Less options' : 'More options';
            });
        });
    </script>
</head>
<body>
    <form method="GET" action="/" id="search-form" class="search-form">
        <input type="text" id="search" name="search-input" placeholder="Enter file hash or category" required>
        <button type="submit">Search</button>
    </form>

    <h2>Upload File</h2>

    <form action="/" method="post" enctype="multipart/form-data">
        <label for="uploaded_file">Select File:</label>
        <input type="file" name="uploaded_file" id="uploaded_file">

        <label for="text_content">Or enter text content:</label>
        <textarea name="text_content" id="text_content" rows="5"></textarea>

        <label for="category">Category:</label>
        <input type="text" name="category" id="category" value="{{.Reply}}" required {{if .Reply}}readonly{{end}}>

        <a id="more-options-link" class="more-options-link">More options</a>
        
        <div id="optional-fields" class="optional-fields">
            <label for="btc">BTC/PIX (optional):</label>
            <input type="text" name="btc" id="btc" placeholder="BTC address">
            
            <label for="user">User (optional):</label>
            <input type="text" name="user" id="user" placeholder="Username">
            
            <label for="title">Title (optional):</label>
            <input type="text" name="title" id="title" placeholder="Content title">
            
            <label for="description">Description (optional):</label>
            <input type="text" name="description" id="description" placeholder="Content description">
            
            <label for="url">URL (optional):</label>
            <input type="url" name="url" id="url" placeholder="Related URL">
        </div>

        <input type="submit" value="Upload">
    </form>

    <div class="section-divider"></div>

    <h2>P2P Synchronization</h2>
    
    <form action="/" method="post" enctype="multipart/form-data">
        <input type="hidden" name="p2p_sync" value="true">
        
        <label for="servers">Server List (one per line, format: host:port):</label>
        <textarea name="servers" id="servers" rows="5" placeholder="example1.com:8081&#10;example2.com:8081&#10;192.168.1.100:8081">{{.ServersText}}</textarea>
        
        <!-- Checkbox "Allow sending files" removido conforme solicitado -->
        <div class="mode-description">
            O cliente enviará arquivos locais para os servidores e baixará todos os arquivos dos servidores.
        </div>
        
        <input type="submit" value="Synchronize">
    </form>

    {{if .P2PResults}}
    <div class="results">
        <h3>Synchronization Results</h3>
        
        {{range .P2PResults}}
        <div class="result-card {{if eq .Status "error"}}error{{end}}">
            <h3>
                {{.Server}}
                <span class="status {{.Status}}">{{.Status}}</span>
            </h3>
            
            {{if .Downloaded}}
            <div class="section-title">Downloaded Files:</div>
            <ul class="file-list">
                {{range .Downloaded}}
                <li>{{.}}</li>
                {{end}}
            </ul>
            {{else}}
            <div class="section-title">Downloaded Files:</div>
            <p class="empty-message">No files downloaded</p>
            {{end}}
            
            {{if .Uploaded}}
            <div class="section-title">Uploaded Files:</div>
            <ul class="file-list">
                {{range .Uploaded}}
                <li>{{.}}</li>
                {{end}}
            </ul>
            {{else}}
            <div class="section-title">Uploaded Files:</div>
            <p class="empty-message">No files uploaded</p>
            {{end}}
            
            {{if .Errors}}
            <div class="section-title">Errors:</div>
            <ul class="file-list error-list">
                {{range .Errors}}
                <li>{{.}}</li>
                {{end}}
            </ul>
            {{end}}
            
            <div class="time">Time: {{.ElapsedTime}}</div>
        </div>
        {{end}}
    </div>
    {{end}}
</body>
</html>`

	tmpl, err := template.New("index").Parse(htmlTemplate)
	if err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}

	data := struct {
		Reply          string
		ServersText    string
		P2PResults     []SyncResult
	}{
		Reply:          reply,
		ServersText:    "",
		P2PResults:     p2pResults,
	}

	tmpl.Execute(w, data)
}

// Sync with a P2P server
func p2pSyncWithServer(serverAddr string, allowSendFiles bool) SyncResult {
	result := SyncResult{
		Server:     serverAddr,
		Status:     "error",
		Downloaded: []string{},
		Uploaded:   []string{},
		Errors:     []string{},
	}
	
	startTime := time.Now()
	
	// Connect to server for file listing
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Error connecting: %v", err))
		result.ElapsedTime = time.Since(startTime).String()
		return result
	}
	
	// Send list command
	_, err = conn.Write([]byte{CmdList})
	if err != nil {
		conn.Close()
		result.Errors = append(result.Errors, fmt.Sprintf("Error sending list command: %v", err))
		result.ElapsedTime = time.Since(startTime).String()
		return result
	}
	
	// Read list size
	sizeBuffer := make([]byte, 4)
	_, err = io.ReadFull(conn, sizeBuffer)
	if err != nil {
		conn.Close()
		result.Errors = append(result.Errors, fmt.Sprintf("Error reading list size: %v", err))
		result.ElapsedTime = time.Since(startTime).String()
		return result
	}
	
	listSize := binary.BigEndian.Uint32(sizeBuffer)
	
	// Read file list
	listBuffer := make([]byte, listSize)
	_, err = io.ReadFull(conn, listBuffer)
	if err != nil {
		conn.Close()
		result.Errors = append(result.Errors, fmt.Sprintf("Error reading file list: %v", err))
		result.ElapsedTime = time.Since(startTime).String()
		return result
	}
	
	conn.Close()
	
	// Process file list
	serverFiles := strings.Split(string(listBuffer), "\n")
	
	// Para cada arquivo no servidor, baixe-o (independentemente de existir localmente ou não)
	for _, filePath := range serverFiles {
		if filePath == "" {
			continue
		}
		
		// Sempre baixa o arquivo do servidor, mesmo que já exista localmente
		downloadErr := downloadFile(serverAddr, filePath, &result)
		if downloadErr != nil {
			result.Errors = append(result.Errors, downloadErr.Error())
		}
	}
	
	// Enviar arquivos locais para o servidor (sempre ativado)
	// List local files
	localFiles := listAllFiles()
	
	// Para cada arquivo local, verifique se precisamos enviá-lo
	for _, filePath := range localFiles {
		// Verifica se o arquivo está na lista do servidor
		found := false
		for _, serverFile := range serverFiles {
			if serverFile == filePath {
				found = true
				break
			}
		}
		
		if !found {
			// Arquivo não existe no servidor, faça upload
			uploadErr := uploadFile(serverAddr, filePath, &result)
			if uploadErr != nil {
				result.Errors = append(result.Errors, uploadErr.Error())
			}
		}
	}
	
	result.Status = "success"
	result.ElapsedTime = time.Since(startTime).String()
	return result
}

// Download a file from the server
// Download a file from the server
func downloadFile(serverAddr string, filePath string, result *SyncResult) error {
    // Connect to server
    conn, err := net.Dial("tcp", serverAddr)
    if err != nil {
        return fmt.Errorf("Error connecting for download of %s: %v", filePath, err)
    }
    defer conn.Close()
    
    // Send get file command
    _, err = conn.Write([]byte{CmdGetFile})
    if err != nil {
        return fmt.Errorf("Error sending get command for %s: %v", filePath, err)
    }
    
    // Send path size
    pathSizeBuffer := make([]byte, 4)
    binary.BigEndian.PutUint32(pathSizeBuffer, uint32(len(filePath)))
    _, err = conn.Write(pathSizeBuffer)
    if err != nil {
        return fmt.Errorf("Error sending path size for %s: %v", filePath, err)
    }
    
    // Send path
    _, err = conn.Write([]byte(filePath))
    if err != nil {
        return fmt.Errorf("Error sending path for %s: %v", filePath, err)
    }
    
    // Read response status
    statusBuffer := make([]byte, 1)
    _, err = io.ReadFull(conn, statusBuffer)
    if err != nil {
        return fmt.Errorf("Error reading status for %s: %v", filePath, err)
    }
    
    status := statusBuffer[0]
    
    if status == CmdError {
        // Read error message size
        errSizeBuffer := make([]byte, 4)
        _, err = io.ReadFull(conn, errSizeBuffer)
        if err != nil {
            return fmt.Errorf("Error reading error size for %s: %v", filePath, err)
        }
        
        errSize := binary.BigEndian.Uint32(errSizeBuffer)
        
        // Read error message
        errBuffer := make([]byte, errSize)
        _, err = io.ReadFull(conn, errBuffer)
        if err != nil {
            return fmt.Errorf("Error reading error message for %s: %v", filePath, err)
        }
        
        return fmt.Errorf("Error downloading %s: %s", filePath, string(errBuffer))
    }
    
    // Read file size
    fileSizeBuffer := make([]byte, 8)
    _, err = io.ReadFull(conn, fileSizeBuffer)
    if err != nil {
        return fmt.Errorf("Error reading file size for %s: %v", filePath, err)
    }
    
    fileSize := binary.BigEndian.Uint64(fileSizeBuffer)
    
    // Create a temporary file to store the downloaded content
    tempFile, err := ioutil.TempFile("", "p2p_download_*")
    if err != nil {
        return fmt.Errorf("Error creating temp file for %s: %v", filePath, err)
    }
    defer os.Remove(tempFile.Name())
    defer tempFile.Close()
    
    // Receive file data
    buffer := make([]byte, BufferSize)
    var totalReceived uint64 = 0
    
    for totalReceived < fileSize {
        toRead := BufferSize
        if fileSize-totalReceived < uint64(BufferSize) {
            toRead = int(fileSize - totalReceived)
        }
        
        n, err := io.ReadFull(conn, buffer[:toRead])
        if err != nil {
            if err == io.EOF {
                break
            }
            return fmt.Errorf("Error receiving data for %s: %v", filePath, err)
        }
        
        _, err = tempFile.Write(buffer[:n])
        if err != nil {
            return fmt.Errorf("Error writing to temp file %s: %v", filePath, err)
        }
        
        totalReceived += uint64(n)
    }
    
    // Read the temporary file content
    fileContent, err := ioutil.ReadFile(tempFile.Name())
    if err != nil {
        return fmt.Errorf("Error reading temp file %s: %v", tempFile.Name(), err)
    }
    
    // Extract file information from the path
    fileName := filepath.Base(filePath)
    fileExt := filepath.Ext(fileName)
    if fileExt != "" {
        fileExt = fileExt[1:] // Remove leading dot
    }
    
    // Use the file name as the category
    category := strings.TrimSuffix(fileName, filepath.Ext(fileName))
    
    // Save the file using the same pattern as HTTP uploads
    fileHash, _, err := saveFileWithHashPattern(
        fileContent,
        fileExt,
        fileName,
        category,
        "",    // No BTC info
        nil,   // No metadata
    )
    
    if err != nil {
        return fmt.Errorf("Error saving downloaded file %s: %v", filePath, err)
    }
    
    // Add to downloaded files list
    result.Downloaded = append(result.Downloaded, fmt.Sprintf("%s (saved as %s)", filePath, fileHash))
    
    return nil
}

// Upload a file to the server
func uploadFile(serverAddr string, filePath string, result *SyncResult) error {
	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("Error getting file info for %s: %v", filePath, err)
	}
	
	fileSize := fileInfo.Size()
	
	// Connect to server
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return fmt.Errorf("Error connecting for upload of %s: %v", filePath, err)
	}
	defer conn.Close()
	
	// Send put file command
	_, err = conn.Write([]byte{CmdPutFile})
	if err != nil {
		return fmt.Errorf("Error sending put command for %s: %v", filePath, err)
	}
	
	// Send path size
	pathSizeBuffer := make([]byte, 4)
	binary.BigEndian.PutUint32(pathSizeBuffer, uint32(len(filePath)))
	_, err = conn.Write(pathSizeBuffer)
	if err != nil {
		return fmt.Errorf("Error sending path size for %s: %v", filePath, err)
	}
	
	// Send path
	_, err = conn.Write([]byte(filePath))
	if err != nil {
		return fmt.Errorf("Error sending path for %s: %v", filePath, err)
	}
	
	// Send file size
	fileSizeBuffer := make([]byte, 8)
	binary.BigEndian.PutUint64(fileSizeBuffer, uint64(fileSize))
	_, err = conn.Write(fileSizeBuffer)
	if err != nil {
		return fmt.Errorf("Error sending file size for %s: %v", filePath, err)
	}
	
	// Read response status
	statusBuffer := make([]byte, 1)
	_, err = io.ReadFull(conn, statusBuffer)
	if err != nil {
		return fmt.Errorf("Error reading status for %s: %v", filePath, err)
	}
	
	status := statusBuffer[0]
	
	if status == CmdError {
		// Read error message size
		errSizeBuffer := make([]byte, 4)
		_, err = io.ReadFull(conn, errSizeBuffer)
		if err != nil {
			return fmt.Errorf("Error reading error size for %s: %v", filePath, err)
		}
		
		errSize := binary.BigEndian.Uint32(errSizeBuffer)
		
		// Read error message
		errBuffer := make([]byte, errSize)
		_, err = io.ReadFull(conn, errBuffer)
		if err != nil {
			return fmt.Errorf("Error reading error message for %s: %v", filePath, err)
		}
		
		return fmt.Errorf("Error uploading %s: %s", filePath, string(errBuffer))
	}
	
	// Open file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("Error opening file %s for upload: %v", filePath, err)
	}
	defer file.Close()
	
	// Send file data
	buffer := make([]byte, BufferSize)
	for {
		n, err := file.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("Error reading file %s for upload: %v", filePath, err)
		}
		
		_, err = conn.Write(buffer[:n])
		if err != nil {
			return fmt.Errorf("Error sending data for %s: %v", filePath, err)
		}
	}
	
	// Read final status
	_, err = io.ReadFull(conn, statusBuffer)
	if err != nil {
		return fmt.Errorf("Error reading final status for %s: %v", filePath, err)
	}
	
	finalStatus := statusBuffer[0]
	
	if finalStatus == CmdError {
		// Read error message size
		errSizeBuffer := make([]byte, 4)
		_, err = io.ReadFull(conn, errSizeBuffer)
		if err != nil {
			return fmt.Errorf("Error reading final error size for %s: %v", filePath, err)
		}
		
		errSize := binary.BigEndian.Uint32(errSizeBuffer)
		
		// Read error message
		errBuffer := make([]byte, errSize)
		_, err = io.ReadFull(conn, errBuffer)
		if err != nil {
			return fmt.Errorf("Error reading final error message for %s: %v", filePath, err)
		}
		
		return fmt.Errorf("Error finalizing upload of %s: %s", filePath, string(errBuffer))
	} else {
		// Read success message size
		successSizeBuffer := make([]byte, 4)
		_, err = io.ReadFull(conn, successSizeBuffer)
		if err != nil {
			return fmt.Errorf("Error reading success size for %s: %v", filePath, err)
		}
		
		successSize := binary.BigEndian.Uint32(successSizeBuffer)
		
		// Read success message (optional)
		if successSize > 0 {
			successBuffer := make([]byte, successSize)
			_, err = io.ReadFull(conn, successBuffer)
			if err != nil {
				return fmt.Errorf("Error reading success message for %s: %v", filePath, err)
			}
		}
		
		// Add to uploaded files list
		result.Uploaded = append(result.Uploaded, filePath)
	}
	
	return nil
}

// List all files in data_tmp, metadata and owners folders
func listAllFiles() []string {
	var fileList []string
	
	// Function to walk directories recursively
	walkDir := func(dir string) {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				fileList = append(fileList, path)
			}
			return nil
		})
	}
	
	// List files in each directory
	walkDir(UploadDirBase)
	walkDir(MetadataDir)
	walkDir(OwnersDir)
	
	return fileList
}

// Handle P2P connections
func handleP2PConnection(conn net.Conn) {
	defer conn.Close()
	
	// Read command
	cmdBuffer := make([]byte, 1)
	_, err := io.ReadFull(conn, cmdBuffer)
	if err != nil {
		log.Printf("Error reading P2P command: %v", err)
		return
	}
	
	cmd := cmdBuffer[0]
	
	switch cmd {
	case CmdList:
		// List files in data_tmp, metadata and owners folders
		fileList := listAllFiles()
		fileListStr := strings.Join(fileList, "\n")
		
		// Send list size
		sizeBuffer := make([]byte, 4)
		binary.BigEndian.PutUint32(sizeBuffer, uint32(len(fileListStr)))
		conn.Write(sizeBuffer)
		
		// Send list
		conn.Write([]byte(fileListStr))
		
	case CmdGetFile:
		// Read path size
		pathSizeBuffer := make([]byte, 4)
		_, err := io.ReadFull(conn, pathSizeBuffer)
		if err != nil {
			log.Printf("Error reading path size: %v", err)
			return
		}
		
		pathSize := binary.BigEndian.Uint32(pathSizeBuffer)
		
		// Read path
		pathBuffer := make([]byte, pathSize)
		_, err = io.ReadFull(conn, pathBuffer)
		if err != nil {
			log.Printf("Error reading file path: %v", err)
			return
		}
		
		filePath := string(pathBuffer)
		
		// Check if file exists
		fileInfo, err := os.Stat(filePath)
		//if err != nil {
			// Send error
			//conn.Write([]byte{CmdError})
			//errorMsg := "File not found"
			
			// Send error message size
			//errSizeBuffer := make([]byte, 4)
			//binary.BigEndian.PutUint32(errSizeBuffer, uint32(len(errorMsg)))
			//conn.Write(errSizeBuffer)
			
			// Send error message
			//conn.Write([]byte(errorMsg))
			//return
		//}
		
		// Open file
		file, err := os.Open(filePath)
		if err != nil {
			// Send error
			conn.Write([]byte{CmdError})
			errorMsg := fmt.Sprintf("Error opening file: %v", err)
			
			// Send error message size
			errSizeBuffer := make([]byte, 4)
			binary.BigEndian.PutUint32(errSizeBuffer, uint32(len(errorMsg)))
			conn.Write(errSizeBuffer)
			
			// Send error message
			conn.Write([]byte(errorMsg))
			return
		}
		defer file.Close()
		
		// Send success
		conn.Write([]byte{CmdSuccess})
		
		// Send file size
		fileSizeBuffer := make([]byte, 8)
		binary.BigEndian.PutUint64(fileSizeBuffer, uint64(fileInfo.Size()))
		conn.Write(fileSizeBuffer)
		
		// Send file in blocks
		buffer := make([]byte, BufferSize)
		for {
			n, err := file.Read(buffer)
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Printf("Error reading file %s: %v", filePath, err)
				return
			}
			
			_, err = conn.Write(buffer[:n])
			if err != nil {
				log.Printf("Error sending file data for %s: %v", filePath, err)
				return
			}
		}
		
	case CmdPutFile:
		// Removida a verificação de AllowSendFiles
		// Agora sempre aceita arquivos
		
		// Read path size
		pathSizeBuffer := make([]byte, 4)
		_, err := io.ReadFull(conn, pathSizeBuffer)
		if err != nil {
			log.Printf("Error reading path size: %v", err)
			return
		}
		
		pathSize := binary.BigEndian.Uint32(pathSizeBuffer)
		
		// Read path
		pathBuffer := make([]byte, pathSize)
		_, err = io.ReadFull(conn, pathBuffer)
		if err != nil {
			log.Printf("Error reading file path: %v", err)
			return
		}
		
		filePath := string(pathBuffer)
		
		// Read file size
		fileSizeBuffer := make([]byte, 8)
		_, err = io.ReadFull(conn, fileSizeBuffer)
		if err != nil {
			log.Printf("Error reading file size: %v", err)
			return
		}
		
		fileSize := binary.BigEndian.Uint64(fileSizeBuffer)
		
		// Check if file already exists
		//if _, err := os.Stat(filePath); err == nil {
			// Send error
			//conn.Write([]byte{CmdError})
			//errorMsg := "File already exists"
			
			// Send error message size
			//errSizeBuffer := make([]byte, 4)
			//binary.BigEndian.PutUint32(errSizeBuffer, uint32(len(errorMsg)))
			//conn.Write(errSizeBuffer)
			
			// Send error message
			//conn.Write([]byte(errorMsg))
			//return
		//}
		
		// Create temporary directory to receive file
		tempDir := os.TempDir()
		tempFile, err := ioutil.TempFile(tempDir, "p2p_upload_*")
		if err != nil {
			// Send error
			conn.Write([]byte{CmdError})
			errorMsg := fmt.Sprintf("Error creating temporary file: %v", err)
			
			// Send error message size
			errSizeBuffer := make([]byte, 4)
			binary.BigEndian.PutUint32(errSizeBuffer, uint32(len(errorMsg)))
			conn.Write(errSizeBuffer)
			
			// Send error message
			conn.Write([]byte(errorMsg))
			return
		}
		tempFilePath := tempFile.Name()
		defer os.Remove(tempFilePath) // Remove temporary file when done
		
		// Send success to start transfer
		conn.Write([]byte{CmdSuccess})
		
		// Receive file in blocks
		buffer := make([]byte, BufferSize)
		var totalReceived uint64 = 0
		
		for totalReceived < fileSize {
			toRead := BufferSize
			if fileSize-totalReceived < uint64(BufferSize) {
				toRead = int(fileSize - totalReceived)
			}
			
			n, err := io.ReadFull(conn, buffer[:toRead])
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Printf("Error receiving file data: %v", err)
				tempFile.Close()
				return
			}
			
			_, err = tempFile.Write(buffer[:n])
			if err != nil {
				log.Printf("Error writing to temporary file: %v", err)
				tempFile.Close()
				return
			}
			
			totalReceived += uint64(n)
		}
		
		// Close temporary file
		tempFile.Close()
		
		// Read temporary file content
		fileContent, err := ioutil.ReadFile(tempFilePath)
		if err != nil {
			// Send error
			conn.Write([]byte{CmdError})
			errorMsg := fmt.Sprintf("Error reading temporary file: %v", err)
			
			// Send error message size
			errSizeBuffer := make([]byte, 4)
			binary.BigEndian.PutUint32(errSizeBuffer, uint32(len(errorMsg)))
			conn.Write(errSizeBuffer)
			
			// Send error message
			conn.Write([]byte(errorMsg))
			return
		}
		
		// Extract file information
		fileExt := filepath.Ext(filePath)
		if fileExt != "" {
			fileExt = fileExt[1:] // Remove leading dot
		}
		
		// Use original filename for display
		originalFileName := filepath.Base(filePath)
		
		// Save file with hash pattern
		fileHash, _, err := saveFileWithHashPattern(
			fileContent,
			fileExt,
			originalFileName,
			originalFileName, // Use filename as category
			"",               // No BTC info
			nil,              // No metadata
		)
		
		if err != nil {
			// Send error
			conn.Write([]byte{CmdError})
			errorMsg := fmt.Sprintf("Error saving file with hash pattern: %v", err)
			
			// Send error message size
			errSizeBuffer := make([]byte, 4)
			binary.BigEndian.PutUint32(errSizeBuffer, uint32(len(errorMsg)))
			conn.Write(errSizeBuffer)
			
			// Send error message
			conn.Write([]byte(errorMsg))
			return
		}
		
		// Send success confirmation
		conn.Write([]byte{CmdSuccess})
		successMsg := fmt.Sprintf("File saved successfully with hash: %s", fileHash)
		
		// Send success message size
		successSizeBuffer := make([]byte, 4)
		binary.BigEndian.PutUint32(successSizeBuffer, uint32(len(successMsg)))
		conn.Write(successSizeBuffer)
		
		// Send success message
		conn.Write([]byte(successMsg))
	}
}

// Start P2P server
func startP2PServer() {
	listener, err := net.Listen("tcp", ":"+P2PPort)
	if err != nil {
		log.Fatalf("Error starting P2P server: %v", err)
	}
	defer listener.Close()
	
	log.Printf("P2P server started on port %s", P2PPort)
	
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting P2P connection: %v", err)
			continue
		}
		
		go handleP2PConnection(conn)
	}
}

// Handler para arquivos estáticos - Versão corrigida
func staticFileHandler(w http.ResponseWriter, r *http.Request) {
	// Primeiro, verificar se há parâmetro de busca, independente do path
	if r.Method == http.MethodGet && r.URL.Query().Get("search-input") != "" {
		searchHandler(w, r)
		return
	}
	
	// Verificar se é a rota raiz
	path := r.URL.Path
	if path == "/" {
		uploadHandler(w, r)
		return
	}
	
	// Verificar se é um arquivo existente
	filePath := path[1:] // Remove leading slash
	if _, err := os.Stat(filePath); err == nil {
		// Serve file
		http.ServeFile(w, r, filePath)
		return
	}
	
	// Tratar outras rotas
	if r.URL.Query().Get("reply") != "" || r.Method == "POST" {
		uploadHandler(w, r)
		return
	} else {
		// Se nenhuma das condições acima, renderizar página principal
		renderMainPage(w, r, "", nil)
	}
}

// Create default CSS and JS files
func createDefaultFiles() {
	// Default CSS
	defaultCSS := `
body {
    font-family: Arial, sans-serif;
    line-height: 1.6;
    margin: 0;
    padding: 20px;
    color: #333;
}

a {
    color: #0066cc;
    text-decoration: none;
}

a:hover {
    text-decoration: underline;
}

.ads, .default {
    margin-bottom: 20px;
}
`
	
	// Default JavaScript
	defaultJS := `
document.addEventListener('DOMContentLoaded', function() {
    console.log('Page loaded');
});
`
	
	// Ads JavaScript (empty)
	adsJS := `
// Placeholder for ads
`
	
	// Save files if they don't exist
	if _, err := os.Stat("default.css"); os.IsNotExist(err) {
		ioutil.WriteFile("default.css", []byte(defaultCSS), 0666)
	}
	
	if _, err := os.Stat("default.js"); os.IsNotExist(err) {
		ioutil.WriteFile("default.js", []byte(defaultJS), 0666)
	}
	
	if _, err := os.Stat("ads.js"); os.IsNotExist(err) {
		ioutil.WriteFile("ads.js", []byte(adsJS), 0666)
	}
}

func main() {
	// Ensure directories exist
	ensureDirectoriesExist()
	
	// Create default CSS and JS files if they don't exist
	createDefaultFiles()
	
	// Configure HTTP routes
	http.HandleFunc("/", staticFileHandler)
	
	// Start P2P server in a separate goroutine
	go startP2PServer()
	
	// Start HTTP server
	log.Printf("HTTP server started on port %s", HTTPPort)
	log.Fatal(http.ListenAndServe(":"+HTTPPort, nil))
}
